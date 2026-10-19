package graphql

import (
	"context"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveRelation runs the resolver against a real (in-memory) database
// rather than mocks: storage is accessed through package-level functions, so
// a mock would never be called.
func TestResolveRelation(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, db.AutoMigrate(&models.Collection{}, &models.Attribute{}, &models.Item{}))

	authors := models.Collection{
		Name:       "authors",
		Attributes: []models.Attribute{{Name: "name", Type: "text", Required: true}},
	}
	require.NoError(t, db.Create(&authors).Error)

	john := models.Item{CollectionID: authors.ID, Data: models.JSONMap{"name": "John Doe"}}
	jane := models.Item{CollectionID: authors.ID, Data: models.JSONMap{"name": "Jane Smith"}}
	require.NoError(t, db.Create(&john).Error)
	require.NoError(t, db.Create(&jane).Error)

	oneToOne := models.Attribute{Name: "author", Type: "relation", Target: "authors", Relation: "oneToOne"}
	manyToMany := models.Attribute{Name: "authors", Type: "relation", Target: "authors", Relation: "manyToMany"}

	t.Run("one-to-one", func(t *testing.T) {
		p := graphql.ResolveParams{Context: auth.WithRole(context.Background(), "viewer"), Source: map[string]any{"author": float64(john.ID)}}
		result, err := ResolveRelation(p, 2, oneToOne)
		require.NoError(t, err)
		related, ok := result.(map[string]any)
		require.True(t, ok, "expected a JSON map, got %T", result)
		assert.Equal(t, "John Doe", related["name"])
		assert.EqualValues(t, john.ID, related["id"])
	})

	t.Run("many-to-many skips missing ids", func(t *testing.T) {
		p := graphql.ResolveParams{Context: auth.WithRole(context.Background(), "viewer"), Source: map[string]any{"authors": []any{float64(john.ID), float64(jane.ID), float64(9999)}}}
		result, err := ResolveRelation(p, 2, manyToMany)
		require.NoError(t, err)
		related, ok := result.([]map[string]any)
		require.True(t, ok, "expected a list, got %T", result)
		require.Len(t, related, 2)
		assert.Equal(t, "John Doe", related[0]["name"])
		assert.Equal(t, "Jane Smith", related[1]["name"])
	})

	t.Run("missing related item resolves to null", func(t *testing.T) {
		p := graphql.ResolveParams{Context: auth.WithRole(context.Background(), "viewer"), Source: map[string]any{"author": float64(9999)}}
		result, err := ResolveRelation(p, 2, oneToOne)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("unset relation resolves to null", func(t *testing.T) {
		p := graphql.ResolveParams{Context: auth.WithRole(context.Background(), "viewer"), Source: map[string]any{}}
		result, err := ResolveRelation(p, 2, oneToOne)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("non-map source is an error", func(t *testing.T) {
		p := graphql.ResolveParams{Context: auth.WithRole(context.Background(), "viewer"), Source: models.Item{}}
		_, err := ResolveRelation(p, 2, oneToOne)
		assert.Error(t, err)
	})
}
