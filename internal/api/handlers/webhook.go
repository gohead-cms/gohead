package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gohead-cms/gohead/pkg/queue"

	"github.com/gohead-cms/gohead/internal/agent/jobs"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// queueClient queues agent runs (webhook triggers, manual runs).
var queueClient queue.Client

// InitQueue sets the queue client of the handlers.
func InitQueue(client queue.Client) {
	queueClient = client
}

// HandleWebhook is the Gin handler for incoming agent webhook requests.
func HandleWebhook(c *gin.Context) {
	// 1. Parse Agent ID from the URL path.
	agentIDStr := c.Param("id")
	agentID, err := strconv.ParseUint(agentIDStr, 10, 32)
	if err != nil {
		c.Set("status", http.StatusBadRequest)
		c.Set("response", "Invalid agent ID format")
		return
	}

	// 2. Retrieve the agent's configuration from storage.
	agent, err := storage.GetAgentByID(uint(agentID))
	if err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Warn("Webhook handler could not find agent")
		c.Set("status", http.StatusNotFound)
		c.Set("response", "Agent not found")
		return
	}

	// 3. Authenticate the request using the webhook token.
	// Only webhook-triggered agents with a configured token can be invoked,
	// and the token is compared in constant time.
	requestToken := c.GetHeader("Webhook-Token")
	expected := agent.Trigger.WebhookToken
	if agent.Trigger.Type != "webhook" || expected == "" ||
		subtle.ConstantTimeCompare([]byte(requestToken), []byte(expected)) != 1 {
		logger.Log.WithField("agent_id", agentID).Warn("Invalid webhook token received")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", "Unauthorized")
		return
	}

	// 4. Bind the incoming JSON body.
	var requestData map[string]any
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.Set("status", http.StatusBadRequest)
		c.Set("response", "Invalid JSON payload")
		return
	}

	// 5. Create the initial input prompt for the LLM.
	jsonData, _ := json.MarshalIndent(requestData, "", "  ")
	initialInput := fmt.Sprintf("A webhook was triggered with the following data. Please process it according to your instructions.\n\nData:\n%s", string(jsonData))

	// 6. Create the job payload.
	payload := jobs.AgentJobPayload{
		AgentID:      uint(agentID),
		InitialInput: initialInput,
	}

	// 7. Enqueue the job for the worker.
	if err := jobs.EnqueueAgentJob(context.Background(), queueClient, payload); err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to enqueue webhook job")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", "Failed to process webhook")
		return
	}

	// 8. Respond immediately with "202 Accepted".
	c.Set("status", http.StatusAccepted)
	c.Set("response", gin.H{"status": "Job accepted for processing"})
}
