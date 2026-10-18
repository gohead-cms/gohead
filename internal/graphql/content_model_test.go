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

func gqlData(t *testing.T, role, query string) map[string]any {
	t.Helper()
	res := runGQL(t, role, query)
	require.Empty(t, res.Errors, "query: %s", query)
	return res.Data.(map[string]any)
}

func TestGraphQLComponentsAndDrafts(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, migrations.MigrateDatabase(db))
	// Types are cached package-wide; don't leak these into other tests.
	t.Cleanup(ResetTypeRegistry)

	require.NoError(t, storage.CreateComponent(&models.Component{
		Name: "seo",
		Attributes: []models.ComponentAttribute{
			{BaseAttribute: models.BaseAttribute{Name: "metaTitle", Type: "string", Required: true}},
			{BaseAttribute: models.BaseAttribute{Name: "priority", Type: "int"}},
		},
	}))
	authors := models.Collection{Name: "authors", Kind: "collection", DraftAndPublish: true,
		Attributes: []models.Attribute{{Name: "name", Type: "string"}}}
	require.NoError(t, storage.SaveCollection(&authors))
	posts := models.Collection{Name: "posts", Kind: "collection", DraftAndPublish: true,
		Attributes: []models.Attribute{
			{Name: "title", Type: "string"},
			{Name: "seo", Type: "component", ComponentRef: "seo"},
			{Name: "faq", Type: "component", ComponentRef: "seo", Repeatable: true},
			{Name: "author", Type: "relation", Relation: "oneToOne", Target: "authors"},
		}}
	require.NoError(t, storage.SaveCollection(&posts))
	require.NoError(t, InitializeGraphQLSchema())

	// A draft author, and a published post pointing at it.
	created := gqlData(t, "editor", `mutation { createauthors(input: {name: "Ada"}) { id status } }`)["createauthors"].(map[string]any)
	assert.Equal(t, "draft", created["status"])
	authorID := created["id"]

	post := gqlData(t, "editor", fmt.Sprintf(`mutation {
		createposts(status: "published", input: {
			title: "Hello", author: "%v",
			seo: {metaTitle: "Hi", priority: 2},
			faq: [{metaTitle: "Q1"}, {metaTitle: "Q2"}]
		}) { id status publishedAt seo { metaTitle priority } faq { metaTitle } }
	}`, authorID))["createposts"].(map[string]any)
	assert.Equal(t, "published", post["status"])
	assert.NotNil(t, post["publishedAt"])
	assert.Equal(t, map[string]any{"metaTitle": "Hi", "priority": 2}, post["seo"])
	assert.Len(t, post["faq"], 2)

	// Invalid component values are rejected by the same validation as REST.
	res := runGQL(t, "editor", `mutation { createposts(input: {title: "x", seo: {priority: 1}}) { id } }`)
	require.NotEmpty(t, res.Errors)

	query := `{ posts { title status seo { metaTitle } faq { metaTitle } author { name } } }`
	viewer := gqlData(t, "viewer", query)["posts"].([]any)
	require.Len(t, viewer, 1)
	p := viewer[0].(map[string]any)
	assert.Equal(t, "Hi", p["seo"].(map[string]any)["metaTitle"])
	assert.Nil(t, p["author"], "the draft author is hidden from viewers")

	// Viewers never see drafts; editors opt in.
	assert.Empty(t, gqlData(t, "viewer", `{ authors { id } }`)["authors"])
	res = runGQL(t, "viewer", `{ authors(status: "draft") { id } }`)
	require.NotEmpty(t, res.Errors)
	assert.Len(t, gqlData(t, "editor", `{ authors(status: "all") { id } }`)["authors"], 1)
	editorPosts := gqlData(t, "editor", `{ posts(status: "all") { author { name status } } }`)["posts"].([]any)
	assert.Equal(t, "Ada", editorPosts[0].(map[string]any)["author"].(map[string]any)["name"])

	// Publishing makes the author visible, also through the relation.
	pub := gqlData(t, "editor", fmt.Sprintf(`mutation { publishauthors(id: "%v") { status } }`, authorID))
	assert.Equal(t, "published", pub["publishauthors"].(map[string]any)["status"])
	p = gqlData(t, "viewer", query)["posts"].([]any)[0].(map[string]any)
	assert.Equal(t, "Ada", p["author"].(map[string]any)["name"])

	res = runGQL(t, "viewer", fmt.Sprintf(`mutation { unpublishauthors(id: "%v") { status } }`, authorID))
	require.NotEmpty(t, res.Errors, "viewers cannot unpublish")
}
