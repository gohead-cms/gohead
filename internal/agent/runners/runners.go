package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/gohead-cms/gohead/internal/agent/functions"
	"github.com/gohead-cms/gohead/internal/agent/jobs"
	"github.com/gohead-cms/gohead/internal/models"
	agentModels "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/llm"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/tmc/langchaingo/llms"
)

// RunsKept is how many runs are kept per agent (set from agents.runs_kept).
var RunsKept = 200

// maxStoredText bounds the text stored per field of a run or step.
const maxStoredText = 16 << 10

// errLimited stops a run because a limit was reached; it is not retried.
type errLimited struct{ reason string }

func (e *errLimited) Error() string { return e.reason }

// AgentRunner executes agentic workflows.
type AgentRunner struct {
	// newClient builds the LLM client; tests replace it.
	newClient func(llm.Config) (llm.Client, error)
	now       func() time.Time
}

// NewAgentRunner creates a new, clean instance of the AgentRunner.
func NewAgentRunner() *AgentRunner {
	return &AgentRunner{newClient: llm.NewAdapter, now: time.Now}
}

// HandleAgentJob is the entry point for processing jobs from the 'agents' queue.
func (r *AgentRunner) HandleAgentJob(ctx context.Context, data []byte) error {
	var payload jobs.AgentJobPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		logger.Log.WithError(err).Error("Failed to unmarshal agent job payload")
		return fmt.Errorf("could not unmarshal payload: %w", err)
	}
	attempt, _ := queue.GetRetryCount(ctx)
	return r.Run(ctx, payload, attempt+1)
}

// Run executes one job and records it as a run. It returns an error only
// when the job should be retried.
func (r *AgentRunner) Run(ctx context.Context, payload jobs.AgentJobPayload, attempt int) error {
	logger.Log.WithField("agent_id", payload.AgentID).Info("Starting agent job execution")

	// Runs requested from the API (Run now, test runs) were recorded as
	// queued: they run once under that ID and are not retried, so the run
	// the caller follows is the one that happens.
	manual := payload.RunID != 0
	agent, err := storage.GetAgentByID(payload.AgentID)
	if err != nil {
		logger.Log.WithError(err).WithField("agent_id", payload.AgentID).Error("Failed to retrieve agent for job")
		if manual {
			storage.FailQueuedRun(payload.RunID, "the agent could not be loaded: "+err.Error())
			return nil
		}
		return err
	}

	run := &agentModels.AgentRun{
		AgentID:   agent.ID,
		AgentName: agent.Name,
		Status:    agentModels.RunRunning,
		Attempt:   attempt,
		Provider:  agent.LLMConfig.Provider,
		Model:     agent.LLMConfig.Model,
		StartedAt: r.now().UTC(),
		DryRun:    payload.DryRun,
	}
	run.TriggerType, run.Trigger = describeTrigger(payload)
	run.Input = truncate(r.createContextualInput(payload), maxStoredText)

	// Record the run first, then check the limits against the runs created
	// before it: runs of the same agent starting together on several
	// workers are ordered by ID and cannot all pass the same check. A run
	// requested from the API was recorded (queued) already: it goes on
	// under the same ID.
	if manual {
		run.ID = payload.RunID
		if err := storage.SaveAgentRun(run); err != nil {
			logger.Log.WithError(err).WithField("run_id", run.ID).Error("Could not start a queued run")
			storage.FailQueuedRun(payload.RunID, "could not be started: "+err.Error())
			return nil
		}
	} else if err := storage.CreateAgentRun(run); err != nil {
		return fmt.Errorf("could not record run: %w", err)
	}
	reason, err := preRunLimit(agent, run.ID, manual, r.now)
	if err != nil {
		r.finish(run, agentModels.RunFailed, "could not check limits: "+err.Error(), nil)
		r.save(run)
		if manual {
			return nil
		}
		return fmt.Errorf("could not check agent usage: %w", err)
	}
	if reason != "" {
		r.finish(run, agentModels.RunLimited, reason, nil)
		r.save(run)
		logger.Log.WithField("agent", agent.Name).WithField("reason", reason).Warn("Agent run skipped: limit reached")
		storage.PruneAgentRuns(agent.ID, RunsKept)
		return nil
	}

	rec := &llm.UsageRecorder{}
	runErr := r.runConversation(llm.WithUsageRecorder(ctx, rec), agent, payload, run, rec)

	var limited *errLimited
	switch {
	case errors.As(runErr, &limited):
		r.finish(run, agentModels.RunLimited, limited.reason, rec)
		runErr = nil // do not retry
	case runErr != nil:
		r.finish(run, agentModels.RunFailed, runErr.Error(), rec)
	default:
		r.finish(run, agentModels.RunSucceeded, run.Error, rec)
	}
	r.save(run)
	storage.PruneAgentRuns(agent.ID, RunsKept)

	if runErr != nil {
		logger.Log.WithError(runErr).WithField("agent_id", payload.AgentID).Error("Agent conversation loop failed")
		if manual {
			return nil
		}
		return runErr
	}
	logger.Log.WithField("agent_id", payload.AgentID).WithField("run_id", run.ID).WithField("status", run.Status).Info("Agent job completed")
	return nil
}

func (r *AgentRunner) save(run *agentModels.AgentRun) {
	if err := storage.SaveAgentRun(run); err != nil {
		logger.Log.WithError(err).Warn("Failed to save agent run")
	}
}

func (r *AgentRunner) finish(run *agentModels.AgentRun, status, errText string, rec *llm.UsageRecorder) {
	now := r.now().UTC()
	run.Status = status
	run.Error = truncate(errText, maxStoredText)
	run.FinishedAt = &now
	run.DurationMs = now.Sub(run.StartedAt).Milliseconds()
	if rec != nil {
		u := rec.Total()
		run.InputTokens, run.OutputTokens, run.CostUSD = u.InputTokens, u.OutputTokens, u.CostUSD
	}
}

// preRunLimit returns why run runID of the agent may not start ("" if it
// may), counting only the runs created before it. A database error fails
// closed.
// A queued run (wasQueued) was numbered when it was requested: it also
// counts the runs started since, whatever their ID.
func preRunLimit(agent *agentModels.Agent, runID uint, wasQueued bool, now func() time.Time) (string, error) {
	l := agent.Limits
	if !l.Any() {
		return "", nil
	}
	day, err := storage.AgentUsageBefore(agent.ID, now().UTC().Add(-24*time.Hour), runID, wasQueued)
	if err != nil {
		return "", err
	}
	if l.MaxRunsPerDay > 0 && day.Runs >= l.MaxRunsPerDay {
		return fmt.Sprintf("max_runs_per_day reached (%d runs in the last 24 hours)", day.Runs), nil
	}
	if l.MaxRunsPerHour > 0 {
		hour, err := storage.AgentUsageBefore(agent.ID, now().UTC().Add(-time.Hour), runID, wasQueued)
		if err != nil {
			return "", err
		}
		if hour.Runs >= l.MaxRunsPerHour {
			return fmt.Sprintf("max_runs_per_hour reached (%d runs in the last hour)", hour.Runs), nil
		}
	}
	if l.MaxTokensPerDay > 0 && day.Tokens() >= l.MaxTokensPerDay {
		return fmt.Sprintf("max_tokens_per_day reached (%d tokens in the last 24 hours)", day.Tokens()), nil
	}
	if l.MaxCostPerDay > 0 && day.CostUSD >= l.MaxCostPerDay {
		return fmt.Sprintf("max_cost_per_day reached ($%.4f in the last 24 hours)", day.CostUSD), nil
	}
	return "", nil
}

// inRunLimit returns why a running run must stop ("" to continue). Daily
// limits are checked against the live usage of all the agent's other runs,
// including runs in progress on other workers.
func (r *AgentRunner) inRunLimit(agent *agentModels.Agent, run *agentModels.AgentRun, used llm.Usage) string {
	l := agent.Limits
	if l.MaxTokensPerRun > 0 && used.Total() > l.MaxTokensPerRun {
		return fmt.Sprintf("max_tokens_per_run reached (%d tokens)", used.Total())
	}
	if l.MaxTokensPerDay == 0 && l.MaxCostPerDay == 0 {
		return ""
	}
	day, err := storage.AgentUsageSince(agent.ID, r.now().UTC().Add(-24*time.Hour), run.ID)
	if err != nil {
		return "could not check daily limits: " + err.Error()
	}
	if l.MaxTokensPerDay > 0 && day.Tokens()+used.Total() > l.MaxTokensPerDay {
		return fmt.Sprintf("max_tokens_per_day reached (%d tokens in the last 24 hours)", day.Tokens()+used.Total())
	}
	if l.MaxCostPerDay > 0 && day.CostUSD+used.CostUSD > l.MaxCostPerDay {
		return fmt.Sprintf("max_cost_per_day reached ($%.4f in the last 24 hours)", day.CostUSD+used.CostUSD)
	}
	return ""
}

// runConversation contains the main agent loop: LLM calls, function
// execution, history management, step recording and in-run limits.
func (r *AgentRunner) runConversation(ctx context.Context, agent *agentModels.Agent, payload jobs.AgentJobPayload,
	run *agentModels.AgentRun, rec *llm.UsageRecorder) (runErr error) {
	// Content the agent's functions change is attributed to this run, so
	// it shows in the history and the run can be reverted.
	ctx = storage.WithActor(ctx, storage.Actor{Kind: models.ActorAgent, Name: agent.Name, AgentRunID: run.ID})
	switch {
	case payload.DryRun:
		ctx = functions.WithWriteMode(ctx, functions.WriteDryRun)
	case agent.RequireApproval:
		ctx = functions.WithWriteMode(ctx, functions.WritePropose)
	}
	llmClient, err := r.newClient(llm.Config{
		Provider:             agent.LLMConfig.Provider,
		Model:                agent.LLMConfig.Model,
		APIKey:               agent.LLMConfig.APIKey,
		APISecret:            agent.LLMConfig.APISecret,
		BaseURL:              agent.LLMConfig.BaseURL,
		InputCostPerMillion:  agent.LLMConfig.InputCostPerMillion,
		OutputCostPerMillion: agent.LLMConfig.OutputCostPerMillion,
	})
	if err != nil {
		return fmt.Errorf("could not create LLM adapter: %w", err)
	}

	registry := functions.NewRegistry(agent.Functions)
	langchainTools := registry.ToLangchainTools()

	history, err := storage.GetConversationHistory(agent.ID)
	if err != nil {
		return fmt.Errorf("could not load conversation history: %w", err)
	}
	history = trimHistory(history, maxHistoryMessages)
	messages := []llm.Message{{Role: llm.RoleSystem, Content: agent.SystemPrompt}}
	messages = append(messages, history...)
	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: r.createContextualInput(payload)})

	seq := 0
	step := func(s *agentModels.AgentRunStep) {
		seq++
		s.RunID, s.Seq = run.ID, seq
		storage.AddAgentRunStep(s)
	}
	// saveProgress stores the live usage, so concurrent runs and the admin
	// UI see it before the run ends.
	saveProgress := func() {
		u := rec.Total()
		run.InputTokens, run.OutputTokens, run.CostUSD = u.InputTokens, u.OutputTokens, u.CostUSD
		if err := storage.SaveAgentRun(run); err != nil {
			logger.Log.WithError(err).Warn("Failed to save run progress")
		}
	}
	defer func() {
		// Keep the conversation when the run ends or a limit stops it, but
		// not when it fails: the job is retried, and the retry starts again
		// from the same history and input.
		var limited *errLimited
		if runErr != nil && !errors.As(runErr, &limited) {
			return
		}
		if payload.DryRun {
			return // test runs leave the agent's memory as it was
		}
		if err := storage.SaveConversationHistory(agent.ID, messages[1:]); err != nil {
			logger.Log.WithError(err).Warn("Could not save conversation history")
		}
	}()

	finished := false
	for i := 0; i < agent.MaxTurns; i++ {
		run.Turns = i + 1
		start := r.now()
		response, err := llmClient.Chat(ctx, messages, llm.WithTools(langchainTools))
		llmStep := &agentModels.AgentRunStep{Type: agentModels.StepLLM, Name: modelName(agent), DurationMs: r.now().Sub(start).Milliseconds()}
		if err != nil {
			llmStep.Error = truncate(err.Error(), maxStoredText)
			step(llmStep)
			return fmt.Errorf("LLM call failed on turn %d: %w", i+1, err)
		}
		llmStep.InputTokens, llmStep.OutputTokens, llmStep.CostUSD = response.Usage.InputTokens, response.Usage.OutputTokens, response.Usage.CostUSD
		if response.Type == llm.ResponseTypeToolCall {
			llmStep.Result = truncate("→ "+response.ToolCall.FunctionCall.Name+"("+response.ToolCall.FunctionCall.Arguments+")", maxStoredText)
		} else {
			llmStep.Result = truncate(response.Content, maxStoredText)
		}
		step(llmStep)
		saveProgress()

		// A final answer completes the run, even if it went over a limit:
		// the work is done and the next run will be refused.
		if response.Type != llm.ResponseTypeToolCall {
			messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: response.Content})
			run.Output = truncate(response.Content, maxStoredText)
			finished = true
			break
		}
		if reason := r.inRunLimit(agent, run, rec.Total()); reason != "" {
			return &errLimited{reason: reason}
		}

		toolCall := response.ToolCall
		messages = append(messages, llm.Message{Role: llm.RoleAssistant, ToolCall: toolCall})
		run.ToolCalls++
		toolStep := &agentModels.AgentRunStep{
			Type: agentModels.StepTool, Name: toolCall.FunctionCall.Name,
			ImplKey: registry.ImplKey(toolCall.FunctionCall.Name), Arguments: truncate(toolCall.FunctionCall.Arguments, maxStoredText),
		}
		result := r.callTool(ctx, registry, toolCall, toolStep, rec)
		messages = append(messages, llm.Message{Role: llm.RoleTool, ToolCallID: toolCall.ID, Content: result})
		step(toolStep)
		saveProgress()
		if reason := r.inRunLimit(agent, run, rec.Total()); reason != "" {
			return &errLimited{reason: reason}
		}
	}
	if !finished {
		run.Error = fmt.Sprintf("stopped after max_turns (%d) without a final answer", agent.MaxTurns)
	}
	return nil
}

// callTool runs one function and fills its step; the returned text goes
// back to the model (errors included, so it can recover).
func (r *AgentRunner) callTool(ctx context.Context, registry *functions.Registry, call *llms.ToolCall, s *agentModels.AgentRunStep, rec *llm.UsageRecorder) string {
	before := rec.Total()
	start := r.now()
	defer func() {
		s.DurationMs = r.now().Sub(start).Milliseconds()
		used := rec.Total()
		s.InputTokens = used.InputTokens - before.InputTokens // LLM calls made by the function itself
		s.OutputTokens = used.OutputTokens - before.OutputTokens
		s.CostUSD = used.CostUSD - before.CostUSD
	}()

	fn, ok := registry.Get(call.FunctionCall.Name)
	if !ok {
		s.Error = "tool not found"
		out, _ := json.Marshal(map[string]string{"error": "Tool not found", "requested_tool": call.FunctionCall.Name})
		s.Result = string(out)
		return string(out)
	}
	result, err := fn(ctx, call.FunctionCall.Arguments)
	if err != nil {
		s.Error = truncate(err.Error(), maxStoredText)
		out, _ := json.Marshal(map[string]string{"error": "Tool execution failed", "message": err.Error()})
		s.Result = string(out)
		return string(out)
	}
	s.Result = truncate(result, maxStoredText)
	if isErrorResult(result) {
		s.Error = "the function reported an error"
	}
	return result
}

// isErrorResult recognizes the {"status": "error", ...} results built-in
// functions return to the model.
func isErrorResult(result string) bool {
	var v struct {
		Status string `json:"status"`
	}
	return json.Unmarshal([]byte(result), &v) == nil && v.Status == "error"
}

func modelName(agent *agentModels.Agent) string {
	if agent.LLMConfig.Model != "" {
		return agent.LLMConfig.Provider + "/" + agent.LLMConfig.Model
	}
	return agent.LLMConfig.Provider
}

// describeTrigger returns the trigger type and payload of a job.
func describeTrigger(p jobs.AgentJobPayload) (string, string) {
	typ := "manual"
	var body any = map[string]any{"input": p.InitialInput}
	if p.TriggerEvent != nil {
		typ = p.TriggerEvent.Type
		body = p.TriggerEvent
	}
	b, _ := json.Marshal(body)
	return typ, truncate(string(b), maxStoredText)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…[truncated]"
}

// createContextualInput creates a detailed initial prompt for the LLM based on the job's trigger.
func (r *AgentRunner) createContextualInput(payload jobs.AgentJobPayload) string {
	if payload.TriggerEvent != nil {
		switch payload.TriggerEvent.Type {
		case "collection_event":
			if payload.TriggerEvent.CollectionEvent != nil {
				event := payload.TriggerEvent.CollectionEvent
				dataJSON, _ := json.Marshal(event.ItemData)
				return fmt.Sprintf(
					"An event has occurred. Here is the data:\n\n"+
						"Event Type: %s\n"+
						"Collection: %s\n"+
						"Item ID: %s\n\n"+
						"Item Data:\n"+
						"```json\n%s\n```\n\n"+
						"Based on your instructions, you must now call the appropriate function to handle this event.",
					event.Event,
					event.Collection,
					event.ItemID,
					string(dataJSON),
				)
			}
		case "webhook":
			if payload.TriggerEvent.WebhookData != nil {
				dataJSON, _ := json.Marshal(payload.TriggerEvent.WebhookData)
				return fmt.Sprintf(
					"A webhook has been received. Here is the payload:\n\n"+
						"```json\n%s\n```\n\n"+
						"Based on your instructions, you must now call the appropriate function to process this webhook.",
					string(dataJSON),
				)
			}
		case "schedule":
			return payload.InitialInput
		}
	}

	if payload.InitialInput != "" {
		return payload.InitialInput
	}

	return "You have been activated. Please execute your task according to your system prompt."
}

// maxHistoryMessages bounds how much past conversation is replayed to the LLM,
// so that event-driven agents do not grow their context without limit.
const maxHistoryMessages = 40

// trimHistory keeps at most max recent messages and starts the window on a
// user message, so a tool result is never separated from the assistant
// message that requested it.
func trimHistory(history []llm.Message, max int) []llm.Message {
	if len(history) <= max {
		return history
	}
	h := history[len(history)-max:]
	for i, m := range h {
		if m.Role == llm.RoleUser {
			return h[i:]
		}
	}
	return nil
}
