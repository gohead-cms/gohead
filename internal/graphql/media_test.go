package graphql

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/media"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphQLMedia(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, migrations.MigrateDatabase(db))
	t.Cleanup(ResetTypeRegistry)

	store, err := media.NewLocal(t.TempDir(), "https://cms.example.com/uploads")
	require.NoError(t, err)
	storage.InitMediaStore(store)
	t.Cleanup(func() { storage.InitMediaStore(nil) })

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 16, 9))))
	img, err := storage.SaveMedia(context.Background(), storage.UploadFile{
		Name: "hero.png", Size: int64(buf.Len()), Content: bytes.NewReader(buf.Bytes()), Alt: "Hero",
	}, storage.UploadPolicy{})
	require.NoError(t, err)

	require.NoError(t, storage.CreateComponent(&models.Component{
		Name: "gallery",
		Attributes: []models.ComponentAttribute{
			{BaseAttribute: models.BaseAttribute{Name: "images", Type: "media", Multiple: true}},
		},
	}))
	pages := models.Collection{Name: "pages", Kind: "collection", Attributes: []models.Attribute{
		{Name: "title", Type: "string"},
		{Name: "hero", Type: "media", AllowedTypes: []string{"image/*"}},
		{Name: "legacy", Type: "media"},
		{Name: "gallery", Type: "component", ComponentRef: "gallery"},
	}}
	require.NoError(t, storage.SaveCollection(&pages))
	require.NoError(t, InitializeGraphQLSchema())

	created := gqlData(t, "editor", fmt.Sprintf(`mutation {
		createpages(input: {title: "Home", hero: "%d", legacy: "https://old.example/x.jpg", gallery: {images: ["%d"]}}) {
			hero { id url width height alt mime } legacy { url } gallery { images { url } }
		}
	}`, img.ID, img.ID))["createpages"].(map[string]any)

	hero := created["hero"].(map[string]any)
	assert.Equal(t, fmt.Sprint(img.ID), fmt.Sprint(hero["id"]))
	assert.Regexp(t, `^https://cms\.example\.com/uploads/\d{4}/\d{2}/[0-9a-f]+-hero\.png$`, hero["url"])
	assert.Equal(t, 16, hero["width"])
	assert.Equal(t, "Hero", hero["alt"])
	assert.Equal(t, map[string]any{"url": "https://old.example/x.jpg"}, created["legacy"])
	assert.Len(t, created["gallery"].(map[string]any)["images"], 1)

	read := gqlData(t, "viewer", `{ pages { hero { url } } }`)["pages"].([]any)
	assert.Equal(t, hero["url"], read[0].(map[string]any)["hero"].(map[string]any)["url"])

	res := runGQL(t, "editor", `mutation { createpages(input: {hero: "999999"}) { title } }`)
	require.NotEmpty(t, res.Errors, "unknown files are rejected")
}
