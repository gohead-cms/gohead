package storage

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingQueue keeps the collection events storage enqueues.
type recordingQueue struct {
	mu     sync.Mutex
	events []events.EventType
}

func (q *recordingQueue) Enqueue(_ context.Context, taskType string, payload []byte, _ ...queue.Option) (string, error) {
	if taskType == events.TaskTypeCollectionEvent {
		var p events.CollectionEventPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return "", err
		}
		q.mu.Lock()
		q.events = append(q.events, p.EventType)
		q.mu.Unlock()
	}
	return "1", nil
}

func (q *recordingQueue) Close() error { return nil }

func (q *recordingQueue) take() []events.EventType {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := q.events
	q.events = nil
	return out
}

// A save that changes nothing sends no item:updated event: two agents that
// react to each other's updates (a summarizer and a tagger on the same
// collection) would otherwise trigger each other forever.
func TestUpdateItemContent_NoEventWhenNothingChanges(t *testing.T) {
	ct := historyFixture(t)
	q := &recordingQueue{}
	InitQueue(q)
	t.Cleanup(func() { InitQueue(nil) })

	agent := Actor{Kind: models.ActorAgent, Name: "tagger", AgentRunID: 1}
	item, err := SaveItem(agent, ct, models.JSONMap{"title": "Hello"})
	require.NoError(t, err)
	assert.Equal(t, []events.EventType{events.EventTypeItemCreated}, q.take())

	_, err = UpdateItemContent(agent, ct, item.ID, models.JSONMap{"title": "Hello"}, nil)
	require.NoError(t, err)
	assert.Empty(t, q.take(), "an unchanged save is not an update")

	_, err = UpdateItemContent(agent, ct, item.ID, models.JSONMap{"title": "Hello again"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []events.EventType{events.EventTypeItemUpdated}, q.take())
}

// Changes an agent makes while it runs send no events, in every process
// (gohead standalone included), so agents can't trigger each other in loops.
func TestAgentRunChanges_SendNoEvents(t *testing.T) {
	ct := historyFixture(t)
	q := &recordingQueue{}
	InitQueue(q)
	t.Cleanup(func() { InitQueue(nil) })

	run := Actor{Kind: models.ActorAgent, Name: "summarizer", AgentRunID: 3, InRun: true}
	item, err := SaveItem(run, ct, models.JSONMap{"title": "Draft"})
	require.NoError(t, err)
	_, err = UpdateItemContent(run, ct, item.ID, models.JSONMap{"title": "Summarized"}, nil)
	require.NoError(t, err)
	require.NoError(t, DeleteItem(run, item.ID))
	assert.Empty(t, q.take())

	// The same agent's change applied from an approved proposal is not in a run.
	approved := Actor{Kind: models.ActorAgent, Name: "summarizer", AgentRunID: 3}
	_, err = SaveItem(approved, ct, models.JSONMap{"title": "Approved"})
	require.NoError(t, err)
	assert.Equal(t, []events.EventType{events.EventTypeItemCreated}, q.take())
}
