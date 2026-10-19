package storage_test

import (
	"testing"
	"time"

	models "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueuedRuns(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, db.AutoMigrate(&models.AgentRun{}))
	now := time.Now().UTC()
	queued := &models.AgentRun{AgentID: 1, Status: models.RunQueued, StartedAt: now}
	require.NoError(t, storage.CreateAgentRun(queued))
	later := &models.AgentRun{AgentID: 1, Status: models.RunSucceeded, StartedAt: now}
	require.NoError(t, storage.CreateAgentRun(later))

	u, err := storage.AgentUsageBefore(1, now.Add(-time.Hour), queued.ID, false)
	require.NoError(t, err)
	assert.Zero(t, u.Runs)
	u, err = storage.AgentUsageBefore(1, now.Add(-time.Hour), queued.ID, true)
	require.NoError(t, err)
	assert.Equal(t, 1, u.Runs, "a queued run also counts the runs started while it waited")

	storage.FailQueuedRun(queued.ID, "agent deleted")
	storage.FailQueuedRun(later.ID, "ignored: not queued")
	var runs []models.AgentRun
	require.NoError(t, db.Order("id").Find(&runs).Error)
	assert.Equal(t, models.RunFailed, runs[0].Status)
	assert.Equal(t, "agent deleted", runs[0].Error)
	assert.NotNil(t, runs[0].FinishedAt)
	assert.Equal(t, models.RunSucceeded, runs[1].Status)
}
