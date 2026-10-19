package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gohead-cms/gohead/internal/agent/functions"
	"github.com/gohead-cms/gohead/internal/agent/jobs"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// Me returns the authenticated caller: a user, or an API token.
func Me(c *gin.Context) {
	username := c.GetString("username")
	role := c.GetString("role")
	if name, ok := strings.CutPrefix(username, "token:"); ok {
		c.Set("response", gin.H{"kind": "token", "username": username, "name": name, "role": role})
		c.Set("status", http.StatusOK)
		return
	}
	user, err := storage.GetUserByUsername(username)
	if err != nil {
		c.Set("response", "User not found")
		c.Set("status", http.StatusNotFound)
		return
	}
	c.Set("response", gin.H{
		"kind": "user", "id": user.ID, "username": user.Username, "email": user.Email, "role": user.Role.Name,
	})
	c.Set("status", http.StatusOK)
}

// GetOverview returns counts and recent activity for the admin dashboard.
func GetOverview(c *gin.Context) {
	o, err := storage.GetOverview(8)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to build the overview")
		c.Set("response", "Failed to load the overview")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", gin.H{"version": APIVersion, "overview": o})
	c.Set("status", http.StatusOK)
}

// ListAllAgentRuns lists the runs of every agent, newest first: page,
// pageSize (≤ 100) and status.
func ListAllAgentRuns(c *gin.Context) {
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
	runs, total, err := storage.ListAllRuns(q)
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

// AgentFunctionCatalog lists the built-in functions agents can use.
func AgentFunctionCatalog(c *gin.Context) {
	c.Set("response", functions.Catalog())
	c.Set("status", http.StatusOK)
}

// RunAgentNow queues a manual run of an agent, with an optional "input"
// message. The run appears in the agent's run history.
func RunAgentNow(c *gin.Context) {
	agent, err := storage.GetAgentByName(c.Param("name"))
	if err != nil {
		c.Set("response", "Agent not found")
		c.Set("status", http.StatusNotFound)
		return
	}
	var body struct {
		Input string `json:"input"`
	}
	_ = c.ShouldBindJSON(&body) // the body is optional
	input := strings.TrimSpace(body.Input)
	if input == "" {
		input = "This is a manual run started from the admin UI. Do your task."
	}
	if asynqClient == nil {
		c.Set("response", "Agent runs need Redis, which is not configured")
		c.Set("status", http.StatusServiceUnavailable)
		return
	}
	if err := jobs.EnqueueAgentJob(context.Background(), asynqClient, jobs.AgentJobPayload{
		AgentID: agent.ID, InitialInput: input,
	}); err != nil {
		logger.Log.WithError(err).WithField("agent", agent.Name).Error("Failed to queue a manual run")
		c.Set("response", "Failed to queue the run")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", gin.H{"status": "queued"})
	c.Set("status", http.StatusAccepted)
}
