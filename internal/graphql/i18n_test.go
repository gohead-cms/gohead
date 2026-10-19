package graphql

import (
	"fmt"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphQLLocales(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, migrations.MigrateDatabase(db))
	t.Cleanup(ResetTypeRegistry)

	_, err := storage.CreateLocale("fr", "Français")
	require.NoError(t, err)
	authors := models.Collection{Name: "authors", Kind: "collection", Attributes: []models.Attribute{
		{Name: "name", Type: "string"},
		{Name: "bio", Type: "text", Localized: true},
	}}
	require.NoError(t, storage.SaveCollection(&authors))
	pages := models.Collection{Name: "pages", Kind: "collection", Attributes: []models.Attribute{
		{Name: "title", Type: "string", Localized: true},
		{Name: "order", Type: "int"},
		{Name: "author", Type: "relation", Relation: "oneToOne", Target: "authors"},
	}}
	require.NoError(t, storage.SaveCollection(&pages))
	require.NoError(t, InitializeGraphQLSchema())

	author := gqlData(t, "editor", `mutation { createauthors(input: {name: "Ada", bio: "Mathematician"}) { id } }`)["createauthors"].(map[string]any)
	gqlData(t, "editor", fmt.Sprintf(`mutation { updateauthors(id: "%v", locale: "fr", input: {bio: "Mathématicienne"}) { id } }`, author["id"]))
	page := gqlData(t, "editor", fmt.Sprintf(`mutation { createpages(input: {title: "Home", order: 1, author: "%v"}) { id locale locales } }`, author["id"]))["createpages"].(map[string]any)
	assert.Equal(t, "en", page["locale"])

	updated := gqlData(t, "editor", fmt.Sprintf(`mutation { updatepages(id: "%v", locale: "fr", input: {title: "Accueil"}) { title order locale locales } }`, page["id"]))["updatepages"].(map[string]any)
	assert.Equal(t, "Accueil", updated["title"])
	assert.Equal(t, 1, updated["order"])
	assert.Equal(t, []any{"en", "fr"}, updated["locales"])

	res := runGQL(t, "editor", fmt.Sprintf(`mutation { updatepages(id: "%v", locale: "fr", input: {order: 2}) { id } }`, page["id"]))
	require.NotEmpty(t, res.Errors, "shared attributes are not written in another locale")

	fr := gqlData(t, "viewer", `{ pages(locale: "fr") { title order locale author { name bio } } }`)["pages"].([]any)
	require.Len(t, fr, 1)
	got := fr[0].(map[string]any)
	assert.Equal(t, "Accueil", got["title"])
	assert.Equal(t, "fr", got["locale"])
	assert.Equal(t, "Mathématicienne", got["author"].(map[string]any)["bio"], "relations are read in the same locale")

	en := gqlData(t, "viewer", `{ pages { title } }`)["pages"].([]any)
	assert.Equal(t, "Home", en[0].(map[string]any)["title"])

	res = runGQL(t, "viewer", `{ pages(locale: "xx") { title } }`)
	require.NotEmpty(t, res.Errors)
}
