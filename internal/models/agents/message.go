package models

import (
	"github.com/gohead-cms/gohead/pkg/llm"
	"gorm.io/gorm"
)

// AgentMessage stores a single message in an agent's conversation history.
type AgentMessage struct {
	gorm.Model
	AgentID uint     `gorm:"index"` // Foreign key to the Agent
	Role    llm.Role `gorm:"type:varchar(20)"`
	Content string   `gorm:"type:text"`
	Turn    int      // The order of the message in the conversation
}
