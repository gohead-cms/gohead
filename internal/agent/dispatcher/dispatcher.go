package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gohead-cms/gohead/internal/models"
	"strconv"
	"strings"
	"time"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/agent/jobs"
	"github.com/gohead-cms/gohead/internal/agent/triggers"
	"github.com/gohead-cms/gohead/internal/webhooks"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// EventDispatcher listens for generic collection events and dispatches
// specific agent jobs to the appropriate queue.
type EventDispatcher struct {
	queue queue.Client
}

// NewEventDispatcher creates a new dispatcher instance.
// It requires a queue client to enqueue the agent:run jobs and webhook
// deliveries.
func NewEventDispatcher(client queue.Client) *EventDispatcher {
	return &EventDispatcher{queue: client}
}

// HandleCollectionEvent is the entry point for processing generic collection events.
func (d *EventDispatcher) HandleCollectionEvent(ctx context.Context, data []byte) error {
	var payload events.CollectionEventPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		logger.Log.WithError(err).Error("Failed to unmarshal collection event payload")
		return fmt.Errorf("could not unmarshal event payload: %w", err)
	}

	if strings.HasPrefix(string(payload.EventType), "collection:") {
		logger.Log.Info("Collection schema event detected, triggering GraphQL schema hot reload.")
		go triggers.TriggerSchemaReload()
	}

	logger.Log.
		WithField("event_type", payload.EventType).
		WithField("collection", payload.CollectionName).
		WithField("item_id", payload.ItemID).
		Info("Processing collection event")

	// 0. Outbound webhooks: one queued delivery per subscribed webhook. A
	// failure retries the whole event; deliveries are deduplicated by ID.
	if err := webhooks.Fanout(ctx, d.queue, payload); err != nil {
		logger.Log.WithError(err).Error("Failed to enqueue webhook deliveries")
		return err
	}

	// 1. Find all agents subscribed to this specific event.
	subscribedAgents, err := storage.FindAgentsByEventTrigger(payload.CollectionName, string(payload.EventType))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to find agents for event trigger")
		return err // Return the error so Asynq can retry the job.
	}

	if len(subscribedAgents) == 0 {
		logger.Log.
			WithField("event_type", payload.EventType).
			WithField("collection", payload.CollectionName).
			Info("No agents subscribed to this event")
		return nil // No error, just no work to do.
	}

	logger.Log.Infof("Found %d agents subscribed to event '%s' on collection '%s'", len(subscribedAgents), payload.EventType, payload.CollectionName)

	// 2. For each subscribed agent, enqueue a specific agent:run job.
	for _, agent := range subscribedAgents {
		// An agent is not triggered by its own changes: an agent that
		// updates the items whose updates trigger it would loop.
		if SelfTriggered(payload, agent.Name) {
			logger.Log.WithField("agent_name", agent.Name).WithField("item_id", payload.ItemID).
				Info("Skipping an event caused by the agent itself")
			continue
		}
		// Create structured TriggerEvent - this contains all the data the runner needs
		triggerEvent := &jobs.TriggerEvent{
			Type: "collection_event",
			CollectionEvent: &jobs.CollectionEventData{
				Collection: payload.CollectionName,
				Event:      string(payload.EventType),
				ItemID:     strconv.FormatUint(uint64(payload.ItemID), 10),
				ItemData:   payload.ItemData,
			},
		}

		// Clean payload - only essential fields
		agentJobPayload := jobs.AgentJobPayload{
			AgentID:      agent.ID,
			TriggerEvent: triggerEvent, // All event data is here
			CreatedAt:    time.Now(),
		}

		logger.Log.WithFields(map[string]any{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
			"event_type": payload.EventType,
			"collection": payload.CollectionName,
			"item_id":    payload.ItemID,
		}).Info("Enqueuing agent job with structured trigger event")

		if err := jobs.EnqueueAgentJob(ctx, d.queue, agentJobPayload); err != nil {
			logger.Log.
				WithError(err).
				WithField("agent_id", agent.ID).
				WithField("agent_name", agent.Name).
				Error("Failed to enqueue agent job from dispatcher")
			// We continue to the next agent even if one fails.
		} else {
			logger.Log.WithField("agent_id", agent.ID).Info("Successfully enqueued agent job")
		}
	}

	return nil
}

// SelfTriggered reports whether the event comes from a change the agent
// made itself.
func SelfTriggered(payload events.CollectionEventPayload, agentName string) bool {
	return payload.ActorKind == models.ActorAgent && payload.ActorName == agentName
}
