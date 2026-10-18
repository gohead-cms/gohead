package functions

import (
	"context"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The runner passes tool arguments as the raw JSON string returned by the
// LLM; the registry tool passes a decoded map. Both must work.
func TestCollectionFunctionsAcceptStringAndMapArgs(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, db.AutoMigrate(&models.Collection{}, &models.Attribute{}, &models.Item{}))
	require.NoError(t, db.Create(&models.Collection{Name: "articles", Kind: "collection",
		Attributes: []models.Attribute{{Name: "title", Type: "string"}}}).Error)
	require.NoError(t, db.Create(&models.Item{CollectionID: 1, Data: models.JSONMap{"title": "Hello"}}).Error)

	ctx := context.Background()
	for _, args := range []any{
		`{"collection_name":"articles"}`,
		map[string]any{"collection_name": "articles"},
	} {
		out, err := getCollectionSchema(ctx, args)
		require.NoError(t, err)
		assert.NotContains(t, out, `"status": "error"`, "%T: %s", args, out)
		assert.Contains(t, out, "title")

		out, err = listCollectionItems(ctx, args)
		require.NoError(t, err)
		assert.NotContains(t, out, `"status": "error"`, "%T: %s", args, out)
		assert.Contains(t, out, "Hello")
	}

	out, err := logMessage(ctx, map[string]any{"message": "hi"})
	require.NoError(t, err)
	assert.NotContains(t, out, `"status": "error"`, out)
}
