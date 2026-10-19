package dispatcher

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestSelfTriggered(t *testing.T) {
	ev := events.CollectionEventPayload{ActorKind: models.ActorAgent, ActorName: "tagger"}
	assert.True(t, SelfTriggered(ev, "tagger"))
	assert.False(t, SelfTriggered(ev, "summarizer"), "other agents still react")
	assert.False(t, SelfTriggered(events.CollectionEventPayload{ActorKind: models.ActorUser, ActorName: "tagger"}, "tagger"))
}
