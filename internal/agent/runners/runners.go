package runner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gohead-cms/gohead/internal/agent/functions"
	"github.com/gohead-cms/gohead/internal/agent/jobs"
	agentModels "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/llm"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/hibiken/asynq"
)

// AgentRunner no longer needs any LLM-related dependencies in its struct.
type AgentRunner struct{}

// NewAgentRunner is now much simpler.
func NewAgentRunner() *AgentRunner {
	return &AgentRunner{}
}

// HandleAgentJob is the entry point for processing jobs from the queue.
func (r *AgentRunner) HandleAgentJob(ctx context.Context, t *asynq.Task) error {
	var payload jobs.AgentJobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Log.WithError(err).Error("Failed to unmarshal agent job payload")
		return fmt.Errorf("could not unmarshal payload: %w", err)
	}

	logger.Log.WithField("agent_id", payload.AgentID).Info("Starting agent job execution")

	// FIX 1: Replaced GetAgentByUUID with GetAgentByID, which uses the correct uint ID.
	agent, err := storage.GetAgentByID(payload.AgentID)
	if err != nil {
		logger.Log.WithError(err).WithField("agent_id", payload.AgentID).Error("Failed to retrieve agent for job")
		return err
	}

	if err := r.runConversation(ctx, agent, payload); err != nil {
		logger.Log.WithError(err).WithField("agent_id", payload.AgentID).Error("Agent conversation loop failed")
		return err
	}

	logger.Log.WithField("agent_id", payload.AgentID).Info("Agent job completed successfully")
	return nil
}

// runConversation now uses your clean llm.Client interface.
func (r *AgentRunner) runConversation(ctx context.Context, agent *agentModels.Agent, payload jobs.AgentJobPayload) error {
	// FIX 2: Manually convert the agent's LLMConfig to the type expected by the llm package.
	llmConfigForAdapter := llm.Config{
		Provider:  agent.LLMConfig.Provider,
		Model:     agent.LLMConfig.Model,
		APIKey:    agent.LLMConfig.APIKey,
		APISecret: agent.LLMConfig.APISecret,
	}

	// 1. Create the LLM client using the converted config.
	llmClient, err := llm.NewAdapter(llmConfigForAdapter)
	if err != nil {
		return fmt.Errorf("could not create LLM adapter: %w", err)
	}

	// 2. Initialize the function registry.
	registry := functions.NewRegistry(agent.Functions)

	// 3. Prepare the message history using your internal llm.Message type.
	history, err := storage.GetConversationHistory(agent.ID)
	if err != nil {
		return fmt.Errorf("could not load conversation history: %w", err)
	}

	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: agent.SystemPrompt},
	}
	messages = append(messages, history...)
	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: payload.InitialInput})

	// 4. Start the execution loop.
	for i := 0; i < agent.MaxTurns; i++ {
		response, err := llmClient.Chat(ctx, messages, llm.WithTools(registry.Specs()))
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}

		if response.Type == llm.ResponseTypeToolCall {
			toolCall := response.ToolCall
			logger.Log.WithField("agent_id", agent.ID).Infof("LLM requested tool call: %s", toolCall.Name)

			messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: ""}) // Placeholder for assistant's turn

			fn, ok := registry.Get(toolCall.Name)
			if !ok {
				return fmt.Errorf("LLM requested unknown tool: %s", toolCall.Name)
			}

			result, err := fn(ctx, toolCall.Arguments)
			if err != nil {
				return fmt.Errorf("tool '%s' execution failed: %w", toolCall.Name, err)
			}

			messages = append(messages, llm.Message{
				Role:    llm.RoleTool,
				Content: result,
			})
			// Continue the loop

		} else {
			messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: response.Content})
			logger.Log.WithField("agent_id", agent.ID).Info("LLM finished with text response. Ending turn.")
			break // End the loop
		}
	}

	// 5. Save the updated conversation history.
	if err := storage.SaveConversationHistory(agent.ID, messages[1:]); err != nil {
		return fmt.Errorf("could not save conversation history: %w", err)
	}

	return nil
}
