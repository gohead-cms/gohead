package migrations

import (
	"time"

	"github.com/gohead-cms/gohead/internal/models"
)

// revision0002 is the revisions table as migration 0002 creates it. It is a
// copy of models.Revision at that time, so later model changes need their
// own migration instead of silently changing this one.
type revision0002 struct {
	ID           uint           `gorm:"primaryKey"`
	EntityType   string         `gorm:"size:16;not null;uniqueIndex:idx_revisions_entity_version,priority:1"`
	EntityID     uint           `gorm:"not null;uniqueIndex:idx_revisions_entity_version,priority:2"`
	Version      uint           `gorm:"not null;uniqueIndex:idx_revisions_entity_version,priority:3"`
	TypeName     string         `gorm:"size:191;index:idx_revisions_type_name"`
	Action       string         `gorm:"size:16;not null"`
	Status       string         `gorm:"size:16"`
	Data         models.JSONMap `gorm:"type:json"`
	ActorKind    string         `gorm:"size:16;not null"`
	ActorName    string         `gorm:"size:191"`
	AgentRunID   *uint          `gorm:"index:idx_revisions_agent_run_id"`
	RestoredFrom *uint
	CreatedAt    time.Time `gorm:"index:idx_revisions_created_at"`
}

func (revision0002) TableName() string { return "revisions" }
