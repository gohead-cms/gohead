package models_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

// Helper function to marshal a struct to JSON for use in tests.
func marshalJSON(v interface{}) datatypes.JSON {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return datatypes.JSON(b)
}

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

		// Assertions for the main fields
		assert.Equal(t, "test-agent", agent.Name)
		assert.Equal(t, 5, agent.MaxTurns)
		assert.Equal(t, "You are a test agent.", agent.SystemPrompt)

		// Assertions for nested JSONB fields by unmarshaling first
		var llmConfig models.LLMConfig
		err = json.Unmarshal(agent.LLMConfig, &llmConfig)
		assert.NoError(t, err)
		assert.Equal(t, "openai", llmConfig.Provider)
		assert.Equal(t, "gpt-3.5-turbo", llmConfig.Model)

		var memoryConfig models.MemoryConfig
		err = json.Unmarshal(agent.Memory, &memoryConfig)
		assert.NoError(t, err)
		assert.Equal(t, "postgres", memoryConfig.Type)
		assert.Equal(t, "conversation", memoryConfig.SessionScope)

		var triggerConfig models.TriggerConfig
		err = json.Unmarshal(agent.Trigger, &triggerConfig)
		assert.NoError(t, err)
		assert.Equal(t, "manual", triggerConfig.Type)

		var functions []models.FunctionSpec
		err = json.Unmarshal(agent.Functions, &functions)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(functions))
		assert.Equal(t, "TestTool", functions[0].Name)
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
	// A valid base agent for testing modifications
	validLLMConfig := models.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}
	validMemoryConfig := models.MemoryConfig{
		Type:         "postgres",
		SessionScope: "user-session",
	}
	validTriggerConfig := models.TriggerConfig{
		Type: "manual",
	}
	validFunctions := []models.FunctionSpec{
		{
			ImplKey:     "email.send",
			Name:        "SendEmail",
			Description: "Sends an email.",
			Parameters:  `{"type": "object", "properties": {}}`,
		},
	}

	// Construct the base agent with the new JSON types using the helper function
	baseAgent := models.Agent{
		Name:         "ValidAgent",
		SystemPrompt: "You are a helpful assistant.",
		MaxTurns:     10,
		LLMConfig:    marshalJSON(validLLMConfig),
		Memory:       marshalJSON(validMemoryConfig),
		Trigger:      marshalJSON(validTriggerConfig),
		Functions:    marshalJSON(validFunctions),
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
		invalidLLMConfig := validLLMConfig
		invalidLLMConfig.Provider = ""
		agent.LLMConfig = marshalJSON(invalidLLMConfig)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "llm_config provider cannot be empty")
	})

	t.Run("Missing Memory Type", func(t *testing.T) {
		agent := baseAgent
		invalidMemoryConfig := validMemoryConfig
		invalidMemoryConfig.Type = ""
		agent.Memory = marshalJSON(invalidMemoryConfig)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory type cannot be empty")
	})

	t.Run("Missing Memory Session Scope", func(t *testing.T) {
		agent := baseAgent
		invalidMemoryConfig := validMemoryConfig
		invalidMemoryConfig.SessionScope = ""
		agent.Memory = marshalJSON(invalidMemoryConfig)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory session scope cannot be empty")
	})

	t.Run("Missing Memory Session Scope", func(t *testing.T) {
		agent := baseAgent
		invalidMemoryConfig := validMemoryConfig
		invalidMemoryConfig.SessionScope = ""
		agent.Memory = marshalJSON(invalidMemoryConfig)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory session scope cannot be empty")
	})

	t.Run("Valid Cron Trigger", func(t *testing.T) {
		agent := baseAgent
		validCronTrigger := models.TriggerConfig{
			Type: "cron",
			Cron: "@hourly",
		}
		agent.Trigger = marshalJSON(validCronTrigger)
		err := models.ValidateAgentSchema(agent)
		assert.NoError(t, err)
	})

	t.Run("Invalid Cron Expression", func(t *testing.T) {
		agent := baseAgent
		invalidCronTrigger := models.TriggerConfig{
			Type: "cron",
			Cron: "invalid-cron",
		}
		agent.Trigger = marshalJSON(invalidCronTrigger)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("Valid Webhook Trigger", func(t *testing.T) {
		agent := baseAgent
		validWebhookTrigger := models.TriggerConfig{
			Type:         "webhook",
			WebhookToken: "super-secret-token",
		}
		agent.Trigger = marshalJSON(validWebhookTrigger)
		err := models.ValidateAgentSchema(agent)
		assert.NoError(t, err)
	})

	t.Run("Missing Webhook Token", func(t *testing.T) {
		agent := baseAgent
		invalidWebhookTrigger := models.TriggerConfig{
			Type:         "webhook",
			WebhookToken: "",
		}
		agent.Trigger = marshalJSON(invalidWebhookTrigger)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook trigger requires a 'webhook_token'")
	})

	t.Run("Missing Function Impl Key", func(t *testing.T) {
		agent := baseAgent
		var tempFunctions []models.FunctionSpec
		json.Unmarshal(agent.Functions, &tempFunctions)
		tempFunctions[0].ImplKey = ""
		agent.Functions = marshalJSON(tempFunctions)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function must have a non-empty 'impl_key'")
	})

	t.Run("Missing Function Name", func(t *testing.T) {
		agent := baseAgent
		var tempFunctions []models.FunctionSpec
		json.Unmarshal(agent.Functions, &tempFunctions)
		tempFunctions[0].Name = ""
		agent.Functions = marshalJSON(tempFunctions)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function must have a non-empty 'name'")
	})

	t.Run("Duplicate Function Impl Key", func(t *testing.T) {
		agent := baseAgent
		var tempFunctions []models.FunctionSpec
		json.Unmarshal(agent.Functions, &tempFunctions)
		tempFunctions = append(tempFunctions, models.FunctionSpec{
			ImplKey:     "email.send", // Duplicate
			Name:        "SendEmail2",
			Description: "A test tool.",
			Parameters:  `{"type": "object", "properties": {}}`,
		})
		agent.Functions = marshalJSON(tempFunctions)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate function implementation key")
	})

	t.Run("Duplicate Function Impl Key", func(t *testing.T) {
		agent := baseAgent
		var tempFunctions []models.FunctionSpec
		json.Unmarshal(agent.Functions, &tempFunctions)
		tempFunctions = append(tempFunctions, models.FunctionSpec{
			ImplKey:     "email.send", // Duplicate
			Name:        "SendEmail2",
			Description: "A test tool.",
			Parameters:  `{"type": "object", "properties": {}}`,
		})
		agent.Functions = marshalJSON(tempFunctions)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate function implementation key")
	})

	t.Run("Missing Function Parameters", func(t *testing.T) {
		agent := baseAgent
		var tempFunctions []models.FunctionSpec
		json.Unmarshal(agent.Functions, &tempFunctions)
		tempFunctions[0].Parameters = ""
		agent.Functions = marshalJSON(tempFunctions)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function parameters cannot be empty")
	})

	t.Run("Invalid Function Parameters JSON", func(t *testing.T) {
		agent := baseAgent
		var tempFunctions []models.FunctionSpec
		json.Unmarshal(agent.Functions, &tempFunctions)
		tempFunctions[0].Parameters = `{"type": "object"` // Invalid JSON
		agent.Functions = marshalJSON(tempFunctions)
		err := models.ValidateAgentSchema(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON in function parameters")
	})
}
