package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/gohead-cms/gohead/internal/models"

	"gorm.io/gorm"
)

// Agent represents an autonomous agent configuration.
// It is stored as a JSONB object in the database.
// Agent represents an autonomous agent configuration.
// It is stored as a JSONB object in the database.
type Agent struct {
	gorm.Model
	Name         string         `json:"name" gorm:"uniqueIndex"`
	SystemPrompt string         `json:"system_prompt" gorm:"type:text;not null"`
	MaxTurns     int            `json:"max_turns" gorm:"not null;default:4"`
	LLMConfig    LLMConfig      `json:"llm_config" gorm:"type:jsonb"`
	Memory       MemoryConfig   `json:"memory" gorm:"type:jsonb"`
	Trigger      TriggerConfig  `json:"trigger" gorm:"type:jsonb"`
	Functions    []FunctionSpec `json:"functions" gorm:"type:jsonb"`
	Config       models.JSONMap `json:"-" gorm:"type:jsonb"`
}

// LLMConfig specifies the large language model to use.
type LLMConfig struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// Value implements the Valuer interface for `LLMConfig`.
func (c LLMConfig) Value() (driver.Value, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLMConfig: %w", err)
	}
	return data, nil
}

// Scan implements the Scanner interface for `LLMConfig`.
func (c *LLMConfig) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid data type for LLMConfig")
	}
	return json.Unmarshal(bytes, c)
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
	return data, nil
}

// Scan implements the Scanner interface for `MemoryConfig`.
func (c *MemoryConfig) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid data type for MemoryConfig")
	}
	return json.Unmarshal(bytes, c)
}

// TriggerConfig defines what initiates an agent's run.
type TriggerConfig struct {
	Type         string `json:"type"` // e.g., "manual", "cron", "webhook"
	Cron         string `json:"cron"`
	WebhookToken string `json:"webhook_token"`
}

// Value implements the Valuer interface for `TriggerConfig`.
// It marshals the struct into JSON for storage in the database.
func (t TriggerConfig) Value() (driver.Value, error) {
	if t.Cron == "" && t.WebhookToken == "" {
		return nil, nil
	}
	data, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TriggerConfig: %w", err)
	}
	return data, nil
}

// Scan implements the Scanner interface for `TriggerConfig`.
// It unmarshals the JSON data from the database into the struct.
func (t *TriggerConfig) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid data type for TriggerConfig")
	}

	return json.Unmarshal(bytes, t)
}

// FunctionSpec describes a tool the agent can call.
// FunctionSpec represents a tool or function the agent can use.
type FunctionSpec struct {
	Name        string         `json:"name"`        // The name of the function
	Description string         `json:"description"` // A description for the LLM
	Parameters  models.JSONMap `json:"parameters"`  // JSON schema for the function's parameters
	ImplKey     string         `json:"impl_key"`    // Key used to look up the actual implementation
}

// Value implements the Valuer interface for `FunctionSpec`.
func (c FunctionSpec) Value() (driver.Value, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FunctionSpec: %w", err)
	}
	return data, nil
}

// Scan implements the Scanner interface for `FunctionSpec`.
func (c *FunctionSpec) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid data type for FunctionSpec")
	}
	return json.Unmarshal(bytes, c)
}

var cronRegex = regexp.MustCompile(`^(?:@(annually|yearly|monthly|weekly|daily|hourly|reboot)|((\d+,)+\d+|(\d+-\d+)|\d+|\*)?( (\d+,)+\d+|(\d+-\d+)|\d+|\*)?( (\d+,)+\d+|(\d+-\d+)|\d+|\*)?( (\d+,)+\d+|(\d+-\d+)|\d+|\*)?( (\d+,)+\d+|(\d+-\d+)|\d+|\*)?( (\d+,)+\d+|(\d+-\d+)|\d+|\*)?( (\d+,)+\d+|(\d+-\d+)|\d+|\*)?)$`)

// ParseAgentInput parses a generic map into a typed Agent struct.
// It handles potential data type mismatches gracefully.
func ParseAgentInput(input map[string]any) (Agent, error) {
	var agent Agent
	b, err := json.Marshal(input)
	if err != nil {
		return agent, err
	}
	if err := json.Unmarshal(b, &agent); err != nil {
		return agent, fmt.Errorf("invalid agent input format: %w", err)
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
		// Check for an empty string first
		if agent.Trigger.Cron == "" {
			return errors.New("cron trigger requires a cron expression")
		}
		// Now check if the expression matches the regex
		if !cronRegex.MatchString(agent.Trigger.Cron) {
			return errors.New("invalid cron expression")
		}
	case "webhook":
		if agent.Trigger.WebhookToken == "" {
			return errors.New("webhook trigger requires a 'webhook_token'")
		}
	default:
		return errors.New("invalid trigger type: must be 'manual', 'cron', or 'webhook'")
	}

	// 5. Functions
	if err := validateFunctionSpecs(agent.Functions); err != nil {
		return err
	}

	return nil
}

// validateFunctionSpecs detects duplicates first, then validates required fields and JSON.
// This ordering makes tests expecting duplicate errors pass even if other fields could be invalid.
func validateFunctionSpecs(funcs []FunctionSpec) error {
	// ---- First pass: duplicate detection (based on non-empty values only)
	seenImpl := make(map[string]struct{})
	seenName := make(map[string]struct{})
	for _, f := range funcs {
		if f.ImplKey != "" {
			if _, ok := seenImpl[f.ImplKey]; ok {
				return errors.New("duplicate function implementation key")
			}
			seenImpl[f.ImplKey] = struct{}{}
		}
		if f.Name != "" {
			if _, ok := seenName[f.Name]; ok {
				return errors.New("duplicate function name")
			}
			seenName[f.Name] = struct{}{}
		}
	}

	// ---- Second pass: required fields + JSON validity
	for _, f := range funcs {
		if f.Name == "" {
			return errors.New("function must have a non-empty 'name'")
		}
		if f.ImplKey == "" {
			return errors.New("function must have a non-empty 'impl_key'")
		}
		// Parameters presence
		switch p := any(f.Parameters).(type) {
		case string:
			if p == "" {
				return errors.New("function parameters cannot be empty")
			}
			if !json.Valid([]byte(p)) {
				return errors.New("invalid JSON in function parameters")
			}
		case json.RawMessage:
			if len(p) == 0 {
				return errors.New("function parameters cannot be empty")
			}
			if !json.Valid(p) {
				return errors.New("invalid JSON in function parameters")
			}
		default:
			// If your struct defines Parameters as string, this won't happen; keep a safe fallback.
			b, _ := json.Marshal(f.Parameters)
			if len(b) == 0 {
				return errors.New("function parameters cannot be empty")
			}
			if !json.Valid(b) {
				return errors.New("invalid JSON in function parameters")
			}
		}
	}

	return nil
}
