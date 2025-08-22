package memory

import (
	"time"

	"gorm.io/gorm"
)

// AgentMessage is a durable row per chat message for an agent session.
type AgentMessage struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	AgentID   string `gorm:"index;size:128;not null"`
	SessionID string `gorm:"index;size:128;not null"`

	Role    string `gorm:"size:32;not null"`
	Name    string `gorm:"size:128"`
	Content string `gorm:"type:text;not null"`
}
