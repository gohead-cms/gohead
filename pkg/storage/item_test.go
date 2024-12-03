package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
)

func TestSaveItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}))

	// Create a sample content item
	item := &models.Item{
		CollectionID: 1,
		Data: models.JSONMap{
			"title":   "Sample Article",
			"content": "This is a test article.",
		},
	}

	// Save the content item
	err := SaveItem(item)
	assert.NoError(t, err)
	assert.NotZero(t, item.ID)

	// Verify the content item exists in the database
	var result models.Item
	err = db.First(&result, item.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Sample Article", result.Data["title"])
}

func TestGetItemByID(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}))

	// Create and save a sample content item
	item := &models.Item{
		CollectionID: 1,
		Data: models.JSONMap{
			"title":   "Test Article",
			"content": "This is a test.",
		},
	}
	assert.NoError(t, db.Create(item).Error)

	// Fetch the content item by ID
	result, err := GetItemByID(item.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Article", result.Data["title"])
}

func TestGetItems(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}))

	// Create and save multiple content items
	items := []models.Item{
		{CollectionID: 1, Data: models.JSONMap{"title": "Article 1"}},
		{CollectionID: 1, Data: models.JSONMap{"title": "Article 2"}},
	}
	assert.NoError(t, db.Create(&items).Error)

	// Fetch all content items
	results, err := GetItems("articles")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Article 1", results[0].Data["title"])
	assert.Equal(t, "Article 2", results[1].Data["title"])
}

func TestUpdateItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}, &models.Relationship{}))

	// Create and save a content item
	item := &models.Item{
		CollectionID: 1,
		Data:         models.JSONMap{"title": "Old Title", "content": "Old Content"},
	}
	assert.NoError(t, db.Create(item).Error)

	// Update the content item
	ct := models.Collection{Name: "articles"}
	updatedData := models.JSONMap{"title": "New Title", "content": "New Content"}
	err := UpdateItem(ct, item.ID, updatedData)
	assert.NoError(t, err)

	// Verify the content item is updated
	var updatedItem models.Item
	assert.NoError(t, db.First(&updatedItem, item.ID).Error)
	assert.Equal(t, "New Title", updatedItem.Data["title"])
}

func TestDeleteItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}, &models.Relationship{}))

	// Create and save a content item
	item := &models.Item{
		CollectionID: 1,
		Data:         models.JSONMap{"title": "Test Title", "content": "Test Content"},
	}
	assert.NoError(t, db.Create(item).Error)

	// Delete the content item
	ct := models.Collection{Name: "articles"}
	err := DeleteItem(ct, item.ID)
	assert.NoError(t, err)

	// Verify the content item is deleted
	var deletedItem models.Item
	err = db.First(&deletedItem, item.ID).Error
	assert.Error(t, err) // Should return an error because the item no longer exists
}
