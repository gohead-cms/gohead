package functions

import (
	"context"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFixture(t *testing.T) {
	t.Helper()
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.Collection{}, &models.Attribute{}, &models.Item{}, &models.Revision{}, &models.Proposal{}, &models.Locale{}))
	require.NoError(t, db.Create(&models.Collection{Name: "articles", Kind: "collection",
		Attributes: []models.Attribute{{Name: "title", Type: "string", Required: true}, {Name: "summary", Type: "text"}}}).Error)
	require.NoError(t, db.Create(&models.Item{CollectionID: 1, Data: models.JSONMap{"title": "Hello", "summary": "old"}}).Error)
}

func itemData(t *testing.T) models.JSONMap {
	t.Helper()
	item, err := storage.GetItemByID(1, 1)
	require.NoError(t, err)
	return item.Data
}

func TestPatchItemKeepsOtherFields(t *testing.T) {
	writeFixture(t)
	out, err := patchCollectionItem(context.Background(), `{"collection_name":"articles","item_id":1,"data":{"summary":"new"}}`)
	require.NoError(t, err)
	assert.Contains(t, out, `"updated"`, out)
	assert.Equal(t, "Hello", itemData(t)["title"])
	assert.Equal(t, "new", itemData(t)["summary"])

	out, _ = patchCollectionItem(context.Background(), `{"collection_name":"articles","item_id":99,"data":{"summary":"x"}}`)
	assert.Contains(t, out, "not found")
	out, _ = patchCollectionItem(context.Background(), `{"collection_name":"articles","data":{"summary":"x"}}`)
	assert.Contains(t, out, "missing required parameters")
}

func TestDryRunWritesNothing(t *testing.T) {
	writeFixture(t)
	ctx := WithWriteMode(context.Background(), WriteDryRun)

	out, err := patchCollectionItem(ctx, map[string]any{"collection_name": "articles", "item_id": "1", "data": map[string]any{"summary": "new"}})
	require.NoError(t, err)
	assert.Contains(t, out, `"dry_run":true`, out)
	assert.Contains(t, out, `would_update`)
	assert.Equal(t, "old", itemData(t)["summary"])

	// Validation still runs, so the run shows what would fail.
	out, _ = upsertCollectionItem(ctx, map[string]any{"collection_name": "articles", "data": map[string]any{"summary": "no title"}})
	assert.Contains(t, out, `"error"`, out)

	out, _ = upsertCollectionItem(ctx, map[string]any{"collection_name": "articles", "data": map[string]any{"title": "New"}})
	assert.Contains(t, out, `would_create`, out)
	out, _ = deleteCollectionItem(ctx, map[string]any{"collection_name": "articles", "item_id": "1"})
	assert.Contains(t, out, `would_delete`, out)

	var items, revisions, proposals int64
	database.DB.Model(&models.Item{}).Count(&items)
	database.DB.Model(&models.Revision{}).Count(&revisions)
	database.DB.Model(&models.Proposal{}).Count(&proposals)
	assert.EqualValues(t, 1, items)
	assert.Zero(t, revisions)
	assert.Zero(t, proposals)
}

func TestProposeModeQueuesChanges(t *testing.T) {
	writeFixture(t)
	ctx := WithWriteMode(context.Background(), WritePropose)
	ctx = storage.WithActor(ctx, storage.Actor{Kind: models.ActorAgent, Name: "summarizer", AgentRunID: 3})

	out, err := patchCollectionItem(ctx, map[string]any{"collection_name": "articles", "item_id": 1.0,
		"data": map[string]any{"summary": "new"}, "reason": "shorter"})
	require.NoError(t, err)
	assert.Contains(t, out, `"pending_approval"`, out)
	assert.Equal(t, "old", itemData(t)["summary"], "nothing written before approval")

	list, total, err := storage.ListProposals(storage.ProposalQuery{Status: models.ProposalPending})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	p := list[0]
	assert.Equal(t, "summarizer", p.AgentName)
	require.NotNil(t, p.AgentRunID)
	assert.EqualValues(t, 3, *p.AgentRunID)
	assert.Equal(t, models.ProposalUpdate, p.Action)
	assert.Equal(t, "shorter", p.Reason)
	assert.Equal(t, "Hello", p.Data["title"], "the proposal holds the whole item")

	// Invalid changes are refused right away, not proposed.
	out, _ = upsertCollectionItem(ctx, map[string]any{"collection_name": "articles", "data": map[string]any{"summary": "no title"}})
	assert.Contains(t, out, `"error"`, out)
	_, total, _ = storage.ListProposals(storage.ProposalQuery{Status: "all"})
	assert.Equal(t, 1, total)
}
