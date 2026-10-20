package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"
)

// Task type and queue of agent runs.
const (
	TaskTypeAgentRun = "agent:run"
	QueueAgents      = "agents"
)

// EnqueueAgentJob creates and enqueues a new agent job.
// This is the single entry point for all triggers.
func EnqueueAgentJob(ctx context.Context, client queue.Client, payload AgentJobPayload) error {
	if client == nil {
		return fmt.Errorf("cannot enqueue agent job: queue client is not configured")
	}
	// 1. Marshal the payload into JSON bytes.
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to marshal agent job payload")
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	// 2. Queue it: "agent:run" tasks are run by the worker.
	id, err := client.Enqueue(ctx, TaskTypeAgentRun, payloadBytes, queue.Queue(QueueAgents))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to enqueue agent job")
		return fmt.Errorf("could not enqueue task: %w", err)
	}

	logger.Log.
		WithField("job_id", id).
		WithField("agent_id", payload.AgentID).
		Info("Successfully enqueued agent job")

	return nil
}
