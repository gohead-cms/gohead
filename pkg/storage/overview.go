package storage

import (
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/database"
)

// Overview summarizes an instance for the admin dashboard.
type Overview struct {
	Counts struct {
		Collections int64 `json:"collections"`
		Singletons  int64 `json:"singletons"`
		Components  int64 `json:"components"`
		Items       int64 `json:"items"`
		Media       int64 `json:"media"`
		MediaBytes  int64 `json:"mediaBytes"`
		Users       int64 `json:"users"`
		Agents      int64 `json:"agents"`
		Webhooks    int64 `json:"webhooks"`
		APITokens   int64 `json:"apiTokens"`
	} `json:"counts"`
	// Agent runs of the last 24 hours.
	Runs struct {
		Total        int64   `json:"total"`
		Succeeded    int64   `json:"succeeded"`
		Failed       int64   `json:"failed"`
		Limited      int64   `json:"limited"`
		Running      int64   `json:"running"`
		InputTokens  int64   `json:"inputTokens"`
		OutputTokens int64   `json:"outputTokens"`
		CostUSD      float64 `json:"costUsd"`
	} `json:"runs24h"`
	// Webhook deliveries of the last 24 hours.
	Deliveries struct {
		Total  int64 `json:"total"`
		Failed int64 `json:"failed"`
	} `json:"deliveries24h"`
	RecentItems      []RecentItem             `json:"recentItems"`
	RecentRuns       []agents.AgentRun        `json:"recentRuns"`
	RecentDeliveries []models.WebhookDelivery `json:"recentDeliveries"`
}

// RecentItem is a recently changed item.
type RecentItem struct {
	ID         uint      `json:"id"`
	Collection string    `json:"collection"`
	Title      string    `json:"title"` // first text value, for display
	UpdatedAt  time.Time `json:"updatedAt"`
}

// GetOverview counts the content model, content, users and integrations,
// and lists recent activity.
func GetOverview(recent int) (*Overview, error) {
	db := database.DB
	o := &Overview{}
	counts := []struct {
		model any
		dst   *int64
	}{
		{&models.Collection{}, &o.Counts.Collections},
		{&models.Singleton{}, &o.Counts.Singletons},
		{&models.Component{}, &o.Counts.Components},
		{&models.Item{}, &o.Counts.Items},
		{&models.Media{}, &o.Counts.Media},
		{&models.User{}, &o.Counts.Users},
		{&agents.Agent{}, &o.Counts.Agents},
		{&models.Webhook{}, &o.Counts.Webhooks},
		{&models.APIToken{}, &o.Counts.APITokens},
	}
	for _, c := range counts {
		if err := db.Model(c.model).Count(c.dst).Error; err != nil {
			return nil, err
		}
	}
	if err := db.Model(&models.Media{}).Select("COALESCE(SUM(size),0)").Scan(&o.Counts.MediaBytes).Error; err != nil {
		return nil, err
	}

	since := time.Now().UTC().Add(-24 * time.Hour)
	var byStatus []struct {
		Status       string
		N            int64
		InputTokens  int64
		OutputTokens int64
		CostUSD      float64
	}
	if err := db.Model(&agents.AgentRun{}).Where("started_at >= ?", since).
		Select("status, COUNT(*) AS n, COALESCE(SUM(input_tokens),0) AS input_tokens, " +
			"COALESCE(SUM(output_tokens),0) AS output_tokens, COALESCE(SUM(cost_usd),0) AS cost_usd").
		Group("status").Scan(&byStatus).Error; err != nil {
		return nil, err
	}
	for _, s := range byStatus {
		o.Runs.Total += s.N
		o.Runs.InputTokens += s.InputTokens
		o.Runs.OutputTokens += s.OutputTokens
		o.Runs.CostUSD += s.CostUSD
		switch s.Status {
		case agents.RunSucceeded:
			o.Runs.Succeeded = s.N
		case agents.RunFailed:
			o.Runs.Failed = s.N
		case agents.RunLimited:
			o.Runs.Limited = s.N
		case agents.RunRunning:
			o.Runs.Running = s.N
		}
	}
	deliveries := db.Model(&models.WebhookDelivery{}).Where("created_at >= ?", since)
	if err := deliveries.Count(&o.Deliveries.Total).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.WebhookDelivery{}).Where("created_at >= ?", since).
		Where("status_code < 200 OR status_code >= 300").Count(&o.Deliveries.Failed).Error; err != nil {
		return nil, err
	}

	var items []models.Item
	if err := db.Order("updated_at DESC").Limit(recent).Find(&items).Error; err != nil {
		return nil, err
	}
	names := map[uint]string{}
	var cols []models.Collection
	if err := db.Select("id", "name").Find(&cols).Error; err != nil {
		return nil, err
	}
	for _, c := range cols {
		names[c.ID] = c.Name
	}
	o.RecentItems = make([]RecentItem, 0, len(items))
	for _, it := range items {
		o.RecentItems = append(o.RecentItems, RecentItem{
			ID: it.ID, Collection: names[it.CollectionID], Title: displayTitle(it.Data), UpdatedAt: it.UpdatedAt,
		})
	}
	o.RecentRuns = []agents.AgentRun{}
	if err := db.Order("id DESC").Limit(recent).Find(&o.RecentRuns).Error; err != nil {
		return nil, err
	}
	o.RecentDeliveries = []models.WebhookDelivery{}
	if err := db.Order("id DESC").Limit(recent).Find(&o.RecentDeliveries).Error; err != nil {
		return nil, err
	}
	return o, nil
}

// displayTitle picks a readable label for an item: a title or name
// attribute, or else the first short text value.
func displayTitle(data models.JSONMap) string {
	for _, key := range []string{"title", "name", "label", "slug"} {
		if s, ok := data[key].(string); ok && s != "" {
			return truncateText(s, 80)
		}
	}
	for _, v := range data {
		if s, ok := v.(string); ok && s != "" && len(s) <= 120 {
			return truncateText(s, 80)
		}
	}
	return ""
}

func truncateText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// ListAllRuns lists the runs of every agent, newest first (without steps).
func ListAllRuns(q RunQuery) ([]agents.AgentRun, int, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	base := database.DB.Model(&agents.AgentRun{})
	if q.Status != "" {
		base = base.Where("status = ?", q.Status)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []agents.AgentRun
	q2 := database.DB.Model(&agents.AgentRun{})
	if q.Status != "" {
		q2 = q2.Where("status = ?", q.Status)
	}
	err := q2.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&out).Error
	return out, int(total), err
}
