package agents

import (
	"context"
	"encoding/json"
	"time"
)

// Core high-level model of an Agent.
// An Agent has:
//   - LLM provider config
//   - Memory backend selection
//   - A set of Functions (tools) it is allowed to call
//   - One or more Triggers (cron/webhook for MVP)
type Agent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`

	// SystemPrompt is prepended to conversations to guide behavior.
	SystemPrompt string `json:"system_prompt"`

	// Provider configuration (e.g., OpenAI model, API key ref)
	Provider ProviderConfig `json:"provider"`

	// Memory configuration (backend + logical keying strategy)
	Memory MemoryConfig `json:"memory"`

	// Functions that the agent can call.
	Functions []FunctionSpec `json:"functions"`

	// Triggers: e.g. Cron schedules; webhook token for auth
	Triggers TriggerSpec `json:"triggers"`

	// Safety: max turns to avoid infinite tool loops
	MaxTurns int `json:"max_turns"`
}

// ProviderConfig defines LLM specifics
type ProviderConfig struct {
	// Type: "openai" (MVP). Later: "vertex", "anthropic", "ollama", etc.
	Type string `json:"type"`

	// Model: e.g., "gpt-4o-mini", "gpt-4.1-mini"
	Model string `json:"model"`

	// APIKeyRef: how your app resolves secrets; keep string for MVP
	APIKeyRef string `json:"api_key_ref"`

	// Temperature, TopP, etc. kept minimal
	Temperature float32 `json:"temperature"`
	TopP        float32 `json:"top_p"`

	// ToolChoice: "auto" or "none" (MVP)
	ToolChoice string `json:"tool_choice"`
}

// MemoryConfig selects a memory backend and scope strategy
type MemoryConfig struct {
	// Backend: "bbolt" (MVP), "redis" (later)
	Backend string `json:"backend"`

	// Namespace to isolate agent data
	Namespace string `json:"namespace"`

	// How to group sessions: "per-agent", "per-target", "custom"
	SessionScope string `json:"session_scope"`
}

// FunctionSpec describes a tool the LLM can call.
type FunctionSpec struct {
	// Unique name used by the LLM to call the function
	Name string `json:"name"`

	// Human description shown to the model
	Description string `json:"description"`

	// JSONSchema for params (as raw JSON). MVP accepts any object.
	Parameters json.RawMessage `json:"parameters"`

	// Bindings for the actual implementation key
	// (e.g., "collections.upsert" maps to CollectionsUpsertFunction)
	ImplKey string `json:"impl_key"`
}

// TriggerSpec holds minimal triggers for MVP
type TriggerSpec struct {
	// Cron (e.g., "0 * * * *") — optional
	Cron string `json:"cron"`

	// Webhook: when non-empty, opens POST /agents/webhook/{id}?token=...
	WebhookToken string `json:"webhook_token"`
}

// Message holds one chat message in memory
type Message struct {
	Role    string `json:"role"`    // "system"|"user"|"assistant"|"tool"
	Name    string `json:"name"`    // function name if tool message
	Content string `json:"content"` // plain text or JSON for tool responses
}

// Event represents an incoming trigger firing
type Event struct {
	AgentID string          `json:"agent_id"`
	Kind    string          `json:"kind"`    // "cron"|"webhook"
	When    time.Time       `json:"when"`    // schedule time
	Payload json.RawMessage `json:"payload"` // body for webhook or scheduled args
}

// Provider abstracts the LLM
type Provider interface {
	Name() string
	// ChatTools supports tool calling. The provider must receive tool defs.
	ChatTools(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// ChatRequest/Response captures a tool-call capable turn
type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools"` // derived from Functions
	ToolChoice  string           `json:"tool_choice"`
	Temperature float32          `json:"temperature"`
	TopP        float32          `json:"top_p"`
	MaxTokens   int              `json:"max_tokens"`
}

// ToolDefinition aligns with provider's schema for functions
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ChatResponse struct {
	// If assistant text is present and no tool calls, Content is set
	Content string `json:"content"`

	// One or multiple tool calls can be requested
	ToolCalls []ToolCall `json:"tool_calls"`

	// Provider raw may be useful for telemetry
	Raw any `json:"raw"`
}

// ToolCall asks the host to invoke a specific function with JSON args
type ToolCall struct {
	ID        string          `json:"id"`
	FuncName  string          `json:"function_name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Memory abstracts conversation persistence
type Memory interface {
	Append(ctx context.Context, agentID, sessionID string, msg Message) error
	Load(ctx context.Context, agentID, sessionID string, limit int) ([]Message, error)
}

// AgentFunction (tool) executes a side effect or query and returns JSON
type AgentFunction interface {
	Name() string
	Description() string
	// JSONSchema for params (string-encoded)
	Parameters() json.RawMessage
	Call(ctx context.Context, args json.RawMessage) (string, error) // returns JSON string consumed by LLM
}

// FunctionRegistry maps ImplKey -> AgentFunction
type FunctionRegistry interface {
	Get(implKey string) (AgentFunction, bool)
	ToolsFromSpecs(funcs []FunctionSpec) ([]ToolDefinition, []AgentFunction, error)
}

// CollectionService is expected to be part of GoHead core; we reference a minimal interface here.
// Implement this in your app to allow agent functions to mutate/query collections.
type CollectionService interface {
	Upsert(ctx context.Context, collection string, id string, doc map[string]any) error
	Get(ctx context.Context, collection string, id string) (map[string]any, error)
	Query(ctx context.Context, collection string, filter map[string]any, limit int) ([]map[string]any, error)
}
