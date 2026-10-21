package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gohead-cms/gohead/internal/agent/tools"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/robfig/cron"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Agent represents an autonomous agent configuration.
type Agent struct {
	gorm.Model
	Name         string        `json:"name" gorm:"uniqueIndex;size:191"`
	SystemPrompt string        `json:"system_prompt" gorm:"type:text;not null"`
	MaxTurns     int           `json:"max_turns" gorm:"not null;default:4"`
	LLMConfig    LLMConfig     `json:"llm_config" gorm:"type:jsonb"`
	Memory       MemoryConfig  `json:"memory" gorm:"type:jsonb"`
	Trigger      TriggerConfig `json:"trigger" gorm:"type:jsonb"`
	Functions    FunctionSpecs `json:"functions" gorm:"type:jsonb"`
	Limits       Limits        `json:"limits" gorm:"type:json;serializer:json"`
	// RequireApproval makes content writes proposals that an editor
	// approves or rejects, instead of changes.
	RequireApproval bool           `json:"require_approval" gorm:"not null;default:false"`
	Config          models.JSONMap `json:"-" gorm:"type:jsonb"`
}

// LLMConfig specifies the large language model to use.
type LLMConfig struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	BaseURL   string `json:"base_url,omitempty"` // optional API endpoint override
	// Prices in USD per million tokens (for cost tracking and limits).
	InputCostPerMillion  float64 `json:"input_cost_per_million,omitempty"`
	OutputCostPerMillion float64 `json:"output_cost_per_million,omitempty"`
}

// Value implements the Valuer interface for `LLMConfig`.
func (c LLMConfig) Value() (driver.Value, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLMConfig: %w", err)
	}
	return string(data), nil // MySQL refuses binary strings in JSON columns
}

// Scan implements the Scanner interface for `LLMConfig`.
func (c *LLMConfig) Scan(value any) error {
	bytes, err := models.JSONBytes(value)
	if err != nil {
		return errors.New("invalid data type for LLMConfig")
	}
	return json.Unmarshal(bytes, c)
}

// GormDBDataType makes the column JSON on MySQL.
func (LLMConfig) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return models.JSONDBDataType(db, field)
}

// MemoryConfig defines how the conversation memory is stored.
type MemoryConfig struct {
	Type         string `json:"type"` // e.g., "in-memory", "postgres"
	SessionScope string `json:"session_scope"`
}

// Value implements the Valuer interface for `MemoryConfig`.
func (c MemoryConfig) Value() (driver.Value, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MemoryConfig: %w", err)
	}
	return string(data), nil
}

// Scan implements the Scanner interface for `MemoryConfig`.
func (c *MemoryConfig) Scan(value interface{}) error {
	bytes, err := models.JSONBytes(value)
	if err != nil {
		return errors.New("invalid data type for MemoryConfig")
	}
	return json.Unmarshal(bytes, c)
}

// GormDBDataType makes the column JSON on MySQL.
func (MemoryConfig) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return models.JSONDBDataType(db, field)
}

// EventTriggerConfig defines the specifics for a collection event trigger.
type EventTriggerConfig struct {
	Collection string   `json:"collection"` // The name of the collection to listen to.
	Events     []string `json:"events"`     // e.g., ["item:created", "item:updated"]
}

// TriggerConfig defines what initiates an agent's run.
type TriggerConfig struct {
	Type         string             `json:"type"` // e.g., "manual", "cron", "webhook", "collection_event"
	Cron         string             `json:"cron,omitempty"`
	WebhookToken string             `json:"webhook_token,omitempty"`
	EventTrigger EventTriggerConfig `json:"event_trigger,omitempty"`
}

// Value implements the Valuer interface for `TriggerConfig`.
// It marshals the struct into JSON for storage in the database.
func (t TriggerConfig) Value() (driver.Value, error) {
	// If the type is empty, we assume it's an uninitialized struct and save null.
	if t.Type == "" {
		return nil, nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// GormDBDataType makes the column JSON on MySQL.
func (TriggerConfig) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return models.JSONDBDataType(db, field)
}

// Scan implements the Scanner interface for `TriggerConfig`.
// This version is now robust and handles both []byte and string from the database.
func (t *TriggerConfig) Scan(value any) error {
	if value == nil {
		*t = TriggerConfig{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("invalid data type for TriggerConfig")
	}

	return json.Unmarshal(bytes, t)
}

// FunctionSpec is a tool of an agent, as stored. Name, Description and
// Parameters are optional: empty, they come from the tool catalog (see
// Resolved). Agents written with the short form ("tools") store only what
// they override.
type FunctionSpec struct {
	Name        string         `json:"name"`        // The name the model sees
	Description string         `json:"description"` // What the model is told about it
	Parameters  models.JSONMap `json:"parameters"`  // JSON schema of its arguments
	ImplKey     string         `json:"impl_key"`    // The built-in implementation
	// Collections restricts the collections.* functions to these
	// collections. Empty: every collection (not recommended for writes).
	Collections []string `json:"collections,omitempty"`
}

type FunctionSpecs []FunctionSpec

func (f FunctionSpecs) Value() (driver.Value, error) {
	if len(f) == 0 {
		return "[]", nil // Return an empty JSON array if the slice is empty
	}
	b, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// GormDBDataType makes the column JSON on MySQL.
func (FunctionSpecs) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return models.JSONDBDataType(db, field)
}

// Scan implements sql.Scanner for FunctionSpecs, accepting either a JSON array
// of FunctionSpec or a single JSON object (which will be wrapped into a slice).
func (f *FunctionSpecs) Scan(value any) error {
	if value == nil {
		*f = nil
		return nil
	}

	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *models.FunctionSpecs", value)
	}

	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		*f = FunctionSpecs{}
		return nil
	}

	switch s[0] {
	case '[':
		// Proper slice
		return json.Unmarshal(raw, f)
	case '{':
		// Single object stored in DB — wrap into slice for backward/dirty compatibility
		var one FunctionSpec
		if err := json.Unmarshal(raw, &one); err != nil {
			return fmt.Errorf("unmarshal FunctionSpec object: %w", err)
		}
		*f = FunctionSpecs{one}
		return nil
	default:
		return errors.New("functions column is neither JSON object nor array")
	}
}

// ParseAgentInput parses a generic map into a typed Agent struct.
// It handles potential data type mismatches gracefully and then validates the schema.
func ParseAgentInput(input map[string]any) (Agent, error) {
	var agent Agent
	b, err := json.Marshal(input)
	if err != nil {
		return agent, err
	}
	if err := json.Unmarshal(b, &agent); err != nil {
		return agent, fmt.Errorf("invalid agent input format: %w", err)
	}
	// "tools" (the short form) wins over "functions" when both are given,
	// as when a client sends back what it read.
	if specs, ok, err := toolsInput(input); err != nil {
		return agent, err
	} else if ok {
		if _, both := input["functions"]; both && !sameTools(specs, agent.Functions) {
			return agent, errors.New("'tools' and 'functions' describe different tools: send one of them ('tools' is the short form)")
		}
		agent.Functions = specs
	}
	for i := range agent.Functions {
		// The long form may name tools by their short name too.
		agent.Functions[i].ImplKey = tools.ImplKey(strings.TrimSpace(agent.Functions[i].ImplKey))
	}

	if err := ValidateAgentSchema(agent); err != nil {
		return agent, err
	}

	return agent, nil
}

// ValidateAgentSchema ensures that the agent's configuration is valid.
func ValidateAgentSchema(agent Agent) error {
	// 1. Basic required fields
	if agent.Name == "" {
		return errors.New("agent must have a 'name'")
	}
	if agent.SystemPrompt == "" {
		return errors.New("agent must have a 'system_prompt'")
	}
	if agent.MaxTurns <= 0 {
		return errors.New("max_turns must be a positive integer")
	}

	// 2. LLM Configuration
	if agent.LLMConfig.Provider == "" {
		return errors.New("llm_config provider cannot be empty")
	}

	// 3. Memory Configuration
	if agent.Memory.Type == "" {
		return errors.New("memory type cannot be empty")
	}
	if agent.Memory.SessionScope == "" {
		return errors.New("memory session scope cannot be empty")
	}

	// 4. Trigger Configuration
	switch agent.Trigger.Type {
	case "manual":
		// No additional checks needed
	case "cron":
		if agent.Trigger.Cron == "" {
			return errors.New("cron trigger requires a cron expression")
		}
		if _, err := cron.ParseStandard(agent.Trigger.Cron); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	case "webhook":
		if agent.Trigger.WebhookToken == "" {
			return errors.New("webhook trigger requires a 'webhook_token'")
		}
	case "collection_event":
		if agent.Trigger.EventTrigger.Collection == "" {
			return errors.New("event trigger requires a 'collection' name")
		}
		if len(agent.Trigger.EventTrigger.Events) == 0 {
			return errors.New("event trigger requires at least one event type (e.g., 'item:created')")
		}
	default:
		return errors.New("invalid trigger type: must be 'manual', 'cron', 'webhook' or 'collection_event'")
	}

	// 5. Functions
	if err := validateFunctionSpecs(agent.Functions); err != nil {
		return err
	}

	// 6. Limits
	l := agent.Limits
	if l.MaxRunsPerHour < 0 || l.MaxRunsPerDay < 0 || l.MaxTokensPerRun < 0 || l.MaxTokensPerDay < 0 || l.MaxCostPerDay < 0 {
		return errors.New("limits cannot be negative")
	}
	if l.MaxCostPerDay > 0 && agent.LLMConfig.InputCostPerMillion == 0 && agent.LLMConfig.OutputCostPerMillion == 0 {
		return errors.New("limits.max_cost_per_day needs prices in llm_config (input_cost_per_million, output_cost_per_million)")
	}
	if agent.LLMConfig.InputCostPerMillion < 0 || agent.LLMConfig.OutputCostPerMillion < 0 {
		return errors.New("llm_config prices cannot be negative")
	}

	return nil
}

// validateFunctionSpecs checks an agent's tools: each names an
// implementation, the names the model sees are unique and valid, and only
// content tools have collections. Names, descriptions and parameters left
// out come from the catalog.
func validateFunctionSpecs(funcs []FunctionSpec) error {
	seen := make(map[string]struct{})
	for i, f := range funcs {
		if strings.TrimSpace(f.ImplKey) == "" {
			return fmt.Errorf("tool %d: say which tool with 'use' (e.g. \"content.update\")", i+1)
		}
		r := f.Resolved()
		if !toolName.MatchString(r.Name) {
			return fmt.Errorf("tool '%s': names may only use letters, digits, '_' and '-' (64 at most)", r.Name)
		}
		if _, dup := seen[r.Name]; dup {
			return fmt.Errorf("two tools are named '%s': give one of them its own 'name'", r.Name)
		}
		seen[r.Name] = struct{}{}
		if f.Parameters != nil {
			// Sent to the model as a JSON schema: it must serialize.
			if _, err := json.Marshal(f.Parameters); err != nil {
				return fmt.Errorf("tool '%s' has invalid parameters: %v", r.Name, err)
			}
		}
		if _, known := tools.Lookup(r.ImplKey); known && len(f.Collections) > 0 && !tools.Scoped(r.ImplKey) {
			return fmt.Errorf("tool '%s': 'on' (collections) only applies to content tools", r.Name)
		}
		for _, c := range f.Collections {
			if strings.TrimSpace(c) == "" {
				return fmt.Errorf("tool '%s': empty collection name in 'on'", r.Name)
			}
		}
	}
	return nil
}
