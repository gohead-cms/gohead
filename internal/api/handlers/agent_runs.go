package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

func runAgent(c *gin.Context) (*agents.Agent, bool) {
	agent, err := storage.GetAgentByName(c.Param("name"))
	if err != nil {
		c.Set("response", "Agent not found")
		c.Set("status", http.StatusNotFound)
		return nil, false
	}
	return agent, true
}

// ListAgentRuns lists an agent's runs, newest first: page, pageSize
// (≤ 100) and status (running, succeeded, failed, limited).
func ListAgentRuns(c *gin.Context) {
	agent, ok := runAgent(c)
	if !ok {
		return
	}
	q := storage.RunQuery{Page: 1, PageSize: 20, Status: c.Query("status")}
	switch q.Status {
	case "", agents.RunRunning, agents.RunSucceeded, agents.RunFailed, agents.RunLimited:
	default:
		validationError(c, "status must be running, succeeded, failed or limited")
		return
	}
	if v := c.Query("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			validationError(c, "invalid page")
			return
		}
		q.Page = p
	}
	if v := c.Query("pageSize"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			validationError(c, "pageSize must be between 1 and 100")
			return
		}
		q.PageSize = ps
	}
	runs, total, err := storage.ListAgentRuns(agent.ID, q)
	if err != nil {
		c.Set("response", "Failed to list runs")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", runs)
	c.Set("meta", gin.H{"pagination": gin.H{
		"page": q.Page, "pageSize": q.PageSize, "total": total,
		"pageCount": (total + q.PageSize - 1) / q.PageSize,
	}})
	c.Set("status", http.StatusOK)
}

// GetAgentRun returns one run with its steps.
func GetAgentRun(c *gin.Context) {
	agent, ok := runAgent(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("run"), 10, 32)
	if err != nil || id == 0 {
		validationError(c, "invalid run ID")
		return
	}
	run, err := storage.GetAgentRun(agent.ID, uint(id))
	if err != nil {
		if errors.Is(err, storage.ErrRunNotFound) {
			c.Set("response", "Run not found")
			c.Set("status", http.StatusNotFound)
			return
		}
		c.Set("response", "Failed to load run")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", run)
	c.Set("status", http.StatusOK)
}

// GetAgentUsage reports an agent's usage over the last hour and day next
// to its limits.
func GetAgentUsage(c *gin.Context) {
	agent, ok := runAgent(c)
	if !ok {
		return
	}
	now := time.Now().UTC() // runs are stored in UTC
	hour, err1 := storage.AgentUsageSince(agent.ID, now.Add(-time.Hour), 0)
	day, err2 := storage.AgentUsageSince(agent.ID, now.Add(-24*time.Hour), 0)
	if err1 != nil || err2 != nil {
		c.Set("response", "Failed to compute usage")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", gin.H{
		"lastHour":  hour,
		"lastDay":   day,
		"limits":    agent.Limits,
		"costKnown": agent.LLMConfig.InputCostPerMillion > 0 || agent.LLMConfig.OutputCostPerMillion > 0,
	})
	c.Set("status", http.StatusOK)
}
