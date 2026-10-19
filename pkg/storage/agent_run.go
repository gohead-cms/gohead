package storage

import (
	"errors"
	"time"

	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// ErrRunNotFound is returned for unknown runs.
var ErrRunNotFound = errors.New("run not found")

// CreateAgentRun records the start of a run.
func CreateAgentRun(r *agents.AgentRun) error {
	return database.DB.Create(r).Error
}

// SaveAgentRun stores the run's current state (usage, status, output). It
// only updates: a run deleted meanwhile (agent deleted) is not re-created.
func SaveAgentRun(r *agents.AgentRun) error {
	return database.DB.Model(&agents.AgentRun{ID: r.ID}).Select("*").Omit("ID", "Steps").Updates(r).Error
}

// AddAgentRunStep records one step of a run.
func AddAgentRunStep(s *agents.AgentRunStep) {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	if err := database.DB.Create(s).Error; err != nil {
		logger.Log.WithError(err).Warn("Failed to record agent run step")
	}
}

// AgentUsage summarizes an agent's runs in a time window.
type AgentUsage struct {
	Runs         int     `json:"runs"` // runs that called the model (limited runs excluded)
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUsd"`
}

// Tokens is input plus output tokens.
func (u AgentUsage) Tokens() int { return u.InputTokens + u.OutputTokens }

// AgentUsageSince sums an agent's runs started after since. Runs skipped
// by a limit do not count as runs, but tokens and cost spent by runs that
// a limit stopped midway do. excludeRun (or 0) is left out.
func AgentUsageSince(agentID uint, since time.Time, excludeRun uint) (AgentUsage, error) {
	q := database.DB.Model(&agents.AgentRun{}).Where("agent_id = ? AND started_at >= ?", agentID, since.UTC())
	if excludeRun != 0 {
		q = q.Where("id <> ?", excludeRun)
	}
	return sumUsage(q)
}

// AgentUsageBefore is AgentUsageSince over the runs created before run
// beforeRun, which orders runs starting at the same time.
// With wasQueued (a run recorded when it was requested and started later),
// runs started since that are not queued count too.
func AgentUsageBefore(agentID uint, since time.Time, beforeRun uint, wasQueued bool) (AgentUsage, error) {
	q := database.DB.Model(&agents.AgentRun{}).Where("agent_id = ? AND started_at >= ?", agentID, since.UTC())
	if wasQueued {
		q = q.Where("id <> ? AND (id < ? OR status <> ?)", beforeRun, beforeRun, agents.RunQueued)
	} else {
		q = q.Where("id < ?", beforeRun)
	}
	return sumUsage(q)
}

// FailQueuedRun marks a run that never started as failed.
func FailQueuedRun(id uint, reason string) {
	now := time.Now().UTC()
	if err := database.DB.Model(&agents.AgentRun{}).Where("id = ? AND status = ?", id, agents.RunQueued).
		Updates(map[string]any{"status": agents.RunFailed, "error": reason, "finished_at": now}).Error; err != nil {
		logger.Log.WithError(err).WithField("run_id", id).Error("Failed to mark a queued run failed")
	}
}

func sumUsage(q *gorm.DB) (AgentUsage, error) {
	var u AgentUsage
	var row struct {
		Runs         int64
		InputTokens  int64
		OutputTokens int64
		CostUSD      float64
	}
	err := q.Select("COALESCE(SUM(CASE WHEN status <> ? THEN 1 ELSE 0 END),0) AS runs, "+
		"COALESCE(SUM(input_tokens),0) AS input_tokens, COALESCE(SUM(output_tokens),0) AS output_tokens, "+
		"COALESCE(SUM(cost_usd),0) AS cost_usd", agents.RunLimited).Scan(&row).Error
	if err != nil {
		return u, err
	}
	u.Runs, u.InputTokens, u.OutputTokens, u.CostUSD = int(row.Runs), int(row.InputTokens), int(row.OutputTokens), row.CostUSD
	return u, nil
}

// RunQuery filters an agent's runs.
type RunQuery struct {
	Page, PageSize int
	Status         string
}

// ListAgentRuns returns an agent's runs, newest first (without steps).
func ListAgentRuns(agentID uint, q RunQuery) ([]agents.AgentRun, int, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	base := database.DB.Model(&agents.AgentRun{}).Where("agent_id = ?", agentID)
	if q.Status != "" {
		base = base.Where("status = ?", q.Status)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []agents.AgentRun
	err := base.Session(&gorm.Session{}).Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&out).Error
	return out, int(total), err
}

// GetAgentRun returns one run of an agent with its steps in order.
func GetAgentRun(agentID, runID uint) (*agents.AgentRun, error) {
	var r agents.AgentRun
	err := database.DB.Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Where("id = ? AND agent_id = ?", runID, agentID).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunNotFound
	}
	return &r, err
}

// PruneAgentRuns keeps the newest keep runs of an agent (and their steps).
// Runs still in progress are never pruned.
func PruneAgentRuns(agentID uint, keep int) {
	if keep <= 0 {
		return
	}
	var cutoff []agents.AgentRun
	if err := database.DB.Select("id").Where("agent_id = ?", agentID).Order("id DESC").
		Offset(keep).Limit(1).Find(&cutoff).Error; err != nil || len(cutoff) == 0 {
		return
	}
	last := cutoff[0].ID
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		old := "agent_id = ? AND id <= ? AND status NOT IN ?"
		live := []string{agents.RunRunning, agents.RunQueued}
		sub := tx.Model(&agents.AgentRun{}).Select("id").Where(old, agentID, last, live)
		if err := tx.Where("run_id IN (?)", sub).Delete(&agents.AgentRunStep{}).Error; err != nil {
			return err
		}
		return tx.Where(old, agentID, last, live).Delete(&agents.AgentRun{}).Error
	})
	if err != nil {
		logger.Log.WithError(err).Warn("Failed to prune agent runs")
	}
}

// DeleteAgentRuns removes all runs of an agent.
func DeleteAgentRuns(agentID uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		sub := tx.Model(&agents.AgentRun{}).Select("id").Where("agent_id = ?", agentID)
		if err := tx.Where("run_id IN (?)", sub).Delete(&agents.AgentRunStep{}).Error; err != nil {
			return err
		}
		return tx.Where("agent_id = ?", agentID).Delete(&agents.AgentRun{}).Error
	})
}
