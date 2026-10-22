package migrations

import (
	"time"

	"gorm.io/gorm"
)

// Migration 0007 adds connections: other services (remote MCP servers)
// whose tools agents can use.

type connectionTool0007 struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	ReadOnly    bool           `json:"readOnly"`
	Destructive bool           `json:"destructive,omitempty"`
}

type connection0007 struct {
	ID             uint                 `gorm:"primaryKey"`
	Name           string               `gorm:"size:64;uniqueIndex:idx_connections_name;not null"`
	Label          string               `gorm:"size:191"`
	Preset         string               `gorm:"size:64"`
	URL            string               `gorm:"type:text;not null"`
	AuthType       string               `gorm:"size:16;not null"`
	HeaderName     string               `gorm:"size:191"`
	Secret         string               `gorm:"type:text"`
	Tools          []connectionTool0007 `gorm:"type:json;serializer:json"`
	ToolsUpdatedAt *time.Time
	LastError      string `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (connection0007) TableName() string { return "connections" }

func connectionsUp(tx *gorm.DB) error   { return tx.AutoMigrate(&connection0007{}) }
func connectionsDown(tx *gorm.DB) error { return tx.Migrator().DropTable("connections") }
