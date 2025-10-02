package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/hibiken/asynq"
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
)

// CollectionEventPayload is the generic data structure for a collection event.
// This is sent from the storage layer to the dispatcher worker.
type CollectionEventPayload struct {
	EventType      EventType      `json:"event_type"`
	CollectionName string         `json:"collection_name"`
	ItemID         uint           `json:"item_id"`
	ItemData       map[string]any `json:"item_data"` // The full data of the item
}

// EnqueueCollectionEvent creates and enqueues a new generic collection event.
func EnqueueCollectionEvent(ctx context.Context, client *asynq.Client, payload CollectionEventPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to marshal collection event payload")
		return fmt.Errorf("could not marshal event payload: %w", err)
	}

	// Specify that this task should go to the "events" queue.
	task := asynq.NewTask(TaskTypeCollectionEvent, payloadBytes, asynq.Queue("events"))

	info, err := client.EnqueueContext(ctx, task)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to enqueue collection event")
		return fmt.Errorf("could not enqueue event task: %w", err)
	}

	logger.Log.
		WithField("job_id", info.ID).
		WithField("event_type", payload.EventType).
		WithField("collection", payload.CollectionName).
		Info("Successfully enqueued collection event")

	return nil
}
