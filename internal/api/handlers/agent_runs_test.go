package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRunEndpoints(t *testing.T) {
	a := newContentAPI(t)
	admin := a.router.Group("/admin/agents", middleware.AuthMiddleware(), middleware.AdminOnly())
	admin.POST("", CreateAgent)
	admin.GET("/:name/runs", ListAgentRuns)
	admin.GET("/:name/runs/:run", GetAgentRun)
	admin.GET("/:name/usage", GetAgentUsage)

	a.ok("admin", "POST", "/admin/agents", map[string]any{
		"name": "tagger", "system_prompt": "Tag articles.", "max_turns": 3,
		"llm_config": map[string]any{"provider": "openai", "model": "gpt-test", "input_cost_per_million": 1.5},
		"memory":     map[string]any{"type": "in-memory", "session_scope": "agent"},
		"trigger":    map[string]any{"type": "manual"},
		"limits":     map[string]any{"max_runs_per_hour": 30, "max_cost_per_day": 2},
		"functions": []any{map[string]any{
			"name": "remove", "impl_key": "collections.delete_item", "parameters": map[string]any{}, "collections": []any{"drafts"},
		}},
	})
	code, out := a.do("admin", "POST", "/admin/agents", map[string]any{
		"name": "bad", "system_prompt": "x", "max_turns": 1,
		"llm_config": map[string]any{"provider": "openai"}, "memory": map[string]any{"type": "in-memory", "session_scope": "agent"},
		"trigger": map[string]any{"type": "manual"}, "limits": map[string]any{"max_cost_per_day": 1},
	})
	assert.Equal(t, http.StatusBadRequest, code, "%v", out)

	agent, err := storage.GetAgentByName("tagger")
	require.NoError(t, err)
	assert.Equal(t, 30, agent.Limits.MaxRunsPerHour, "limits are stored")
	assert.Equal(t, []string{"drafts"}, agent.Functions[0].Collections, "function scopes are stored")

	now := time.Now().UTC()
	for i, status := range []string{agents.RunSucceeded, agents.RunFailed, agents.RunLimited} {
		run := &agents.AgentRun{AgentID: agent.ID, AgentName: "tagger", Status: status, StartedAt: now.Add(-time.Duration(i) * time.Minute),
			InputTokens: 100, OutputTokens: 10, CostUSD: 0.01}
		require.NoError(t, storage.CreateAgentRun(run))
		storage.AddAgentRunStep(&agents.AgentRunStep{RunID: run.ID, Seq: 1, Type: agents.StepLLM, Name: "openai/gpt-test"})
		storage.AddAgentRunStep(&agents.AgentRunStep{RunID: run.ID, Seq: 2, Type: agents.StepTool, Name: "remove", Result: "{}"})
	}

	runs := a.ok("admin", "GET", "/admin/agents/tagger/runs", nil).([]any)
	require.Len(t, runs, 3)
	assert.NotContains(t, runs[0].(map[string]any), "steps", "lists are light")
	failed := a.ok("admin", "GET", "/admin/agents/tagger/runs?status=failed", nil).([]any)
	require.Len(t, failed, 1)

	id := int(failed[0].(map[string]any)["id"].(float64))
	detail := a.ok("admin", "GET", fmt.Sprintf("/admin/agents/tagger/runs/%d", id), nil).(map[string]any)
	steps := detail["steps"].([]any)
	require.Len(t, steps, 2)
	assert.Equal(t, "llm", steps[0].(map[string]any)["type"])
	assert.Equal(t, "remove", steps[1].(map[string]any)["name"])

	usage := a.ok("admin", "GET", "/admin/agents/tagger/usage", nil).(map[string]any)
	day := usage["lastDay"].(map[string]any)
	assert.EqualValues(t, 2, day["runs"], "limited runs do not count as runs")
	assert.EqualValues(t, 300, day["inputTokens"], "but their tokens do")
	assert.Equal(t, true, usage["costKnown"])

	code, _ = a.do("admin", "GET", "/admin/agents/tagger/runs?status=weird", nil)
	assert.Equal(t, http.StatusBadRequest, code)
	code, _ = a.do("admin", "GET", "/admin/agents/nobody/runs", nil)
	assert.Equal(t, http.StatusNotFound, code)
	code, _ = a.do("admin", "GET", "/admin/agents/tagger/runs/999999", nil)
	assert.Equal(t, http.StatusNotFound, code)
	code, _ = a.do("editor", "GET", "/admin/agents/tagger/runs", nil)
	assert.Equal(t, http.StatusForbidden, code)

	// Deleting the agent deletes its runs and steps.
	require.NoError(t, storage.DeleteAgent(agent.ID))
	left, _, err := storage.ListAgentRuns(agent.ID, storage.RunQuery{})
	require.NoError(t, err)
	assert.Empty(t, left)
	_, err = storage.GetAgentRun(agent.ID, uint(id))
	assert.ErrorIs(t, err, storage.ErrRunNotFound)
}
