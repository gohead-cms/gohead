// internal/models/content_item_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestContentItemCRUD(t *testing.T) {
	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Create a ContentItem
	contentItem := ContentItem{
		ContentType: "articles",
		Data: JSONMap{
			"title":   "Test Article",
			"content": "This is the content of the test article.",
		},
	}

	// Auto-migrate the ContentItem model
	err = db.AutoMigrate(&ContentItem{})
	assert.NoError(t, err)

	err = db.Create(&contentItem).Error
	assert.NoError(t, err)
	assert.NotZero(t, contentItem.ID, "ContentItem ID should be set after creation")

	// Retrieve the ContentItem
	var retrievedItem ContentItem
	err = db.First(&retrievedItem, contentItem.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "articles", retrievedItem.ContentType)
	assert.Equal(t, "Test Article", retrievedItem.Data["title"])
	assert.Equal(t, "This is the content of the test article.", retrievedItem.Data["content"])

	// Update the ContentItem
	err = db.Model(&retrievedItem).Update("Data", JSONMap{
		"title":   "Updated Title",
		"content": "Updated content.",
	}).Error
	assert.NoError(t, err)

	// Verify the update
	var updatedItem ContentItem
	err = db.First(&updatedItem, contentItem.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedItem.Data["title"])
	assert.Equal(t, "Updated content.", updatedItem.Data["content"])

	// Delete the ContentItem
	err = db.Delete(&ContentItem{}, contentItem.ID).Error
	assert.NoError(t, err)

	// Verify deletion
	var deletedItem ContentItem
	err = db.First(&deletedItem, contentItem.ID).Error
	assert.Error(t, err, "Record should not be found after deletion")
}
