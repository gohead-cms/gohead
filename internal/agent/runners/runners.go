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

// AgentRunner executes agentic workflows.
type AgentRunner struct{}

// NewAgentRunner creates a new, clean instance of the AgentRunner.
func NewAgentRunner() *AgentRunner {
	return &AgentRunner{}
}

// HandleAgentJob is the entry point for processing jobs from the 'agents' queue.
func (r *AgentRunner) HandleAgentJob(ctx context.Context, t *asynq.Task) error {
	var payload jobs.AgentJobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Log.WithError(err).Error("Failed to unmarshal agent job payload")
		return fmt.Errorf("could not unmarshal payload: %w", err)
	}

	logger.Log.WithField("agent_id", payload.AgentID).Info("Starting agent job execution")

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

// runConversation contains the main agent loop: LLM calls, function execution, and history management.
func (r *AgentRunner) runConversation(ctx context.Context, agent *agentModels.Agent, payload jobs.AgentJobPayload) error {
	// 1. Setup: Create LLM client and function registry
	llmConfigForAdapter := llm.Config{
		Provider:  agent.LLMConfig.Provider,
		Model:     agent.LLMConfig.Model,
		APIKey:    agent.LLMConfig.APIKey,
		APISecret: agent.LLMConfig.APISecret,
	}

	llmClient, err := llm.NewAdapter(llmConfigForAdapter)
	if err != nil {
		return fmt.Errorf("could not create LLM adapter: %w", err)
	}

	// Create function registry for execution
	registry := functions.NewRegistry(agent.Functions)

	langchainTools := registry.ToLangchainTools()
	logger.Log.WithFields(map[string]any{
		"tools": langchainTools,
	}).Info("tools")
	// 2. Prepare conversation history
	history, err := storage.GetConversationHistory(agent.ID)
	if err != nil {
		return fmt.Errorf("could not load conversation history: %w", err)
	}
	logger.Log.WithFields(map[string]any{
		"history": history,
	}).Info("Agent History")
	contextualInput := r.createContextualInput(payload)
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: agent.SystemPrompt},
	}
	messages = append(messages, history...)
	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: contextualInput})

	// 3. Start execution loop
	for i := 0; i < agent.MaxTurns; i++ {
		// Log the current turn
		logger.Log.WithFields(map[string]any{
			"agent_id":  agent.ID,
			"turn":      i + 1,
			"max_turns": agent.MaxTurns,
		}).Info("Starting turn")

		logger.Log.Info("ENVOI")

		response, err := llmClient.Chat(
			ctx,
			messages,
			llm.WithTools(langchainTools),
		)
		logger.Log.WithFields(map[string]any{
			"response": response,
		}).Info("LLM Response")
		logger.Log.Info(response)
		if err != nil {
			return fmt.Errorf("LLM call failed on turn %d: %w", i+1, err)
		}

		if response.Type == llm.ResponseTypeToolCall {
			toolCall := response.ToolCall
			logger.Log.WithFields(map[string]any{
				"agent_id":       agent.ID,
				"tool_id":        toolCall.ID,
				"tool_name":      toolCall.FunctionCall.Name,
				"tool_arguments": toolCall.FunctionCall.Arguments,
			}).Info("LLM requested tool call")

			// Add assistant message (tool call request)
			messages = append(messages, llm.Message{
				Role:     llm.RoleAssistant,
				ToolCall: toolCall,
			})

			// Execute the tool
			fn, ok := registry.Get(toolCall.FunctionCall.Name)
			if !ok {
				errorPayload := map[string]string{"error": "Tool not found", "requested_tool": toolCall.FunctionCall.Name}
				errorJSON, _ := json.Marshal(errorPayload)
				// Add a tool response message with the error and the corresponding ID
				messages = append(messages, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: toolCall.ID,
					Content:    string(errorJSON),
				})
				logger.Log.WithField("tool_name", toolCall.FunctionCall.Name).Warn("Tool not found in registry")
				continue // Continue to the next turn
			}

			// Execute the function with the arguments
			result, err := fn(ctx, toolCall.FunctionCall.Arguments)
			if err != nil {
				errorPayload := map[string]string{"error": "Tool execution failed", "message": err.Error()}
				errorJSON, _ := json.Marshal(errorPayload)
				// Add a tool response message with the error and the corresponding ID
				messages = append(messages, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: toolCall.ID,
					Content:    string(errorJSON),
				})
				logger.Log.WithError(err).WithField("tool_name", toolCall.FunctionCall.Name).Error("Tool execution failed")
				continue // Continue to the next turn
			}

			// Add tool result to messages
			messages = append(messages, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: toolCall.ID,
				Content:    result,
			})

			logger.Log.WithFields(map[string]any{
				"agent_id":  agent.ID,
				"tool_name": toolCall.FunctionCall.Name,
			}).Info("Tool executed successfully")

		} else { // Plain text response
			messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: response.Content})
			logger.Log.WithField("agent_id", agent.ID).Info("LLM finished with text response. Ending turn.")
			break
		}
	}
	logger.Log.Info("MESSAGE")
	logger.Log.Info(messages)
	// 4. Save the final conversation history
	if err := storage.SaveConversationHistory(agent.ID, messages[1:]); err != nil {
		return fmt.Errorf("could not save conversation history: %w", err)
	}

	return nil
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
