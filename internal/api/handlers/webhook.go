package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/gohead-cms/gohead/internal/agent/jobs"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// asynqClient is a package-level variable to hold the client instance.
var asynqClient *asynq.Client

// InitAsynqClient initializes the trigger package with a shared Asynq client.
// This function should be called once when your server starts.
func InitAsynqClient(client *asynq.Client) {
	asynqClient = client
}

// HandleWebhook is the Gin handler for incoming agent webhook requests.
func HandleWebhook(c *gin.Context) {
	// 1. Parse Agent ID from the URL path.
	agentIDStr := c.Param("id")
	agentID, err := strconv.ParseUint(agentIDStr, 10, 32)
	if err != nil {
		c.Set("status", http.StatusBadRequest)
		c.Set("response", gin.H{"error": "Invalid agent ID format"})
		return
	}

	// 2. Retrieve the agent's configuration from storage.
	agent, err := storage.GetAgentByID(uint(agentID))
	if err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Warn("Webhook handler could not find agent")
		c.Set("status", http.StatusNotFound)
		c.Set("response", gin.H{"error": "Agent not found"})
		return
	}

	// 3. Authenticate the request using the webhook token.
	requestToken := c.GetHeader("Webhook-Token")
	if requestToken != agent.Trigger.WebhookToken {
		logger.Log.WithField("agent_id", agentID).Warn("Invalid webhook token received")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", gin.H{"error": "Unauthorized"})
		return
	}

	// 4. Bind the incoming JSON body.
	var requestData map[string]any
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.Set("status", http.StatusBadRequest)
		c.Set("response", gin.H{"error": "Invalid JSON payload"})
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
	if err := jobs.EnqueueAgentJob(context.Background(), asynqClient, payload); err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to enqueue webhook job")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", gin.H{"error": "Failed to process webhook"})
		return
	}

	// 8. Respond immediately with "202 Accepted".
	c.Set("status", http.StatusAccepted)
	c.Set("response", gin.H{"status": "Job accepted for processing"})
}
