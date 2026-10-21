package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"

	config "github.com/gohead-cms/gohead/pkg/config"
	anthropic_client "github.com/gohead-cms/gohead/pkg/llm/anthropic"
	ollama_client "github.com/gohead-cms/gohead/pkg/llm/ollama"
	openai_client "github.com/gohead-cms/gohead/pkg/llm/openai"
	"github.com/gohead-cms/gohead/pkg/logger"
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
	Role       Role           `json:"role"`
	Content    string         `json:"content"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCall   *llms.ToolCall `json:"tool_call,omitempty"`
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
	ToolCall *llms.ToolCall
	Usage    Usage // tokens of this call, and its estimated cost
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
	cfg    Config
}

// Chat implements the Client interface using the langchaingo library.
func (a *langChainAdapter) Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error) {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	logger.Log.Debug("Running chat with proper history conversion")

	lcMessages := make([]llms.MessageContent, 0, len(messages))
	for _, msg := range messages {
		role := convertRole(msg.Role)
		logger.Log.WithFields(map[string]any{
			"message": msg.Content,
			"role":    msg.Role,
		}).Debug("Message")
		switch msg.Role {
		case RoleTool:
			lcMessages = append(lcMessages, llms.MessageContent{
				Role: role,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: msg.ToolCallID,
					Content:    msg.Content,
				}},
			})
		case RoleAssistant:
			if msg.ToolCall != nil {
				// Assistant message with tool call - needs both content and tool call
				parts := []llms.ContentPart{}

				// Add text content first (even if empty)
				if msg.Content != "" {
					parts = append(parts, llms.TextPart(msg.Content))
				}

				logger.Log.WithFields(map[string]any{
					"arguments": msg.ToolCall.FunctionCall.Arguments,
				}).Info("LLM Response")
				// Add the tool call
				parts = append(parts, llms.ToolCall{
					ID:   msg.ToolCall.ID,
					Type: "function",
					FunctionCall: &llms.FunctionCall{
						Name:      msg.ToolCall.FunctionCall.Name,
						Arguments: msg.ToolCall.FunctionCall.Arguments,
					},
				})

				lcMessages = append(lcMessages, llms.MessageContent{
					Role:  role,
					Parts: parts,
				})
			} else {
				// Regular assistant message with just text
				lcMessages = append(lcMessages, llms.TextParts(role, msg.Content))
			}
		default:
			// Handles RoleUser and RoleSystem as plain text.
			lcMessages = append(lcMessages, llms.TextParts(role, msg.Content))
		}
	}

	var callOpts []llms.CallOption
	if len(cfg.Tools) > 0 {
		callOpts = append(callOpts, llms.WithTools(cfg.Tools))
	}

	if cfg.ToolChoice != nil {
		callOpts = append(callOpts, llms.WithToolChoice(cfg.ToolChoice))
	}

	// Call the model
	res, err := a.client.GenerateContent(ctx, lcMessages, callOpts...)
	if err != nil {
		return nil, fmt.Errorf("langchaingo chat failed: %w", err)
	}

	if res == nil || len(res.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from LLM")
	}

	rawResponseJSON, jsonErr := json.MarshalIndent(res, "", "  ")
	if jsonErr != nil {
		logger.Log.WithError(jsonErr).Warn("Could not marshal raw LLM response for debugging")
	} else {
		logger.Log.WithField("raw_llm_response", string(rawResponseJSON)).Info("Received raw response from LLM")
	}

	// Every choice carries the usage of the whole message: count it once.
	usage := usageFrom(res.Choices[0].GenerationInfo, a.cfg)
	if rec := RecorderFrom(ctx); rec != nil {
		rec.Add(usage)
	}
	return responseFrom(res.Choices, usage), nil
}

// responseFrom reads a model's answer. OpenAI returns one choice with the
// text and the tool calls; langchaingo's Anthropic client returns one choice
// per content block, so "I'll summarize it." and the tool call it announces
// arrive as two choices. The text of every choice is kept, and the first tool
// call, wherever it is, makes the answer a tool call.
func responseFrom(choices []*llms.ContentChoice, usage Usage) *Response {
	var texts []string
	var toolCall *llms.ToolCall
	for _, c := range choices {
		if c == nil {
			continue
		}
		if c.Content != "" {
			texts = append(texts, c.Content)
		}
		if toolCall == nil && len(c.ToolCalls) > 0 {
			toolCall = &c.ToolCalls[0]
		}
	}
	content := strings.Join(texts, "\n\n")
	if toolCall != nil {
		return &Response{Type: ResponseTypeToolCall, Content: content, ToolCall: toolCall, Usage: usage}
	}
	return &Response{Type: ResponseTypeText, Content: content, Usage: usage}
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

var (
	defaultsMu sync.RWMutex
	defaults   Config
)

// SetDefaultConfig sets the instance-wide LLM settings (the `llm` block of
// the configuration). Fields an agent leaves empty fall back to them.
func SetDefaultConfig(cfg Config) {
	defaultsMu.Lock()
	defer defaultsMu.Unlock()
	defaults = cfg
}

func withDefaults(cfg Config) Config {
	defaultsMu.RLock()
	d := defaults
	defaultsMu.RUnlock()
	if cfg.Provider == "" {
		cfg.Provider = d.Provider
		// Model, key and endpoint only make sense for the same provider.
		if cfg.Model == "" {
			cfg.Model = d.Model
		}
		if cfg.APIKey == "" {
			cfg.APIKey = d.APIKey
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = d.BaseURL
		}
	} else if cfg.Provider == d.Provider {
		if cfg.Model == "" {
			cfg.Model = d.Model
		}
		if cfg.APIKey == "" {
			cfg.APIKey = d.APIKey
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = d.BaseURL
		}
	}
	// Prices of the default model apply when the model is the default one
	// and no price was given.
	if cfg.InputCostPerMillion == 0 && cfg.OutputCostPerMillion == 0 &&
		cfg.Provider == d.Provider && cfg.Model == d.Model {
		cfg.InputCostPerMillion, cfg.OutputCostPerMillion = d.InputCostPerMillion, d.OutputCostPerMillion
	}
	return cfg
}

// NewAdapter creates a new `Client` by wrapping the specified LLM provider.
// Empty fields are filled from SetDefaultConfig.
func NewAdapter(cfg Config) (Client, error) {
	cfg = withDefaults(cfg)
	var client llms.Model
	var err error
	switch cfg.Provider {
	case "openai":
		client, err = openai_client.New(cfg.APIKey, cfg.Model, cfg.BaseURL)
	case "anthropic":
		client, err = anthropic_client.New(cfg.APIKey, cfg.Model, cfg.BaseURL)
	case "ollama":
		// Ollama is typically  run locally and uses a model name
		client, err = ollama_client.New(cfg.Model, cfg.BaseURL)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create client for provider %s: %w", cfg.Provider, err)
	}

	return &langChainAdapter{client: client, cfg: cfg}, nil
}

// DefaultProvider returns the instance-wide default provider and model
// (llm.provider, llm.model), for presets such as agent templates.
func DefaultProvider() (provider, model string) {
	defaultsMu.RLock()
	defer defaultsMu.RUnlock()
	return defaults.Provider, defaults.Model
}
