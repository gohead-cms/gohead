package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// langChainAdapter is a wrapper around a langchaingo LLM client.
type langChainAdapter struct {
	client llms.Model
}

// Chat implements the Client interface using the langchaingo library.
func (a *langChainAdapter) Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error) {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	// 1. Convert our internal messages to langchaingo's schema.Message
	lcMessages := make([]schema.ChatMessage, len(messages))
	for i, msg := range messages {
		lcMessages[i] = schema.ChatMessage{
			Text: msg.Content,
			Type: convertRole(msg.Role),
		}
	}

	// 2. The core LangChain call. This is where the magic happens.
	// We pass the tools and the messages, and the library handles the routing.
	res, err := a.client.Generate(ctx, lcMessages, llms.WithTools(options.Tools))
	if err != nil {
		return nil, fmt.Errorf("langchaingo chat failed: %w", err)
	}

	// 3. Convert the langchaingo response back to our internal Response type
	if len(res.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from LLM")
	}

	lcChoice := res.Choices[0]
	if lcChoice.FunctionCall != nil {
		// LLM wants to call a tool
		return &Response{
			Type: ResponseTypeToolCall,
			ToolCall: &ToolCall{
				Name:      lcChoice.FunctionCall.Name,
				Arguments: json.RawMessage(lcChoice.FunctionCall.Arguments),
			},
		}, nil
	}

	// LLM returned a text response
	return &Response{
		Type:    ResponseTypeText,
		Content: lcChoice.Content,
	}, nil
}

// Helper function to map our roles to LangChain's roles.
func convertRole(role Role) schema.ChatMessageType {
	switch role {
	case RoleSystem:
		return schema.ChatMessageTypeSystem
	case RoleUser:
		return schema.ChatMessageTypeHuman
	case RoleAssistant:
		return schema.ChatMessageTypeAI
	case RoleTool:
		return schema.ChatMessageTypeFunction
	default:
		return schema.ChatMessageTypeGeneric
	}
}
