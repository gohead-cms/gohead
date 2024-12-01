package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
)

func TestSaveContentItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.ContentItem{}))

	// Create a sample content item
	item := &models.ContentItem{
		ContentType: "articles",
		Data: models.JSONMap{
			"title":   "Sample Article",
			"content": "This is a test article.",
		},
	}

	// Save the content item
	err := SaveContentItem(item)
	assert.NoError(t, err)
	assert.NotZero(t, item.ID)

	// Verify the content item exists in the database
	var result models.ContentItem
	err = db.First(&result, item.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Sample Article", result.Data["title"])
}

func TestGetContentItemByID(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.ContentItem{}))

	// Create and save a sample content item
	item := &models.ContentItem{
		ContentType: "articles",
		Data: models.JSONMap{
			"title":   "Test Article",
			"content": "This is a test.",
		},
	}
	assert.NoError(t, db.Create(item).Error)

	// Fetch the content item by ID
	result, err := GetContentItemByID("articles", item.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Article", result.Data["title"])
}

func TestGetContentItems(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.ContentItem{}))

	// Create and save multiple content items
	items := []models.ContentItem{
		{ContentType: "articles", Data: models.JSONMap{"title": "Article 1"}},
		{ContentType: "articles", Data: models.JSONMap{"title": "Article 2"}},
	}
	assert.NoError(t, db.Create(&items).Error)

	// Fetch all content items
	results, err := GetContentItems("articles")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Article 1", results[0].Data["title"])
	assert.Equal(t, "Article 2", results[1].Data["title"])
}

func TestUpdateContentItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.ContentItem{}, &models.ContentRelation{}))

	// Create and save a content item
	item := &models.ContentItem{
		ContentType: "articles",
		Data:        models.JSONMap{"title": "Old Title", "content": "Old Content"},
	}
	assert.NoError(t, db.Create(item).Error)

	// Update the content item
	ct := models.ContentType{Name: "articles"}
	updatedData := models.JSONMap{"title": "New Title", "content": "New Content"}
	err := UpdateContentItem(ct, item.ID, updatedData)
	assert.NoError(t, err)

	// Verify the content item is updated
	var updatedItem models.ContentItem
	assert.NoError(t, db.First(&updatedItem, item.ID).Error)
	assert.Equal(t, "New Title", updatedItem.Data["title"])
}

func TestDeleteContentItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.ContentItem{}, &models.ContentRelation{}))

	// Create and save a content item
	item := &models.ContentItem{
		ContentType: "articles",
		Data:        models.JSONMap{"title": "Test Title", "content": "Test Content"},
	}
	assert.NoError(t, db.Create(item).Error)

	// Delete the content item
	ct := models.ContentType{Name: "articles"}
	err := DeleteContentItem(ct, item.ID)
	assert.NoError(t, err)

	// Verify the content item is deleted
	var deletedItem models.ContentItem
	err = db.First(&deletedItem, item.ID).Error
	assert.Error(t, err) // Should return an error because the item no longer exists
}
