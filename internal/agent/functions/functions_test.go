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

func TestUpsertItemInLocale(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, db.AutoMigrate(&models.Collection{}, &models.Attribute{}, &models.Item{}))
	require.NoError(t, db.Create(&models.Collection{Name: "articles", Kind: "collection",
		Attributes: []models.Attribute{{Name: "title", Type: "string", Localized: true}, {Name: "views", Type: "int"}}}).Error)
	require.NoError(t, db.Create(&models.Item{CollectionID: 1, Data: models.JSONMap{"title": "Hello", "views": 1}}).Error)
	require.NoError(t, db.Create(&models.Locale{Code: "en", Name: "English", IsDefault: true}).Error)
	require.NoError(t, db.Create(&models.Locale{Code: "fr", Name: "Français"}).Error)

	ctx := context.Background()
	out, err := listLocales(ctx, map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, out, `"code":"fr"`)

	out, err = upsertCollectionItem(ctx, map[string]any{
		"collection_name": "articles", "item_id": "1", "locale": "fr", "data": map[string]any{"title": "Bonjour"},
	})
	require.NoError(t, err)
	assert.Contains(t, out, `"action":"translated"`)
	var item models.Item
	require.NoError(t, db.First(&item, 1).Error)
	assert.Equal(t, "Hello", item.Data["title"])
	assert.Equal(t, "Bonjour", models.Translation(item.Translations, "fr")["title"])

	out, _ = upsertCollectionItem(ctx, map[string]any{
		"collection_name": "articles", "item_id": "1", "locale": "fr", "data": map[string]any{"views": 2},
	})
	assert.Contains(t, out, "not localized")
	out, _ = upsertCollectionItem(ctx, map[string]any{
		"collection_name": "articles", "locale": "fr", "data": map[string]any{"title": "Nouveau"},
	})
	assert.Contains(t, out, "default locale")
}
