package storage

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupQueryFixtures(t *testing.T) models.Collection {
	t.Helper()
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.Item{}, &models.Collection{}, &models.Attribute{}))

	ct := models.Collection{
		Name: "articles",
		Attributes: []models.Attribute{
			{Name: "title", Type: "string"},
			{Name: "views", Type: "int"},
			{Name: "rating", Type: "float"},
			{Name: "published", Type: "bool"},
			{Name: "author", Type: "relation", Target: "users", Relation: "oneToOne"},
		},
	}
	require.NoError(t, db.Create(&ct).Error)

	other := models.Collection{Name: "pages"}
	require.NoError(t, db.Create(&other).Error)

	rows := []models.JSONMap{
		{"title": "Go generics", "views": 120, "rating": 4.5, "published": true},
		{"title": "Rust ownership", "views": 15, "rating": 3.0, "published": false},
		{"title": "Go 100% coverage", "views": 900, "rating": 4.9, "published": true},
		{"title": "Kubernetes", "views": 9, "rating": 2.5, "published": true},
	}
	for _, r := range rows {
		require.NoError(t, db.Create(&models.Item{CollectionID: ct.ID, Data: r}).Error)
	}
	// An item in another collection must never leak into results.
	require.NoError(t, db.Create(&models.Item{CollectionID: other.ID, Data: models.JSONMap{"title": "Go elsewhere", "views": 5000}}).Error)
	return ct
}

func titles(items []models.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i], _ = it.Data["title"].(string)
	}
	return out
}

func TestQueryItems(t *testing.T) {
	ct := setupQueryFixtures(t)

	t.Run("default order and pagination", func(t *testing.T) {
		items, total, err := QueryItems(ct, ItemQuery{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Equal(t, []string{"Go generics", "Rust ownership"}, titles(items))

		items, _, err = QueryItems(ct, ItemQuery{Page: 2, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, []string{"Go 100% coverage", "Kubernetes"}, titles(items))
	})

	t.Run("text equality", func(t *testing.T) {
		items, total, err := QueryItems(ct, ItemQuery{Filters: []ItemFilter{{Field: "title", Op: OpEq, Value: "Kubernetes"}}})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, []string{"Kubernetes"}, titles(items))
	})

	t.Run("contains escapes LIKE wildcards", func(t *testing.T) {
		items, _, err := QueryItems(ct, ItemQuery{Filters: []ItemFilter{{Field: "title", Op: OpContains, Value: "Go"}}})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"Go generics", "Go 100% coverage"}, titles(items))

		items, _, err = QueryItems(ct, ItemQuery{Filters: []ItemFilter{{Field: "title", Op: OpContains, Value: "100%"}}})
		require.NoError(t, err)
		assert.Equal(t, []string{"Go 100% coverage"}, titles(items))

		items, _, err = QueryItems(ct, ItemQuery{Filters: []ItemFilter{{Field: "title", Op: OpContains, Value: "%"}}})
		require.NoError(t, err)
		assert.Equal(t, []string{"Go 100% coverage"}, titles(items), "a bare % must match literally, not everything")
	})

	t.Run("numeric comparison is numeric, not lexical", func(t *testing.T) {
		items, total, err := QueryItems(ct, ItemQuery{
			Filters: []ItemFilter{{Field: "views", Op: OpGte, Value: "100"}},
			Sort:    []ItemSort{{Field: "views"}},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Equal(t, []string{"Go generics", "Go 100% coverage"}, titles(items))

		items, _, err = QueryItems(ct, ItemQuery{Filters: []ItemFilter{{Field: "rating", Op: OpLt, Value: "3"}}})
		require.NoError(t, err)
		assert.Equal(t, []string{"Kubernetes"}, titles(items))
	})

	t.Run("boolean equality", func(t *testing.T) {
		items, total, err := QueryItems(ct, ItemQuery{Filters: []ItemFilter{{Field: "published", Op: OpEq, Value: "false"}}})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, []string{"Rust ownership"}, titles(items))
	})

	t.Run("combined filters and multi-key sort", func(t *testing.T) {
		items, _, err := QueryItems(ct, ItemQuery{
			Filters: []ItemFilter{
				{Field: "published", Op: OpEq, Value: "true"},
				{Field: "views", Op: OpGt, Value: "10"},
			},
			Sort: []ItemSort{{Field: "rating", Desc: true}},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Go 100% coverage", "Go generics"}, titles(items))
	})

	t.Run("sort by built-in column", func(t *testing.T) {
		items, _, err := QueryItems(ct, ItemQuery{Sort: []ItemSort{{Field: "id", Desc: true}}})
		require.NoError(t, err)
		assert.Equal(t, "Kubernetes", titles(items)[0])
	})

	t.Run("rejects unknown and unsupported fields", func(t *testing.T) {
		cases := []ItemQuery{
			{Filters: []ItemFilter{{Field: "nope", Op: OpEq, Value: "x"}}},
			{Filters: []ItemFilter{{Field: "author", Op: OpEq, Value: "1"}}},
			{Filters: []ItemFilter{{Field: "title", Op: "$regex", Value: "x"}}},
			{Filters: []ItemFilter{{Field: "views", Op: OpEq, Value: "many"}}},
			{Filters: []ItemFilter{{Field: "views", Op: OpContains, Value: "1"}}},
			{Filters: []ItemFilter{{Field: "published", Op: OpGt, Value: "true"}}},
			{Sort: []ItemSort{{Field: "id; DROP TABLE items"}}},
			{Filters: []ItemFilter{{Field: "title') OR 1=1 --", Op: OpEq, Value: "x"}}},
		}
		for _, q := range cases {
			_, _, err := QueryItems(ct, q)
			var qe *QueryFieldError
			assert.ErrorAs(t, err, &qe, "%+v", q)
		}
	})
}

func TestGetItemsByOffset(t *testing.T) {
	ct := setupQueryFixtures(t)

	items, total, err := GetItemsByOffset(ct.ID, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Equal(t, []string{"Go generics", "Rust ownership"}, titles(items))

	items, _, err = GetItemsByOffset(ct.ID, 3, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"Kubernetes"}, titles(items))
}
