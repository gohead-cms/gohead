package models

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestParseAgentInput(t *testing.T) {
	t.Run("Valid Input", func(t *testing.T) {
		input := map[string]any{
			"name":          "test-agent",
			"system_prompt": "You are a test agent.",
			"max_turns":     5,
			"llm_config": map[string]any{
				"provider": "openai",
				"model":    "gpt-3.5-turbo",
			},
			"memory": map[string]any{
				"type":          "postgres",
				"session_scope": "conversation",
			},
			"trigger": map[string]any{
				"type": "manual",
			},
			"functions": []any{
				map[string]any{
					"name":        "TestTool",
					"description": "A test tool.",
					"impl_key":    "tool.test",
					// parameters may be provided as a JSON object; your validator will marshal it to JSON.
					"parameters": map[string]any{
						"type":       "object",
						"properties": map[string]any{},
					},
				},
			},
		}
		agent, err := ParseAgentInput(input)
		assert.NoError(t, err)

		// Assertions for the main fields
		assert.Equal(t, "test-agent", agent.Name)
		assert.Equal(t, 5, agent.MaxTurns)
		assert.Equal(t, "You are a test agent.", agent.SystemPrompt)

		// Assertions for nested fields (direct access, no unmarshaling needed)
		assert.Equal(t, "openai", agent.LLMConfig.Provider)
		assert.Equal(t, "gpt-3.5-turbo", agent.LLMConfig.Model)

		// memory is a JSONMap, so direct map access is correct
		assert.Equal(t, "postgres", agent.Memory.Type)
		assert.Equal(t, "conversation", agent.Memory.SessionScope)

		assert.Equal(t, "manual", agent.Trigger.Type)

		functions := agent.Functions
		assert.Equal(t, 1, len(functions))
		function := functions[0]
		assert.Equal(t, "TestTool", function.Name)
	})

	t.Run("Invalid Input Format - Inside the map", func(t *testing.T) {
		// Pass a valid top-level map, but with an invalid data type inside.
		// Forcing a type that json.Unmarshal can't handle.
		input := map[string]any{
			"name":      "test-agent",
			"max_turns": "five", // Invalid type: string instead of int
		}

		_, err := ParseAgentInput(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json: cannot unmarshal string into Go struct field Agent.max_turns of type int")
	})
}
func TestValidateAgentSchema(t *testing.T) {
	// A valid base agent for testing modifications
	validLLMConfig := LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}
	validMemoryMap := MemoryConfig{
		Type:         "postgres",
		SessionScope: "user-session",
	}
	validTriggerConfig := TriggerConfig{
		Type: "manual",
	}
	validFunctionsList := []FunctionSpec{
		{
			ImplKey:     "email.send",
			Name:        "SendEmail",
			Description: "Sends an email.",
			Parameters:  models.JSONMap{"type": "object", "properties": map[string]any{}},
		},
	}

	baseAgent := Agent{
		Name:         "ValidAgent",
		SystemPrompt: "You are a helpful assistant.",
		MaxTurns:     10,
		LLMConfig:    validLLMConfig,
		Memory:       validMemoryMap,
		Trigger:      validTriggerConfig,
		Functions:    validFunctionsList,
	}

	t.Run("Valid Agent", func(t *testing.T) {
		err := ValidateAgentSchema(baseAgent)
		assert.NoError(t, err)
	})

	t.Run("Missing Name", func(t *testing.T) {
		agent := baseAgent
		agent.Name = ""
		err := ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent must have a 'name'")
	})

	t.Run("Missing System Prompt", func(t *testing.T) {
		agent := baseAgent
		agent.SystemPrompt = ""
		err := ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent must have a 'system_prompt'")
	})

	t.Run("Invalid Max Turns", func(t *testing.T) {
		agent := baseAgent
		agent.MaxTurns = 0
		err := ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_turns must be a positive integer")
	})

	t.Run("Missing LLM Provider", func(t *testing.T) {
		agent := baseAgent
		agent.LLMConfig.Provider = ""
		err := ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "llm_config provider cannot be empty")
	})

	t.Run("Missing Memory Type", func(t *testing.T) {
		agent := baseAgent
		agent.Memory.Type = ""
		err := ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory type cannot be empty")
	})

	t.Run("Missing Memory Session Scope", func(t *testing.T) {
		agent := baseAgent
		agent.Memory.SessionScope = ""
		err := ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory session scope cannot be empty")
	})

	t.Run("Valid Cron Trigger", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger = TriggerConfig{
			Type: "cron",
			Cron: "@hourly",
		}
		err := ValidateAgentSchema(agent)
		assert.NoError(t, err)
	})

	t.Run("Invalid Cron Expression", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger = TriggerConfig{
			Type: "cron",
			Cron: "invalid-cron",
		}
		err := ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("Valid Webhook Trigger", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger = TriggerConfig{
			Type:         "webhook",
			WebhookToken: "super-secret-token",
		}
		err := ValidateAgentSchema(agent)
		assert.NoError(t, err)
	})

	t.Run("Missing Webhook Token", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger = TriggerConfig{
			Type:         "webhook",
			WebhookToken: "",
		}
		err := ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook trigger requires a 'webhook_token'")
	})
}
