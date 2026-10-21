package models

import (
	"bytes"
	"maps"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Initialize logger for testing
func init() {
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestParseAgentInput(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		hasError bool
	}{
		{
			name: "Valid Agent Input",
			input: map[string]any{
				"name":          "sales_chatbot",
				"system_prompt": "You are a friendly sales chatbot.",
				"max_turns":     5,
				"llm_config": map[string]any{
					"provider": "openai",
					"model":    "gpt-4",
				},
				"memory": map[string]any{
					"type":          "in-memory",
					"session_scope": "conversation",
				},
				"trigger": map[string]any{
					"type": "manual",
				},
				"functions": []any{
					map[string]any{
						"name":        "search_product",
						"description": "Searches for a product.",
						"impl_key":    "search_product_impl",
						"parameters": map[string]any{
							"type": "object",
						},
					},
				},
			},
			hasError: false,
		},
		{
			name: "Invalid Agent Input - Missing Name",
			input: map[string]any{
				"system_prompt": "You are a chatbot.",
			},
			hasError: true,
		},
		{
			name: "Invalid Agent Input - MaxTurns is not a number",
			input: map[string]any{
				"name":          "test_agent",
				"system_prompt": "...",
				"max_turns":     "five",
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAgentInput(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAgentSchema(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		hasError bool
		errMsg   string
	}{
		{
			name: "Valid Schema",
			agent: Agent{
				Name:         "test_agent",
				SystemPrompt: "A helpful assistant.",
				MaxTurns:     4,
				LLMConfig:    LLMConfig{Provider: "google", Model: "gemini-pro"},
				Memory:       MemoryConfig{Type: "in-memory", SessionScope: "user"},
				Trigger:      TriggerConfig{Type: "manual"},
			},
			hasError: false,
		},
		{
			name:     "Missing Name",
			agent:    Agent{SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}},
			hasError: true,
			errMsg:   "agent must have a 'name'",
		},
		{
			name:     "Missing System Prompt",
			agent:    Agent{Name: "test_agent", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}},
			hasError: true,
			errMsg:   "agent must have a 'system_prompt'",
		},
		{
			name:     "MaxTurns is 0",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 0, LLMConfig: LLMConfig{Provider: "p"}},
			hasError: true,
			errMsg:   "max_turns must be a positive integer",
		},
		{
			name:     "Missing LLM Provider",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1},
			hasError: true,
			errMsg:   "llm_config provider cannot be empty",
		},
		{
			name:     "Missing Memory Type",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}},
			hasError: true,
			errMsg:   "memory type cannot be empty",
		},
		{
			name:     "Missing Memory Session Scope",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory"}},
			hasError: true,
			errMsg:   "memory session scope cannot be empty",
		},
		{
			name:     "Invalid Trigger Type",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "invalid"}},
			hasError: true,
			errMsg:   "invalid trigger type: must be 'manual', 'cron', 'webhook' or 'collection_event'",
		},
		{
			name:     "Cron Trigger with Empty Cron Expression",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "cron"}},
			hasError: true,
			errMsg:   "cron trigger requires a cron expression",
		},
		{
			name:     "Cron Trigger with Invalid Cron Expression",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "cron", Cron: "invalid"}},
			hasError: true,
			errMsg:   "invalid cron expression",
		},
		{
			name:     "Webhook Trigger with Empty Token",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "webhook"}},
			hasError: true,
			errMsg:   "webhook trigger requires a 'webhook_token'",
		},
		{
			name:     "Tool with an invalid name",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "manual"}, Functions: FunctionSpecs{{Name: "save summary", ImplKey: "collections.patch_item"}}},
			hasError: true,
			errMsg:   "names may only use letters",
		},
		{
			name:     "Function with Empty ImplKey",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "manual"}, Functions: FunctionSpecs{{Name: "test_func"}}},
			hasError: true,
			errMsg:   "say which tool with 'use'",
		},
		{
			name:     "Function with Invalid Parameters JSON",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "manual"}, Functions: FunctionSpecs{{Name: "test_func", ImplKey: "test_impl", Parameters: models.JSONMap{"invalid": make(chan int)}}}},
			hasError: true,
			errMsg:   "has invalid parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentSchema(tt.agent)
			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFunctionSpecsScan(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected FunctionSpecs
		hasError bool
	}{
		{
			name:  "Valid JSON Array",
			input: []byte(`[{"name": "func1", "impl_key": "impl1", "parameters": {}}]`),
			expected: FunctionSpecs{
				{Name: "func1", ImplKey: "impl1", Parameters: models.JSONMap{}},
			},
			hasError: false,
		},
		{
			name:  "Valid JSON Object",
			input: []byte(`{"name": "func1", "impl_key": "impl1", "parameters": {}}`),
			expected: FunctionSpecs{
				{Name: "func1", ImplKey: "impl1", Parameters: models.JSONMap{}},
			},
			hasError: false,
		},
		{
			name:     "Nil Value",
			input:    nil,
			expected: nil,
			hasError: false,
		},
		{
			name:     "Empty String",
			input:    []byte(``),
			expected: FunctionSpecs{},
			hasError: false,
		},
		{
			name:     "Invalid JSON",
			input:    []byte(`{"name": "func1"`),
			expected: nil,
			hasError: true,
		},
		{
			name:     "Unsupported Type",
			input:    123,
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fs FunctionSpecs
			err := fs.Scan(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.input == nil {
					assert.Nil(t, fs)
				} else {
					assert.Equal(t, tt.expected, fs)
				}
			}
		})
	}
}

func TestValidateLimitsAndScopes(t *testing.T) {
	base := func() Agent {
		return Agent{
			Name: "a", SystemPrompt: "p", MaxTurns: 2,
			LLMConfig: LLMConfig{Provider: "openai"},
			Memory:    MemoryConfig{Type: "in-memory", SessionScope: "agent"},
			Trigger:   TriggerConfig{Type: "manual"},
		}
	}
	a := base()
	a.Limits = Limits{MaxRunsPerHour: 10, MaxTokensPerRun: 5000}
	assert.NoError(t, ValidateAgentSchema(a))

	a = base()
	a.Limits = Limits{MaxCostPerDay: 1}
	assert.ErrorContains(t, ValidateAgentSchema(a), "needs prices", "a cost limit without prices could never trigger")
	a.LLMConfig.InputCostPerMillion = 2.5
	assert.NoError(t, ValidateAgentSchema(a))

	a = base()
	a.Limits = Limits{MaxRunsPerDay: -1}
	assert.Error(t, ValidateAgentSchema(a))

	a = base()
	a.Functions = FunctionSpecs{{Name: "rm", ImplKey: "collections.delete_item", Parameters: models.JSONMap{}, Collections: []string{"drafts"}}}
	assert.NoError(t, ValidateAgentSchema(a))
	a.Functions = FunctionSpecs{{Name: "sum", ImplKey: "llm.summarize", Parameters: models.JSONMap{}, Collections: []string{"drafts"}}}
	assert.ErrorContains(t, ValidateAgentSchema(a), "only applies to content tools")
	a.Functions = FunctionSpecs{{Name: "rm", ImplKey: "collections.delete_item", Parameters: models.JSONMap{}, Collections: []string{" "}}}
	assert.Error(t, ValidateAgentSchema(a))
}

// The short form: {"use", "on"} with everything else from the catalog.
func TestToolsShortForm(t *testing.T) {
	base := map[string]any{
		"name": "summarizer", "system_prompt": "Summarize.", "max_turns": 3,
		"llm_config": map[string]any{"provider": "openai"},
		"memory":     map[string]any{"type": "in-memory", "session_scope": "agent"},
		"trigger":    map[string]any{"type": "manual"},
	}

	in := maps.Clone(base)
	in["tools"] = []any{
		map[string]any{"use": "content.update", "on": []any{"articles"}},
		map[string]any{"use": "ai.summarize"},
		map[string]any{"use": "collections.list_items", "name": "list_articles", "on": []any{"articles"}},
	}
	// "tools" wins over "functions", as when a client sends back what it read.
	in["functions"] = []any{map[string]any{"name": "ignored", "impl_key": "system.log", "parameters": map[string]any{}}}
	agent, err := ParseAgentInput(in)
	require.NoError(t, err)
	require.Len(t, agent.Functions, 3)

	// Stored: only what the agent says.
	assert.Equal(t, FunctionSpec{ImplKey: "collections.patch_item", Collections: []string{"articles"}}, agent.Functions[0])
	assert.Equal(t, "llm.summarize_text", agent.Functions[1].ImplKey)
	assert.Equal(t, "list_articles", agent.Functions[2].Name)

	// Resolved for the model: the catalog's name, description and schema.
	r := agent.Functions.Resolved()
	assert.Equal(t, "update_item", r[0].Name)
	assert.Equal(t, "Change some attributes of an item and keep the others, validated like API writes. Only in: articles", r[0].Description)
	assert.Equal(t, []any{"collection_name", "item_id", "data"}, toAny(r[0].Parameters["required"]))
	assert.Equal(t, "summarize_text", r[1].Name)
	assert.Equal(t, "list_articles", r[2].Name)

	// Back to the short form: nothing but what was written.
	assert.Equal(t, []Tool{
		{Use: "content.update", On: []string{"articles"}},
		{Use: "ai.summarize"},
		{Use: "content.list", On: []string{"articles"}, Name: "list_articles"},
	}, agent.Functions.Tools())

	// Two tools with the same default name need their own.
	in = maps.Clone(base)
	in["tools"] = []any{map[string]any{"use": "content.update", "on": []any{"a"}}, map[string]any{"use": "content.update", "on": []any{"b"}}}
	_, err = ParseAgentInput(in)
	assert.ErrorContains(t, err, "two tools are named 'update_item'")

	in["tools"] = []any{map[string]any{"use": "ai.summarize", "on": []any{"a"}}}
	_, err = ParseAgentInput(in)
	assert.ErrorContains(t, err, "only applies to content tools")

	in["tools"] = []any{map[string]any{"on": []any{"a"}}}
	_, err = ParseAgentInput(in)
	assert.ErrorContains(t, err, "say which tool with 'use'")
}

// Agents written in the long form keep working: what they give is kept, and
// an empty schema gets the catalog's.
func TestToolsLongFormStillWorks(t *testing.T) {
	f := FunctionSpec{Name: "save", Description: "Save an article", ImplKey: "collections.upsert_item", Parameters: models.JSONMap{"type": "object", "properties": map[string]any{}}, Collections: []string{"articles"}}
	r := f.Resolved()
	assert.Equal(t, "save", r.Name)
	assert.Equal(t, "Save an article", r.Description)
	assert.NotEmpty(t, r.Parameters["properties"], "an empty schema gets the tool's own")
	assert.Equal(t, Tool{Use: "content.save", On: []string{"articles"}, Name: "save", Description: "Save an article"}, ToolOf(f))

	custom := FunctionSpec{Name: "check_stock", ImplKey: "shop.check_stock"}
	assert.Equal(t, "check_stock", custom.Resolved().Name, "functions outside the catalog keep their name")
	assert.Equal(t, Tool{Use: "shop.check_stock", Name: "check_stock"}, ToolOf(custom))
}

func toAny(v any) []any {
	switch x := v.(type) {
	case []string:
		out := make([]any, len(x))
		for i, s := range x {
			out[i] = s
		}
		return out
	case []any:
		return x
	}
	return nil
}
