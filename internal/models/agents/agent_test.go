package models

import (
	"bytes"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
			errMsg:   "invalid trigger type: must be 'manual', 'cron', or 'webhook'",
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
			name:     "Function with Empty Name",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "manual"}, Functions: FunctionSpecs{{Name: ""}}},
			hasError: true,
			errMsg:   "function must have a non-empty 'name'",
		},
		{
			name:     "Function with Empty ImplKey",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "manual"}, Functions: FunctionSpecs{{Name: "test_func"}}},
			hasError: true,
			errMsg:   "function must have a non-empty 'impl_key'",
		},
		{
			name:     "Function with Invalid Parameters JSON",
			agent:    Agent{Name: "test", SystemPrompt: "...", MaxTurns: 1, LLMConfig: LLMConfig{Provider: "p"}, Memory: MemoryConfig{Type: "in-memory", SessionScope: "user"}, Trigger: TriggerConfig{Type: "manual"}, Functions: FunctionSpecs{{Name: "test_func", ImplKey: "test_impl", Parameters: models.JSONMap{"invalid": make(chan int)}}}},
			hasError: true,
			errMsg:   "function parameters cannot be empty",
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
