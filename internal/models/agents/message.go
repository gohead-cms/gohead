package models

import (
	"github.com/gohead-cms/gohead/pkg/llm"
	"github.com/tmc/langchaingo/llms"
	"gorm.io/gorm"
)

// DBToolCall represents a simplified tool call for database storage
type DBToolCall struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	FunctionName string `json:"function_name"`
	Arguments    string `json:"arguments"`
}

// AgentMessage stores a single message in an agent's conversation history.
type AgentMessage struct {
	gorm.Model
	AgentID    uint        `gorm:"index"` // Foreign key to the Agent
	Role       llm.Role    `gorm:"type:varchar(20)"`
	Content    string      `gorm:"type:text"`
	ToolCallID string      `gorm:"type:varchar(255)" json:"tool_call_id,omitempty"`
	ToolCall   *DBToolCall `gorm:"serializer:json"`
	Turn       int         // The order of the message in the conversation
}

// ToLangChainToolCall converts a DBToolCall to llms.ToolCall.
func (d *DBToolCall) ToLangChainToolCall() *llms.ToolCall {
	if d == nil {
		return nil
	}

	// llms.ToolCall expects a FunctionCall with Name and Arguments.
	// Some versions of langchaingo use a struct; others a pointer.
	// We construct it defensively.
	fc := llms.FunctionCall{
		Name:      d.FunctionName,
		Arguments: d.Arguments,
	}

	tc := &llms.ToolCall{
		ID:           d.ID,
		Type:         d.Type,
		FunctionCall: &fc,
	}
	return tc
}

// FromLangChainToolCall creates DBToolCall from llms.ToolCall
func FromLangChainToolCall(tc *llms.ToolCall) *DBToolCall {
	if tc == nil {
		return nil
	}
	return &DBToolCall{
		ID:           tc.ID,
		Type:         tc.Type,
		FunctionName: tc.FunctionCall.Name,
		Arguments:    tc.FunctionCall.Arguments,
	}
}
