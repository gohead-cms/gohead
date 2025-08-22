package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gohead-cms/gohead/internal/agents"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AgentRecord: flat GORM model that stores the Agent config as JSON blobs.
// Keeps schema stable while Agent struct evolves.
type AgentRecord struct {
	ID        string `gorm:"primaryKey;size:128"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name         string `gorm:"size:256;not null"`
	Description  string `gorm:"size:1024"`
	Enabled      bool   `gorm:"index"`
	SystemPrompt string `gorm:"type:text"`

	ProviderJSON  datatypes.JSON `gorm:"type:jsonb"`
	MemoryJSON    datatypes.JSON `gorm:"type:jsonb"`
	FunctionsJSON datatypes.JSON `gorm:"type:jsonb"`
	TriggersJSON  datatypes.JSON `gorm:"type:jsonb"`

	MaxTurns int
}

func (AgentRecord) TableName() string { return "agents" }

func AutoMigrateAgentStore() error {
	return database.DB.AutoMigrate(&AgentRecord{})
}

// ToRecord converts runtime Agent to DB record.
func ToRecord(a *agents.Agent) (*AgentRecord, error) {
	pj, err := json.Marshal(a.Provider)
	if err != nil {
		return nil, err
	}
	mj, err := json.Marshal(a.Memory)
	if err != nil {
		return nil, err
	}
	fj, err := json.Marshal(a.Functions)
	if err != nil {
		return nil, err
	}
	tj, err := json.Marshal(a.Triggers)
	if err != nil {
		return nil, err
	}
	return &AgentRecord{
		ID:            a.ID,
		Name:          a.Name,
		Description:   a.Description,
		Enabled:       a.Enabled,
		SystemPrompt:  a.SystemPrompt,
		ProviderJSON:  pj,
		MemoryJSON:    mj,
		FunctionsJSON: fj,
		TriggersJSON:  tj,
		MaxTurns:      a.MaxTurns,
	}, nil
}

// FromRecord converts DB record to runtime Agent.
func FromRecord(r *AgentRecord) (*agents.Agent, error) {
	var prov agents.ProviderConfig
	var mem agents.MemoryConfig
	var funcs []agents.FunctionSpec
	var trig agents.TriggerSpec

	if err := json.Unmarshal(r.ProviderJSON, &prov); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(r.MemoryJSON, &mem); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(r.FunctionsJSON, &funcs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(r.TriggersJSON, &trig); err != nil {
		return nil, err
	}
	return &agents.Agent{
		ID:           r.ID,
		Name:         r.Name,
		Description:  r.Description,
		Enabled:      r.Enabled,
		SystemPrompt: r.SystemPrompt,
		Provider:     prov,
		Memory:       mem,
		Functions:    funcs,
		Triggers:     trig,
		MaxTurns:     r.MaxTurns,
	}, nil
}

// UpsertAgent persists (create/update) an Agent config.
func UpsertAgent(ctx context.Context, a *agents.Agent) error {
	rec, err := ToRecord(a)
	if err != nil {
		return err
	}
	tx := database.DB.WithContext(ctx)
	if err := tx.Clauses(
	// GORM upsert (ON CONFLICT DO UPDATE) — works on PG/MySQL/SQLite appropriately
	).Save(rec).Error; err != nil {
		logger.Log.WithError(err).WithField("agent_id", a.ID).Error("failed to upsert agent")
		return err
	}
	return nil
}

// LoadAllAgents returns all Agent configs from DB.
func LoadAllAgents(ctx context.Context) ([]*agents.Agent, error) {
	var recs []AgentRecord
	if err := database.DB.WithContext(ctx).Find(&recs).Error; err != nil {
		return nil, err
	}
	out := make([]*agents.Agent, 0, len(recs))
	for _, r := range recs {
		a, err := FromRecord(&r)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}
