package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/hibiken/asynq"
)

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
