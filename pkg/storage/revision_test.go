package storage

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func historyFixture(t *testing.T) models.Collection {
	t.Helper()
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.Item{}, &models.Collection{}, &models.Attribute{}, &models.Revision{}))
	ct := &models.Collection{Name: "posts", Attributes: []models.Attribute{{Name: "title", Type: "string"}}}
	require.NoError(t, db.Create(ct).Error)
	return *ct
}

func versions(t *testing.T, id uint) []string {
	t.Helper()
	revs, _, err := ListRevisions(RevisionQuery{EntityType: models.RevisionEntityItem, EntityID: id, PageSize: 100})
	require.NoError(t, err)
	var out []string
	for _, r := range revs {
		out = append(out, r.Action)
	}
	return out
}

func TestRevertAgentRun(t *testing.T) {
	ct := historyFixture(t)
	user := Actor{Kind: models.ActorUser, Name: "alice"}
	agent := Actor{Kind: models.ActorAgent, Name: "rewriter", AgentRunID: 7}

	edited, err := SaveItem(user, ct, models.JSONMap{"title": "Mine"})
	require.NoError(t, err)
	removed, err := SaveItem(user, ct, models.JSONMap{"title": "Keep"})
	require.NoError(t, err)
	touched, err := SaveItem(user, ct, models.JSONMap{"title": "Before"})
	require.NoError(t, err)

	// The run edits one item twice, creates one, deletes one and edits a
	// last one that a person changes afterwards.
	_, err = UpdateItemContent(agent, ct, edited.ID, models.JSONMap{"title": "Agent 1"}, nil)
	require.NoError(t, err)
	_, err = UpdateItemContent(agent, ct, edited.ID, models.JSONMap{"title": "Agent 2"}, nil)
	require.NoError(t, err)
	created, err := SaveItem(agent, ct, models.JSONMap{"title": "Spam"})
	require.NoError(t, err)
	require.NoError(t, DeleteItem(agent, removed.ID))
	_, err = UpdateItemContent(agent, ct, touched.ID, models.JSONMap{"title": "Agent"}, nil)
	require.NoError(t, err)
	_, err = UpdateItemContent(user, ct, touched.ID, models.JSONMap{"title": "Fixed by hand"}, nil)
	require.NoError(t, err)

	byRun, total, err := ListRevisions(RevisionQuery{AgentRunID: 7})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Equal(t, "rewriter", byRun[0].ActorName)

	reverter := Actor{Kind: models.ActorUser, Name: "admin"}
	out, err := RevertAgentRun(reverter, 7, false)
	require.NoError(t, err)
	results := map[uint]RevertOutcome{}
	for _, o := range out {
		results[o.EntityID] = o
	}
	require.Len(t, results, 4)
	assert.Equal(t, "reverted", results[edited.ID].Result)
	assert.EqualValues(t, 1, results[edited.ID].Version, "back to the version before the run")
	assert.Equal(t, "deleted", results[created.ID].Result)
	assert.Equal(t, "reverted", results[removed.ID].Result)
	assert.Equal(t, "skipped", results[touched.ID].Result)
	assert.Contains(t, results[touched.ID].Reason, "user alice")

	item, err := GetItemByID(ct.ID, edited.ID)
	require.NoError(t, err)
	assert.Equal(t, "Mine", item.Data["title"])
	_, err = GetItemByID(ct.ID, created.ID)
	assert.Error(t, err, "the item the run created is gone")
	item, err = GetItemByID(ct.ID, removed.ID)
	require.NoError(t, err, "the item the run deleted is back")
	assert.Equal(t, "Keep", item.Data["title"])
	item, err = GetItemByID(ct.ID, touched.ID)
	require.NoError(t, err)
	assert.Equal(t, "Fixed by hand", item.Data["title"], "later changes are not overwritten")

	latest, err := LatestRevision(models.RevisionEntityItem, edited.ID)
	require.NoError(t, err)
	assert.Equal(t, "admin", latest.ActorName, "the revert is attributed to who asked for it")
	assert.Equal(t, models.RevisionRestore, latest.Action)

	// With force, the edited item is reverted too; the rest is done.
	out, err = RevertAgentRun(reverter, 7, true)
	require.NoError(t, err)
	for _, o := range out {
		if o.EntityID == touched.ID {
			assert.Equal(t, "reverted", o.Result)
		}
	}
	item, err = GetItemByID(ct.ID, touched.ID)
	require.NoError(t, err)
	assert.Equal(t, "Before", item.Data["title"])

	_, err = RevertAgentRun(reverter, 999, false)
	assert.ErrorIs(t, err, ErrRunHasNoChanges)
}

func TestRevisionRetentionAndNoops(t *testing.T) {
	ct := historyFixture(t)
	prev := RevisionLimit
	RevisionLimit = 3
	t.Cleanup(func() { RevisionLimit = prev })

	item, err := SaveItem(SystemActor, ct, models.JSONMap{"title": "v1"})
	require.NoError(t, err)
	_, err = UpdateItemContent(SystemActor, ct, item.ID, models.JSONMap{"title": "v1"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"create"}, versions(t, item.ID), "an unchanged save is not a revision")

	for _, title := range []string{"v2", "v3", "v4", "v5"} {
		_, err = UpdateItemContent(SystemActor, ct, item.ID, models.JSONMap{"title": title}, nil)
		require.NoError(t, err)
	}
	revs, total, err := ListRevisions(RevisionQuery{EntityType: models.RevisionEntityItem, EntityID: item.ID})
	require.NoError(t, err)
	assert.Equal(t, 3, total, "only the last RevisionLimit revisions are kept")
	assert.EqualValues(t, 5, revs[0].Version)
	_, err = GetRevision(models.RevisionEntityItem, item.ID, 2)
	assert.ErrorIs(t, err, ErrRevisionNotFound)

	_, _, err = RestoreItemRevision(SystemActor, ct, item.ID, 1)
	assert.ErrorIs(t, err, ErrRevisionNotFound, "pruned versions cannot be restored")
}

func TestDiffData(t *testing.T) {
	changes := DiffData(
		models.JSONMap{"a": 1.0, "b": "x", "c": []any{1.0}},
		models.JSONMap{"a": 1, "b": "y", "d": true},
	)
	assert.Equal(t, []FieldChange{
		{Field: "b", Change: "changed", Before: "x", After: "y"},
		{Field: "c", Change: "removed", Before: []any{1.0}},
		{Field: "d", Change: "added", After: true},
	}, changes, "numbers compare by JSON value")
}
