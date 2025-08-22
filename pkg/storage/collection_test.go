package storage_test

import (
	"bytes"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestCollectionStorage(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	err := db.AutoMigrate(&models.Collection{}, &models.Attribute{}, &models.Item{})
	assert.NoError(t, err, "Failed to apply migrations")

	// Seed initial data
	testCollection := &models.Collection{
		Name: "articles",
		Attributes: []models.Attribute{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "text", Required: true},
		},
	}
	err = db.Create(testCollection).Error
	assert.NoError(t, err, "Failed to seed initial 'articles' collection")

	t.Run("GetAllCollections_NoFilters", func(t *testing.T) {
		// Call with no filters, no sort, no range
		collections, total, err := storage.GetAllCollections(nil, nil, nil)
		assert.NoError(t, err, "Expected no error retrieving all collections")
		assert.GreaterOrEqual(t, len(collections), 1, "Expected at least one collection")
		assert.GreaterOrEqual(t, total, 1, "Expected total count to be >= 1")
	})

	t.Run("GetAllCollections_WithFilter", func(t *testing.T) {
		// Filter by name = "articles"
		filters := map[string]any{"name": "articles"}
		collections, total, err := storage.GetAllCollections(filters, nil, nil)
		assert.NoError(t, err, "Expected no error retrieving filtered collections")
		assert.Equal(t, 1, len(collections), "Expected exactly one collection with name = 'articles'")
		assert.Equal(t, 1, total, "Expected total count = 1 for name='articles'")
		assert.Equal(t, "articles", collections[0].Name, "Collection name mismatch")
	})

	t.Run("GetAllCollections_WithSort", func(t *testing.T) {
		// Suppose we created more collections, let's test sort by 'name' DESC
		sortValues := []string{"name", "DESC"}
		collections, total, err := storage.GetAllCollections(nil, sortValues, nil)
		assert.NoError(t, err, "Expected no error retrieving sorted collections")
		assert.GreaterOrEqual(t, len(collections), 1)
		assert.GreaterOrEqual(t, total, 1)
		// If we had multiple collections (articles, products, etc.), we'd check order
	})

	t.Run("GetAllCollections_WithRange", func(t *testing.T) {
		// Range as [0, 0] => first item only
		rangeValues := []int{0, 0}
		collections, total, err := storage.GetAllCollections(nil, nil, rangeValues)
		assert.NoError(t, err, "Expected no error retrieving paginated collections")
		assert.Equal(t, 1, len(collections), "Expected 1 collection in this range")
		assert.GreaterOrEqual(t, total, 1)
	})
}
