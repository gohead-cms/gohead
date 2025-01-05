package storage_test

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
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
	err := db.AutoMigrate(&models.Collection{},
		&models.Attribute{},
		&models.Item{},
	)
	assert.NoError(t, err, "Failed to apply migrations")

	// Seed initial data
	testCollection := &models.Collection{
		Name: "articles",
		Attributes: []models.Attribute{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "text", Required: true},
		},
		// Relationships: []models.Relationship{
		//	{Field: "author", CollectionID: 1, RelationType: "one-to-one"},
		// },
	}

	assert.NoError(t, db.Create(testCollection).Error, "Failed to seed initial content type")

	t.Run("SaveCollection", func(t *testing.T) {
		newCollection := &models.Collection{
			Name: "products",
			Attributes: []models.Attribute{
				{Name: "name", Type: "string", Required: true},
				{Name: "price", Type: "int", Required: true},
			},
		}

		err := storage.SaveCollection(newCollection)
		assert.NoError(t, err, "Failed to save content type")

		var fetched models.Collection
		err = db.Where("name = ?", "products").First(&fetched).Error
		assert.NoError(t, err, "Failed to fetch saved content type")
		assert.Equal(t, "products", fetched.Name, "Content type name mismatch")
	})

	t.Run("GetCollection", func(t *testing.T) {
		ct, err := storage.GetCollectionByName("articles")
		assert.NoError(t, err, "Expected no error when retrieving content type")
		assert.Equal(t, "articles", ct.Name, "Content type name mismatch")
		assert.Equal(t, 2, len(ct.Attributes), "Expected 2 fields")
		// assert.Equal(t, 1, len(ct.Relationships), "Expected 1 relationship")
	})

	t.Run("GetCollectionByName", func(t *testing.T) {
		ct, err := storage.GetCollectionByName("articles")
		assert.NoError(t, err, "Expected no error when retrieving content type by name")
		assert.Equal(t, "articles", ct.Name, "Content type name mismatch")
	})

	t.Run("GetAllCollections", func(t *testing.T) {
		cts, err := storage.GetAllCollections()
		assert.NoError(t, err, "Expected no error when retrieving all content types")
		assert.GreaterOrEqual(t, len(cts), 1, "Expected at least one content type")
	})

	t.Run("UpdateCollection", func(t *testing.T) {
		updatedCollection := &models.Collection{
			Name: "articles",
			Attributes: []models.Attribute{
				{Name: "title", Type: "string", Required: true},
				{Name: "summary", Type: "string", Required: false},
			},
		}

		err := storage.UpdateCollection("articles", updatedCollection)
		assert.NoError(t, err, "Failed to update content type")

		ct, err := storage.GetCollectionByName("articles")
		assert.NoError(t, err, "Expected no error when retrieving updated content type")
		assert.Equal(t, 2, len(ct.Attributes), "Expected 2 fields after update")
		assert.Equal(t, "summary", ct.Attributes[1].Name, "Expected updated field name")
	})

	t.Run("DeleteCollection", func(t *testing.T) {
		err := storage.DeleteCollection(testCollection.ID)
		assert.NoError(t, err, "Failed to delete content type")

		_, err = storage.GetCollectionByName("articles")
		assert.Error(t, err, "Expected error when fetching deleted content type")
		assert.Contains(t, err.Error(), "content type 'articles' not found", "Error message mismatch")
	})
}
