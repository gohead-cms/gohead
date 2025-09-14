package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/hibiken/asynq"
)

// AgentJobPayload is the data structure for an agent execution job.
// It's the "contract" sent to the worker via Redis.
type AgentJobPayload struct {
	AgentID      uint           `json:"agent_id"`      // Changed from uuid.UUID to uint
	TriggerType  string         `json:"trigger_type"`  // e.g., "cron", "webhook"
	TriggerData  map[string]any `json:"trigger_data"`  // For webhook body, etc.
	InitialInput string         `json:"initial_input"` // The first message to the agent
}

// EnqueueAgentJob creates and enqueues a new agent job.
// This is the single entry point for all triggers.
func EnqueueAgentJob(ctx context.Context, client *asynq.Client, payload AgentJobPayload) error {
	// 1. Marshal the payload into JSON bytes.
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to marshal agent job payload")
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	// 2. Create a new Asynq task.
	// "agent:run" is the task type that your worker will listen for.
	task := asynq.NewTask("agent:run", payloadBytes, asynq.Queue("agents"))

	// 3. Enqueue the task.
	info, err := client.EnqueueContext(ctx, task)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to enqueue agent job")
		return fmt.Errorf("could not enqueue task: %w", err)
	}

	logger.Log.
		WithField("job_id", info.ID).
		WithField("agent_id", payload.AgentID).
		Info("Successfully enqueued agent job")

	return nil
}
