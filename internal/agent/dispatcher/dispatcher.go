package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/agent/jobs"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/hibiken/asynq"
)

// EventDispatcher listens for generic events and enqueues specific agent jobs.
type EventDispatcher struct {
	asynqClient *asynq.Client
}

// NewEventDispatcher creates a new dispatcher instance.
func NewEventDispatcher(client *asynq.Client) *EventDispatcher {
	return &EventDispatcher{asynqClient: client}
}

// HandleCollectionEvent is the handler for the "events:collection" queue.
func (d *EventDispatcher) HandleCollectionEvent(ctx context.Context, t *asynq.Task) error {
	var payload events.CollectionEventPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("could not unmarshal event payload: %w", err)
	}

	logger.Log.WithField("event", payload).Info("Processing collection event")

	// 1. Find all agents subscribed to this event.
	subscribedAgents, err := storage.FindAgentsByEventTrigger(payload.CollectionName, payload.EventType)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to find subscribed agents")
		return err // Asynq will retry this job
	}

	if len(subscribedAgents) == 0 {
		logger.Log.Info("No agents subscribed to this event.")
		return nil // No error, just no work to do
	}

	logger.Log.Infof("Found %d agent(s) subscribed to this event.", len(subscribedAgents))

	// 2. For each subscribed agent, enqueue an `agent:run` job.
	for _, agent := range subscribedAgents {
		jobPayload := jobs.AgentJobPayload{
			AgentID:      agent.ID,
			TriggerType:  "collection_event",
			InitialInput: fmt.Sprintf("An event '%s' occurred for item %d in collection '%s'. Please process the provided data.", payload.EventType, payload.ItemID, payload.CollectionName),
			TriggerData:  payload.ItemData,
		}

		if err := jobs.EnqueueAgentJob(ctx, d.asynqClient, jobPayload); err != nil {
			logger.Log.WithError(err).WithField("agent_id", agent.ID).Error("Failed to enqueue agent job from event")
			// We continue to the next agent even if one fails.
		}
	}

	return nil
}
