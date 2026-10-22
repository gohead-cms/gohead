package handlers

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gohead-cms/gohead/internal/agent/functions"
	"github.com/gohead-cms/gohead/internal/agent/jobs"
	"github.com/gohead-cms/gohead/internal/demo"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// Me returns the authenticated caller: a user, or an API token.
func Me(c *gin.Context) {
	username := c.GetString("username")
	role := c.GetString("role")
	if name, ok := strings.CutPrefix(username, "token:"); ok {
		c.Set("response", gin.H{"kind": "token", "username": username, "name": name, "role": role, "permissions": auth.PermissionsOf(role)})
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
		"permissions": auth.PermissionsOf(user.Role.Name),
		// The demo account sees the admin, read-only.
		"demo": demo.IsDemo(user.Role.Name),
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
	case "", agents.RunQueued, agents.RunRunning, agents.RunSucceeded, agents.RunFailed, agents.RunLimited:
	default:
		validationError(c, "status must be queued, running, succeeded, failed or limited")
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
	queueRun(c, agent, jobs.AgentJobPayload{AgentID: agent.ID, InitialInput: input})
}

// TestAgent queues a test run (dry run): the agent runs for real (model
// calls, reads), but content functions only validate and report the
// changes they would make. Body: {"input": "…"} for a manual message, or
// {"item_id": 42, "event": "item:created"} to simulate the agent's
// collection event on an existing item.
func TestAgent(c *gin.Context) {
	agent, err := storage.GetAgentByName(c.Param("name"))
	if err != nil {
		c.Set("response", "Agent not found")
		c.Set("status", http.StatusNotFound)
		return
	}
	var body struct {
		Input  string `json:"input"`
		ItemID uint   `json:"item_id"`
		Event  string `json:"event"`
	}
	if !bindOptional(c, &body) {
		return
	}
	if body.ItemID != 0 && strings.TrimSpace(body.Input) != "" {
		validationError(c, "give either input or item_id")
		return
	}
	payload := jobs.AgentJobPayload{AgentID: agent.ID, DryRun: true, InitialInput: strings.TrimSpace(body.Input)}
	if body.ItemID != 0 {
		collection := agent.Trigger.EventTrigger.Collection
		if agent.Trigger.Type != "collection_event" || collection == "" {
			validationError(c, "item_id simulates a collection event: this agent is not triggered by one")
			return
		}
		ct, err := storage.GetCollectionByName(collection)
		if err != nil {
			validationError(c, "the agent's collection '"+collection+"' does not exist")
			return
		}
		item, err := storage.GetItemByID(ct.ID, body.ItemID)
		if err != nil {
			validationError(c, fmt.Sprintf("item %d not found in %s", body.ItemID, collection))
			return
		}
		event := body.Event
		events := agent.Trigger.EventTrigger.Events
		if event == "" && len(events) > 0 {
			event = events[0]
		}
		if event == "" || !slices.Contains(events, event) {
			validationError(c, "event must be one of the agent's events: "+strings.Join(events, ", "))
			return
		}
		payload.TriggerEvent = &jobs.TriggerEvent{Type: "collection_event", CollectionEvent: &jobs.CollectionEventData{
			Collection: collection, Event: event, ItemID: fmt.Sprint(item.ID), ItemData: item.Data,
		}}
	} else if payload.InitialInput == "" {
		payload.InitialInput = "This is a test run started from the admin UI. Do your task."
	}
	queueRun(c, agent, payload)
}

// queueRun records a queued run and sends the job to the workers; the
// response carries the run ID to follow.
func queueRun(c *gin.Context, agent *agents.Agent, payload jobs.AgentJobPayload) {
	if queueClient == nil {
		c.Set("response", "Agent runs need the job queue, which is not configured")
		c.Set("status", http.StatusServiceUnavailable)
		return
	}
	run := &agents.AgentRun{
		AgentID: agent.ID, AgentName: agent.Name, Status: agents.RunQueued, DryRun: payload.DryRun,
		Provider: agent.LLMConfig.Provider, Model: agent.LLMConfig.Model, StartedAt: time.Now().UTC(),
		TriggerType: "manual",
	}
	if payload.TriggerEvent != nil {
		run.TriggerType = payload.TriggerEvent.Type
	}
	if err := storage.CreateAgentRun(run); err != nil {
		c.Set("response", "Failed to record the run")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	payload.RunID = run.ID
	if err := jobs.EnqueueAgentJob(context.Background(), queueClient, payload); err != nil {
		logger.Log.WithError(err).WithField("agent", agent.Name).Error("Failed to queue a run")
		run.Status, run.Error = agents.RunFailed, "could not be queued"
		_ = storage.SaveAgentRun(run)
		c.Set("response", "Could not queue the run: the job queue is unreachable")
		c.Set("status", http.StatusServiceUnavailable)
		return
	}
	c.Set("response", gin.H{"status": "queued", "runId": run.ID, "dryRun": payload.DryRun})
	c.Set("status", http.StatusAccepted)
}
