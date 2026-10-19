// jobs/payload.go - Updated AgentJobPayload to support both old and new formats

package jobs

import "time"

// AgentJobPayload is the data structure for an agent execution job.
// It's the "contract" sent to the worker via Redis.
type AgentJobPayload struct {
	AgentID      uint          `json:"agent_id"`                // Changed from uuid.UUID to uint
	InitialInput string        `json:"initial_input"`           // The first message to the agent
	TriggerEvent *TriggerEvent `json:"trigger_event,omitempty"` // Structured event data
	// DryRun: a test run; content functions only simulate their writes.
	DryRun bool `json:"dry_run,omitempty"`
	// RunID: the run recorded (queued) when the job was requested, so the
	// requester can follow it; 0 lets the worker record it.
	RunID     uint      `json:"run_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"` // When the job was created
}

// TriggerEvent represents a structured trigger event
type TriggerEvent struct {
	Type            string               `json:"type"` // "collection_event", "webhook", "schedule"
	CollectionEvent *CollectionEventData `json:"collection_event,omitempty"`
	WebhookData     map[string]any       `json:"webhook_data,omitempty"`
	ScheduleData    map[string]any       `json:"schedule_data,omitempty"`
}

// CollectionEventData represents a collection event
type CollectionEventData struct {
	Collection string         `json:"collection"` // "categories", "posts", etc.
	Event      string         `json:"event"`      // "item:created", "item:updated", etc.
	ItemID     string         `json:"item_id"`    // String version of the ID
	ItemData   map[string]any `json:"item_data"`  // The actual item data
}
