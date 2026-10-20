package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"
)

// EventType defines the type of collection event that occurred.
type EventType string

// Using a constant prevents typos between the producer and the consumer.
const TaskTypeCollectionEvent = "events:collection"

const (
	EventTypeCollectionCreated EventType = "collection:created"
	EventTypeCollectionUpdated EventType = "collection:updated"
	EventTypeCollectionDeleted EventType = "collection:deleted"
	EventTypeItemCreated       EventType = "item:created"
	EventTypeItemUpdated       EventType = "item:updated"
	EventTypeItemDeleted       EventType = "item:deleted"
	// Draft-and-publish collections only: an item became visible to readers,
	// or was withdrawn.
	EventTypeItemPublished   EventType = "item:published"
	EventTypeItemUnpublished EventType = "item:unpublished"

	// Singleton content events. Their CollectionName is
	// "singleton:<name>", like the singleton's permission key.
	EventTypeSingletonUpdated EventType = "singleton:updated"
	EventTypeSingletonDeleted EventType = "singleton:deleted"
)

// CollectionEventPayload is the generic data structure for a collection event.
// This is sent from the storage layer to the dispatcher worker.
type CollectionEventPayload struct {
	EventType      EventType      `json:"event_type"`
	CollectionName string         `json:"collection_name"`
	ItemID         uint           `json:"item_id"`
	ItemData       map[string]any `json:"item_data"` // The full data of the item
	// ItemTranslations holds localized values in other locales than the
	// default, by locale.
	ItemTranslations map[string]any `json:"item_translations,omitempty"`
	// ActorKind and ActorName say who made the change (models.ActorUser,
	// ActorAgent…): an agent is not triggered by its own changes.
	ActorKind string `json:"actor_kind,omitempty"`
	ActorName string `json:"actor_name,omitempty"`
}

// QueueEvents is the queue of collection events, processed by the dispatcher.
const QueueEvents = "events"

// EnqueueCollectionEvent creates and enqueues a new generic collection event.
// When no queue client is configured (e.g. in tests or a deployment without
// Redis), the event is dropped with a debug log instead of panicking.
func EnqueueCollectionEvent(ctx context.Context, client queue.Client, payload CollectionEventPayload) error {
	if client == nil {
		logger.Log.WithField("event_type", payload.EventType).Debug("No queue client configured; skipping collection event")
		return nil
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to marshal collection event payload")
		return fmt.Errorf("could not marshal event payload: %w", err)
	}

	id, err := client.Enqueue(ctx, TaskTypeCollectionEvent, payloadBytes, queue.Queue(QueueEvents))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to enqueue collection event")
		return fmt.Errorf("could not enqueue event task: %w", err)
	}

	logger.Log.
		WithField("job_id", id).
		WithField("event_type", payload.EventType).
		WithField("collection", payload.CollectionName).
		Info("Successfully enqueued collection event")

	return nil
}
