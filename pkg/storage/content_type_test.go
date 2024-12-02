package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
)

func TestContentTypeStorage(t *testing.T) {
	// Set up the test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Seed initial data
	testContentType := &models.ContentType{
		Name: "articles",
		Fields: []models.Field{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "text", Required: true},
		},
		Relationships: []models.Relationship{
			{FieldName: "author", RelatedType: "users", RelationType: "one-to-one"},
		},
	}

	assert.NoError(t, db.Create(testContentType).Error, "Failed to seed initial content type")

	t.Run("SaveContentType", func(t *testing.T) {
		newContentType := &models.ContentType{
			Name: "products",
			Fields: []models.Field{
				{Name: "name", Type: "string", Required: true},
				{Name: "price", Type: "int", Required: true},
			},
		}

		err := storage.SaveContentType(newContentType)
		assert.NoError(t, err, "Failed to save content type")

		var fetched models.ContentType
		err = db.Where("name = ?", "products").First(&fetched).Error
		assert.NoError(t, err, "Failed to fetch saved content type")
		assert.Equal(t, "products", fetched.Name, "Content type name mismatch")
	})

	t.Run("GetContentType", func(t *testing.T) {
		ct, err := storage.GetContentType("articles")
		assert.NoError(t, err, "Expected no error when retrieving content type")
		assert.Equal(t, "articles", ct.Name, "Content type name mismatch")
		assert.Equal(t, 2, len(ct.Fields), "Expected 2 fields")
		assert.Equal(t, 1, len(ct.Relationships), "Expected 1 relationship")
	})

	t.Run("GetContentTypeByName", func(t *testing.T) {
		ct, err := storage.GetContentTypeByName("articles")
		assert.NoError(t, err, "Expected no error when retrieving content type by name")
		assert.Equal(t, "articles", ct.Name, "Content type name mismatch")
	})

	t.Run("GetAllContentTypes", func(t *testing.T) {
		cts, err := storage.GetAllContentTypes()
		assert.NoError(t, err, "Expected no error when retrieving all content types")
		assert.GreaterOrEqual(t, len(cts), 1, "Expected at least one content type")
	})

	t.Run("UpdateContentType", func(t *testing.T) {
		updatedContentType := &models.ContentType{
			Name: "articles",
			Fields: []models.Field{
				{Name: "title", Type: "string", Required: true},
				{Name: "summary", Type: "string", Required: false},
			},
		}

		err := storage.UpdateContentType("articles", updatedContentType)
		assert.NoError(t, err, "Failed to update content type")

		ct, err := storage.GetContentType("articles")
		assert.NoError(t, err, "Expected no error when retrieving updated content type")
		assert.Equal(t, 2, len(ct.Fields), "Expected 2 fields after update")
		assert.Equal(t, "summary", ct.Fields[1].Name, "Expected updated field name")
	})

	t.Run("DeleteContentType", func(t *testing.T) {
		err := storage.DeleteContentType(testContentType.ID)
		assert.NoError(t, err, "Failed to delete content type")

		_, err = storage.GetContentType("articles")
		assert.Error(t, err, "Expected error when fetching deleted content type")
		assert.Contains(t, err.Error(), "content type not found", "Error message mismatch")
	})
}
