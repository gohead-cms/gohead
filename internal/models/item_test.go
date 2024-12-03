// internal/models/content_item_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestItemCRUD(t *testing.T) {
	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Create a Item
	Item1 := Item{
		CollectionID: 1,
		Data: JSONMap{
			"title":   "Test Article",
			"content": "This is the content of the test article.",
		},
	}

	// Auto-migrate the Item model
	err = db.AutoMigrate(&Item{})
	assert.NoError(t, err)

	err = db.Create(&Item1).Error
	assert.NoError(t, err)
	assert.NotZero(t, Item1.ID, "Item ID should be set after creation")

	// Retrieve the Item
	var retrievedItem Item
	err = db.First(&retrievedItem, Item1.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "articles", retrievedItem.CollectionID)
	assert.Equal(t, "Test Article", retrievedItem.Data["title"])
	assert.Equal(t, "This is the content of the test article.", retrievedItem.Data["content"])

	// Update the Item
	err = db.Model(&retrievedItem).Update("Data", JSONMap{
		"title":   "Updated Title",
		"content": "Updated content.",
	}).Error
	assert.NoError(t, err)

	// Verify the update
	var updatedItem Item
	err = db.First(&updatedItem, Item1.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedItem.Data["title"])
	assert.Equal(t, "Updated content.", updatedItem.Data["content"])

	// Delete the Item
	err = db.Delete(&Item{}, Item1.ID).Error
	assert.NoError(t, err)

	// Verify deletion
	var deletedItem Item
	err = db.First(&deletedItem, Item1.ID).Error
	assert.Error(t, err, "Record should not be found after deletion")
}
