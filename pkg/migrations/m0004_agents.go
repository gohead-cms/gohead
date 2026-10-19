package migrations

import (
	"time"

	"github.com/gohead-cms/gohead/internal/models"

	"gorm.io/gorm"
)

// proposal0004 is the proposals table as migration 0004 creates it.
type proposal0004 struct {
	ID          uint           `gorm:"primaryKey"`
	AgentName   string         `gorm:"size:191;index:idx_proposals_agent_name"`
	AgentRunID  *uint          `gorm:"index:idx_proposals_agent_run_id"`
	Action      string         `gorm:"size:16;not null"`
	Collection  string         `gorm:"size:191;not null;index:idx_proposals_collection"`
	ItemID      *uint          `gorm:"index:idx_proposals_item_id"`
	Locale      string         `gorm:"size:35"`
	Data        models.JSONMap `gorm:"type:json"`
	Reason      string         `gorm:"type:text"`
	BaseVersion uint
	Status      string    `gorm:"size:16;not null;index:idx_proposals_status"`
	ReviewedBy  string    `gorm:"size:191"`
	ReviewNote  string    `gorm:"type:text"`
	Error       string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"index:idx_proposals_created_at"`
	DecidedAt   *time.Time
}

func (proposal0004) TableName() string { return "proposals" }

type agent0004 struct {
	RequireApproval bool `gorm:"not null;default:false"`
}

func (agent0004) TableName() string { return "agents" }

type agentRun0004 struct {
	DryRun bool `gorm:"not null;default:false"`
}

func (agentRun0004) TableName() string { return "agent_runs" }

func agentsV2Up(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&proposal0004{}); err != nil {
		return err
	}
	if !tx.Migrator().HasColumn(&agent0004{}, "RequireApproval") {
		if err := tx.Migrator().AddColumn(&agent0004{}, "RequireApproval"); err != nil {
			return err
		}
	}
	if !tx.Migrator().HasColumn(&agentRun0004{}, "DryRun") {
		if err := tx.Migrator().AddColumn(&agentRun0004{}, "DryRun"); err != nil {
			return err
		}
	}
	return nil
}

func agentsV2Down(tx *gorm.DB) error {
	if tx.Migrator().HasColumn(&agentRun0004{}, "DryRun") {
		if err := tx.Migrator().DropColumn(&agentRun0004{}, "DryRun"); err != nil {
			return err
		}
	}
	if tx.Migrator().HasColumn(&agent0004{}, "RequireApproval") {
		if err := tx.Migrator().DropColumn(&agent0004{}, "RequireApproval"); err != nil {
			return err
		}
	}
	return tx.Migrator().DropTable(&proposal0004{})
}
