package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms"

	config "github.com/gohead-cms/gohead/pkg/config"
	anthropic_client "github.com/gohead-cms/gohead/pkg/llm/anthropic"
	ollama_client "github.com/gohead-cms/gohead/pkg/llm/ollama"
	openai_client "github.com/gohead-cms/gohead/pkg/llm/openai"
)

// Role represents the role of a message sender.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Provider represents the LLM provider.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderOllama    Provider = "ollama"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ToolCall represents a request to call a tool.
type ToolCall struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

// ResponseType indicates the type of the LLM's response.
type ResponseType string

const (
	ResponseTypeText     ResponseType = "text"
	ResponseTypeToolCall ResponseType = "tool_call"
)

// Response represents the output from an LLM chat call.
type Response struct {
	Type     ResponseType
	Content  string
	ToolCall *ToolCall
}

// Client is the interface that all LLM providers must implement.
type Client interface {
	Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error)
}

// Config represents the LLM configuration, likely from your agent model.
type Config = config.LLMConfig

// Option is a functional option for the Chat method.
type Option func(*options)

type options struct {
	Tools      []llms.Tool
	ToolChoice any
}

func WithTools(tools []llms.Tool) Option {
	return func(o *options) {
		o.Tools = tools
	}
}

// WithToolChoice sets the tool choice option for the LLM call.
func WithToolChoice(choice any) Option {
	return func(o *options) {
		o.ToolChoice = choice
	}
}

// langChainAdapter is a wrapper around a langchaingo LLM client.
type langChainAdapter struct {
	client llms.Model
}

// Chat implements the Client interface using the langchaingo library.
func (a *langChainAdapter) Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error) {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	// 1) Convert internal messages to langchaingo []llms.MessageContent
	lcMessages := make([]llms.MessageContent, 0, len(messages))
	for _, msg := range messages {
		lcMessages = append(lcMessages, llms.TextParts(convertRole(msg.Role), msg.Content))
	}

	// 2) Build call options - now cfg.Tools is already []llms.Tool
	var callOpts []llms.CallOption
	if len(cfg.Tools) > 0 {
		callOpts = append(callOpts, llms.WithTools(cfg.Tools))
	}

	// Add tool choice if specified
	if cfg.ToolChoice != nil {
		// You might need to handle ToolChoice conversion here
		// depending on your LLM provider's expectations
	}

	// 3) Call the model
	res, err := a.client.GenerateContent(ctx, lcMessages, callOpts...)
	if err != nil {
		return nil, fmt.Errorf("langchaingo chat failed: %w", err)
	}

	// 4) Convert the langchaingo response back to our internal Response type
	if res == nil || len(res.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from LLM")
	}

	choice := res.Choices[0]

	// Function-calling (single func)
	if choice.FuncCall != nil {
		return &Response{
			Type: ResponseTypeToolCall,
			ToolCall: &ToolCall{
				Name:      choice.FuncCall.Name,
				Arguments: choice.FuncCall.Arguments,
			},
		}, nil
	}

	// Multi-tool calling path
	if len(choice.ToolCalls) > 0 && choice.ToolCalls[0].FunctionCall != nil {
		fc := choice.ToolCalls[0].FunctionCall
		return &Response{
			Type: ResponseTypeToolCall,
			ToolCall: &ToolCall{
				Name:      fc.Name,
				Arguments: fc.Arguments,
			},
		}, nil
	}

	// Plain text response
	return &Response{
		Type:    ResponseTypeText,
		Content: choice.Content,
	}, nil
}

// Helper function to map our roles to LangChainGo roles.
func convertRole(role Role) llms.ChatMessageType {
	switch role {
	case RoleSystem:
		return llms.ChatMessageTypeSystem
	case RoleUser:
		return llms.ChatMessageTypeHuman
	case RoleAssistant:
		return llms.ChatMessageTypeAI
	case RoleTool:
		return llms.ChatMessageTypeTool
	default:
		return llms.ChatMessageTypeGeneric
	}
}

// NewAdapter creates a new `Client` by wrapping the specified LLM provider.
func NewAdapter(cfg Config) (Client, error) {
	var client llms.Model
	var err error
	if err := config.ApplyLLMEnv(cfg); err != nil {
		return nil, err
	}
	switch cfg.Provider {
	case "openai":
		client, err = openai_client.New()
	case "anthropic":
		client, err = anthropic_client.New()
	case "ollama":
		// Ollama is typically  run locally and uses a model name
		client, err = ollama_client.New(cfg.Model)
	default:
		return nil, errors.New("unsupported LLM provider")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create client for provider %s: %w", cfg.Provider, err)
	}

	return &langChainAdapter{client: client}, nil
}
