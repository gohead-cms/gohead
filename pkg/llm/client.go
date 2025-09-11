package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// Role represents the role of a message sender.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ToolCall represents a request to call a tool.
type ToolCall struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"`
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
type Config struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// Option is a functional option for the Chat method.
type Option func(*options)

type options struct {
	Tools []tools.Tool
}

// WithTools adds tools to the LLM's context.
func WithTools(tools []tools.Tool) Option {
	return func(o *options) {
		o.Tools = tools
	}
}

// NewClient is a factory function that creates an LLM client based on the provider.
func NewClient(cfg Config) (Client, error) {
	switch cfg.Provider {
	case "openai":
		lcClient, err := openai.New(openai.WithModel(cfg.Model), openai.WithToken(cfg.APIKey))
		if err != nil {
			return nil, err
		}
		return &langChainAdapter{
			client: lcClient,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
