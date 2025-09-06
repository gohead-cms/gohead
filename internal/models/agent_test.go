package models_test

import (
	"encoding/json"
	"errors"
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
					"impl_key":    "tool.test",
					"name":        "TestTool",
					"description": "A test tool.",
					"parameters":  `{"type": "object", "properties": {}}`,
				},
			},
		}
		agent, err := models.ParseAgentInput(input)
		assert.NoError(t, err)
		assert.Equal(t, "test-agent", agent.Name)
		assert.Equal(t, 5, agent.MaxTurns)
		assert.Equal(t, "openai", agent.LLMConfig.Provider)
		assert.Equal(t, "postgres", agent.Memory.Type)
		assert.Equal(t, "manual", agent.Trigger.Type)
		assert.Equal(t, 1, len(agent.Functions))
	})

	t.Run("Invalid Input Format - Inside the map", func(t *testing.T) {
		// Pass a valid top-level map, but with an invalid data type inside.
		// Forcing a type that json.Unmarshal can't handle.
		input := map[string]any{
			"name":      "test-agent",
			"max_turns": "five", // Invalid type: string instead of int
		}

		_, err := models.ParseAgentInput(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid agent input format")

		// For a more specific assertion on the underlying JSON error:
		if err != nil {
			var unmarshalErr *json.UnmarshalTypeError
			assert.True(t, errors.As(err, &unmarshalErr))
			assert.Contains(t, unmarshalErr.Error(), "cannot unmarshal string into Go struct field Agent.max_turns of type int")
		}
	})
}
func TestValidateAgentSchema(t *testing.T) {
	// A valid base agent for testing modifications
	baseAgent := models.Agent{
		Name:         "ValidAgent",
		SystemPrompt: "You are a helpful assistant.",
		MaxTurns:     10,
		LLMConfig: models.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Memory: models.MemoryConfig{
			Type:         "postgres",
			SessionScope: "user-session",
		},
		Trigger: models.TriggerConfig{
			Type: "manual",
		},
		Functions: []models.FunctionSpec{
			{
				ImplKey:     "email.send",
				Name:        "SendEmail",
				Description: "Sends an email.",
				Parameters:  `{"type": "object", "properties": {}}`,
			},
		},
	}

	t.Run("Valid Agent", func(t *testing.T) {
		err := models.ValidateAgentSchema(baseAgent)
		assert.NoError(t, err)
	})

	t.Run("Missing Name", func(t *testing.T) {
		agent := baseAgent
		agent.Name = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent must have a 'name'")
	})

	t.Run("Missing System Prompt", func(t *testing.T) {
		agent := baseAgent
		agent.SystemPrompt = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent must have a 'system_prompt'")
	})

	t.Run("Invalid Max Turns", func(t *testing.T) {
		agent := baseAgent
		agent.MaxTurns = 0
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_turns must be a positive integer")
	})

	t.Run("Missing LLM Provider", func(t *testing.T) {
		agent := baseAgent
		agent.LLMConfig.Provider = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "llm_config provider cannot be empty")
	})

	t.Run("Missing Memory Type", func(t *testing.T) {
		agent := baseAgent
		agent.Memory.Type = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory type cannot be empty")
	})

	t.Run("Missing Memory Session Scope", func(t *testing.T) {
		agent := baseAgent
		agent.Memory.SessionScope = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory session scope cannot be empty")
	})

	t.Run("Invalid Trigger Type", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger.Type = "invalid"
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid trigger type")
	})

	t.Run("Valid Cron Trigger", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger.Type = "cron"
		agent.Trigger.Cron = "@hourly"
		err := models.ValidateAgentSchema(agent)
		assert.NoError(t, err)
	})

	t.Run("Invalid Cron Expression", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger.Type = "cron"
		agent.Trigger.Cron = "invalid-cron"
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("Valid Webhook Trigger", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger.Type = "webhook"
		agent.Trigger.WebhookToken = "super-secret-token"
		err := models.ValidateAgentSchema(agent)
		assert.NoError(t, err)
	})

	t.Run("Missing Webhook Token", func(t *testing.T) {
		agent := baseAgent
		agent.Trigger.Type = "webhook"
		agent.Trigger.WebhookToken = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook trigger requires a 'webhook_token'")
	})

	t.Run("Missing Function Impl Key", func(t *testing.T) {
		agent := baseAgent
		agent.Functions[0].ImplKey = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function must have a non-empty 'impl_key'")
	})

	t.Run("Missing Function Name", func(t *testing.T) {
		agent := baseAgent
		agent.Functions[0].Name = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function must have a non-empty 'name'")
	})

	t.Run("Duplicate Function Impl Key", func(t *testing.T) {
		agent := baseAgent
		agent.Functions = append(agent.Functions, models.FunctionSpec{
			ImplKey:     "email.send", // Duplicate
			Name:        "SendEmail2",
			Description: "A test tool.",
			Parameters:  `{"type": "object", "properties": {}}`,
		})
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
	})

	t.Run("Duplicate Function Name", func(t *testing.T) {
		agent := baseAgent
		agent.Functions = append(agent.Functions, models.FunctionSpec{
			ImplKey:     "email.send.2",
			Name:        "SendEmail", // Duplicate
			Description: "A test tool.",
			Parameters:  `{"type": "object", "properties": {}}`,
		})
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
	})

	t.Run("Missing Function Parameters", func(t *testing.T) {
		agent := baseAgent
		agent.Functions[0].Parameters = ""
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
	})

	t.Run("Invalid Function Parameters JSON", func(t *testing.T) {
		agent := baseAgent
		agent.Functions[0].Parameters = `{"type": "object"` // Invalid JSON
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
	})
}
