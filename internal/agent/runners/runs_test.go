package runner

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/agent/jobs"
	"github.com/gohead-cms/gohead/internal/models"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/llm"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func init() {
	logger.InitLogger("error")
	logger.Log.SetOutput(&bytes.Buffer{})
}

// scripted is a fake LLM: it answers with the next scripted response and
// reports usage like the real adapter does.
type scripted struct {
	mu        sync.Mutex
	responses []llm.Response
	calls     int
	err       error
}

func (s *scripted) Chat(ctx context.Context, _ []llm.Message, _ ...llm.Option) (*llm.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if len(s.responses) == 0 {
		return &llm.Response{Type: llm.ResponseTypeText, Content: "done"}, nil
	}
	r := s.responses[0]
	s.responses = s.responses[1:]
	if rec := llm.RecorderFrom(ctx); rec != nil {
		rec.Add(r.Usage)
	}
	return &r, nil
}

func toolCall(id, name, args string, in, out int, cost float64) llm.Response {
	return llm.Response{
		Type:     llm.ResponseTypeToolCall,
		ToolCall: &llms.ToolCall{ID: id, Type: "function", FunctionCall: &llms.FunctionCall{Name: name, Arguments: args}},
		Usage:    llm.Usage{InputTokens: in, OutputTokens: out, CostUSD: cost, Calls: 1},
	}
}

func text(content string, in, out int, cost float64) llm.Response {
	return llm.Response{Type: llm.ResponseTypeText, Content: content, Usage: llm.Usage{InputTokens: in, OutputTokens: out, CostUSD: cost, Calls: 1}}
}

type fixture struct {
	t        *testing.T
	runner   *AgentRunner
	llm      *scripted
	articles models.Collection
	drafts   models.Collection
}

func setup(t *testing.T) *fixture {
	t.Helper()
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, migrations.MigrateDatabase(db))
	f := &fixture{t: t, llm: &scripted{}}
	f.articles = models.Collection{Name: "articles", Kind: "collection", Attributes: []models.Attribute{{Name: "title", Type: "string"}}}
	f.drafts = models.Collection{Name: "drafts", Kind: "collection", Attributes: []models.Attribute{{Name: "title", Type: "string"}}}
	require.NoError(t, storage.SaveCollection(&f.articles))
	require.NoError(t, storage.SaveCollection(&f.drafts))
	f.runner = &AgentRunner{newClient: func(llm.Config) (llm.Client, error) { return f.llm, nil }, now: time.Now}
	return f
}

func (f *fixture) agent(limits agents.Limits, fns ...agents.FunctionSpec) *agents.Agent {
	f.t.Helper()
	a := &agents.Agent{
		Name: "tagger", SystemPrompt: "You tag things.", MaxTurns: 4,
		LLMConfig: agents.LLMConfig{Provider: "openai", Model: "gpt-test", InputCostPerMillion: 1, OutputCostPerMillion: 2},
		Memory:    agents.MemoryConfig{Type: "in-memory", SessionScope: "agent"},
		Trigger:   agents.TriggerConfig{Type: "manual"},
		Functions: fns, Limits: limits,
	}
	require.NoError(f.t, storage.SaveAgent(a))
	return a
}

func (f *fixture) run(a *agents.Agent) (*agents.AgentRun, error) {
	f.t.Helper()
	err := f.runner.Run(context.Background(), jobs.AgentJobPayload{
		AgentID: a.ID,
		TriggerEvent: &jobs.TriggerEvent{Type: "collection_event", CollectionEvent: &jobs.CollectionEventData{
			Collection: "articles", Event: "item:created", ItemID: "1", ItemData: map[string]any{"title": "Hello"},
		}},
	}, 1)
	runs, _, lerr := storage.ListAgentRuns(a.ID, storage.RunQuery{PageSize: 1})
	require.NoError(f.t, lerr)
	require.NotEmpty(f.t, runs)
	full, gerr := storage.GetAgentRun(a.ID, runs[0].ID)
	require.NoError(f.t, gerr)
	return full, err
}

var upsert = agents.FunctionSpec{Name: "save", ImplKey: "collections.upsert_item", Parameters: models.JSONMap{"type": "object"}, Collections: []string{"articles"}}
var del = agents.FunctionSpec{Name: "remove", ImplKey: "collections.delete_item", Parameters: models.JSONMap{"type": "object"}, Collections: []string{"drafts"}}

func TestRunIsRecorded(t *testing.T) {
	f := setup(t)
	a := f.agent(agents.Limits{}, upsert)
	f.llm.responses = []llm.Response{
		toolCall("c1", "save", `{"collection_name":"articles","data":{"title":"Tagged"}}`, 100, 20, 0.0001),
		text("All done.", 150, 10, 0.00017),
	}

	run, err := f.run(a)
	require.NoError(t, err)
	assert.Equal(t, agents.RunSucceeded, run.Status)
	assert.Equal(t, "collection_event", run.TriggerType)
	assert.Contains(t, run.Trigger, `"Hello"`)
	assert.Contains(t, run.Input, "Event Type: item:created")
	assert.Equal(t, "All done.", run.Output)
	assert.Equal(t, 2, run.Turns)
	assert.Equal(t, 1, run.ToolCalls)
	assert.Equal(t, 250, run.InputTokens)
	assert.Equal(t, 30, run.OutputTokens)
	assert.InDelta(t, 0.00027, run.CostUSD, 1e-9)
	assert.NotNil(t, run.FinishedAt)
	assert.Equal(t, "openai", run.Provider)

	require.Len(t, run.Steps, 3)
	assert.Equal(t, agents.StepLLM, run.Steps[0].Type)
	assert.Contains(t, run.Steps[0].Result, "→ save(")
	assert.Equal(t, 100, run.Steps[0].InputTokens)
	tool := run.Steps[1]
	assert.Equal(t, agents.StepTool, tool.Type)
	assert.Equal(t, "save", tool.Name)
	assert.Equal(t, "collections.upsert_item", tool.ImplKey)
	assert.Contains(t, tool.Arguments, "Tagged")
	assert.Contains(t, tool.Result, `"created"`)
	assert.Empty(t, tool.Error)
	assert.Equal(t, "All done.", run.Steps[2].Result)

	items, total, _ := storage.GetItems(f.articles.ID, 1, 10)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Tagged", items[0].Data["title"])
}

func TestFunctionScopes(t *testing.T) {
	f := setup(t)
	a := f.agent(agents.Limits{}, upsert, del)
	article, err := storage.SaveItem(storage.SystemActor, f.articles, models.JSONMap{"title": "Keep me"})
	require.NoError(t, err)
	draft, err := storage.SaveItem(storage.SystemActor, f.drafts, models.JSONMap{"title": "Old draft"})
	require.NoError(t, err)

	f.llm.responses = []llm.Response{
		toolCall("c1", "remove", `{"item_id":"`+itoa(article.ID)+`"}`, 10, 5, 0),                            // out of scope
		toolCall("c2", "save", `{"collection_name":"drafts","data":{"title":"nope"}}`, 10, 5, 0),            // out of scope
		toolCall("c3", "remove", `{"item_id":"`+itoa(draft.ID)+`","collection_name":"articles"}`, 10, 5, 0), // wrong collection
		toolCall("c4", "remove", `{"item_id":"`+itoa(draft.ID)+`"}`, 10, 5, 0),                              // allowed
	}
	a.MaxTurns = 5
	require.NoError(t, storage.UpdateAgent(a.ID, a))

	run, err := f.run(a)
	require.NoError(t, err)
	tools := []agents.AgentRunStep{}
	for _, s := range run.Steps {
		if s.Type == agents.StepTool {
			tools = append(tools, s)
		}
	}
	require.Len(t, tools, 4)
	assert.Contains(t, tools[0].Result, "not allowed to access collection 'articles'")
	assert.NotEmpty(t, tools[0].Error, "refusals are flagged in the run")
	assert.Contains(t, tools[1].Result, "not allowed to access collection 'drafts'")
	assert.Contains(t, tools[2].Result, "is not in collection 'articles'")
	assert.Contains(t, tools[3].Result, "successfully deleted")

	_, err = storage.GetItemByID(f.articles.ID, article.ID)
	assert.NoError(t, err, "the article survived")
	_, err = storage.GetItemByID(f.drafts.ID, draft.ID)
	assert.Error(t, err, "the draft was deleted")
	_, total, _ := storage.GetItems(f.drafts.ID, 1, 10)
	assert.Equal(t, 0, total, "nothing was written to drafts")
}

func TestRunLimits(t *testing.T) {
	t.Run("runs per hour", func(t *testing.T) {
		f := setup(t)
		a := f.agent(agents.Limits{MaxRunsPerHour: 1})
		f.llm.responses = []llm.Response{text("one", 10, 1, 0)}
		run, err := f.run(a)
		require.NoError(t, err)
		assert.Equal(t, agents.RunSucceeded, run.Status)

		run, err = f.run(a)
		require.NoError(t, err, "limited runs are not retried")
		assert.Equal(t, agents.RunLimited, run.Status)
		assert.Contains(t, run.Error, "max_runs_per_hour")
		assert.Empty(t, run.Steps)
		assert.Equal(t, 1, f.llm.calls, "the model was not called")
	})

	t.Run("tokens per run stop a run", func(t *testing.T) {
		f := setup(t)
		a := f.agent(agents.Limits{MaxTokensPerRun: 150}, upsert)
		f.llm.responses = []llm.Response{
			toolCall("c1", "save", `{"collection_name":"articles","data":{"title":"a"}}`, 80, 10, 0),
			toolCall("c2", "save", `{"collection_name":"articles","data":{"title":"b"}}`, 80, 10, 0),
			text("never reached", 10, 1, 0),
		}
		run, err := f.run(a)
		require.NoError(t, err)
		assert.Equal(t, agents.RunLimited, run.Status)
		assert.Contains(t, run.Error, "max_tokens_per_run")
		assert.Equal(t, 180, run.InputTokens+run.OutputTokens)
		assert.Equal(t, 2, f.llm.calls)
		_, total, _ := storage.GetItems(f.articles.ID, 1, 10)
		assert.Equal(t, 1, total, "the second tool call was not executed")
	})

	t.Run("daily cost spans runs", func(t *testing.T) {
		f := setup(t)
		a := f.agent(agents.Limits{MaxCostPerDay: 0.01}, upsert)
		f.llm.responses = []llm.Response{text("first", 1000, 100, 0.008)}
		run, _ := f.run(a)
		assert.Equal(t, agents.RunSucceeded, run.Status)

		// The second run goes over the limit with a function call: it stops
		// before running the function.
		f.llm.responses = []llm.Response{
			toolCall("c1", "save", `{"collection_name":"articles","data":{"title":"x"}}`, 1000, 100, 0.008),
		}
		run, _ = f.run(a)
		assert.Equal(t, agents.RunLimited, run.Status, "stopped once the day's cost exceeds the limit")
		assert.Contains(t, run.Error, "max_cost_per_day")
		_, total, _ := storage.GetItems(f.articles.ID, 1, 10)
		assert.Zero(t, total, "the function was not called")

		run, _ = f.run(a)
		assert.Equal(t, agents.RunLimited, run.Status)
		assert.Empty(t, run.Steps, "later runs do not start")
		assert.Equal(t, 2, f.llm.calls)
	})

	t.Run("a final answer over the limit still completes the run", func(t *testing.T) {
		f := setup(t)
		a := f.agent(agents.Limits{MaxTokensPerRun: 50})
		f.llm.responses = []llm.Response{text("done", 100, 10, 0)}
		run, err := f.run(a)
		require.NoError(t, err)
		assert.Equal(t, agents.RunSucceeded, run.Status)
		assert.Equal(t, "done", run.Output)
		history, _ := storage.GetConversationHistory(a.ID)
		assert.Len(t, history, 2, "input and answer are kept")
	})

	t.Run("runs are counted in creation order", func(t *testing.T) {
		f := setup(t)
		a := f.agent(agents.Limits{MaxRunsPerHour: 1})
		// Two runs that started together: the earlier one is still running.
		first := &agents.AgentRun{AgentID: a.ID, Status: agents.RunRunning, StartedAt: time.Now().UTC()}
		require.NoError(t, storage.CreateAgentRun(first))
		run, err := f.run(a)
		require.NoError(t, err)
		assert.Equal(t, agents.RunLimited, run.Status, "a run in progress counts")
		assert.Zero(t, f.llm.calls)
	})
}

func TestFailedRun(t *testing.T) {
	f := setup(t)
	a := f.agent(agents.Limits{})
	f.llm.err = errors.New("provider unavailable")
	run, err := f.run(a)
	require.Error(t, err, "failures are retried by the queue")
	assert.Equal(t, agents.RunFailed, run.Status)
	assert.Contains(t, run.Error, "provider unavailable")
	require.Len(t, run.Steps, 1)
	assert.Contains(t, run.Steps[0].Error, "provider unavailable")

	history, err := storage.GetConversationHistory(a.ID)
	require.NoError(t, err)
	assert.Empty(t, history, "a failed run leaves the history to its retry")
}

func TestNestedRelationsRespectScopes(t *testing.T) {
	f := setup(t)
	posts := models.Collection{Name: "posts", Kind: "collection", Attributes: []models.Attribute{
		{Name: "title", Type: "string"},
		{Name: "draft", Type: "relation", Target: "drafts", Relation: "oneToOne"},
	}}
	require.NoError(t, storage.SaveCollection(&posts))
	draft, err := storage.SaveItem(storage.SystemActor, f.drafts, models.JSONMap{"title": "Existing"})
	require.NoError(t, err)

	savePost := agents.FunctionSpec{Name: "save_post", ImplKey: "collections.upsert_item", Parameters: models.JSONMap{"type": "object"}, Collections: []string{"posts"}}
	a := f.agent(agents.Limits{}, savePost)
	f.llm.responses = []llm.Response{
		toolCall("c1", "save_post", `{"collection_name":"posts","data":{"title":"a","draft":{"title":"created in drafts"}}}`, 10, 5, 0),
		toolCall("c2", "save_post", `{"collection_name":"posts","data":{"title":"b","draft":{"id":`+itoa(draft.ID)+`}}}`, 10, 5, 0),
		text("ok", 10, 5, 0),
	}
	run, err := f.run(a)
	require.NoError(t, err)
	var tools []agents.AgentRunStep
	for _, s := range run.Steps {
		if s.Type == agents.StepTool {
			tools = append(tools, s)
		}
	}
	require.Len(t, tools, 2)
	assert.Contains(t, tools[0].Result, "not allowed to access collection 'drafts'", "nested creates need the target in scope")
	assert.Contains(t, tools[1].Result, `"created"`, "connecting an existing item by ID is allowed")

	_, total, _ := storage.GetItems(f.drafts.ID, 1, 10)
	assert.Equal(t, 1, total, "nothing was created in drafts")
	_, total, _ = storage.GetItems(posts.ID, 1, 10)
	assert.Equal(t, 1, total)
}

func TestPruning(t *testing.T) {
	f := setup(t)
	a := f.agent(agents.Limits{})
	prev := RunsKept
	RunsKept = 3
	defer func() { RunsKept = prev }()
	for i := 0; i < 5; i++ {
		f.llm.responses = []llm.Response{text("ok", 1, 1, 0)}
		_, _ = f.run(a)
	}
	runs, total, err := storage.ListAgentRuns(a.ID, storage.RunQuery{PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, runs, 3)
	var orphans int64
	database.DB.Model(&agents.AgentRunStep{}).Where("run_id NOT IN (?)", database.DB.Model(&agents.AgentRun{}).Select("id")).Count(&orphans)
	assert.Zero(t, orphans, "steps of pruned runs are removed")
}

func itoa(id uint) string {
	b := []byte{}
	for id > 0 || len(b) == 0 {
		b = append([]byte{byte('0' + id%10)}, b...)
		id /= 10
	}
	return string(b)
}
