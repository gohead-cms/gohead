// single_item_test.go
package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

func TestSingleItemStorage(t *testing.T) {
	logger.InitLogger("silent") // or "debug" if you want verbose logs
	db := setupTestDB(t)
	assert.NotNil(t, db)

	//
	// 1. Create a SingleType with a required attribute
	//
	singleType := models.SingleType{
		Name: "homepage",
		Attributes: []models.Attribute{
			{
				Name:     "title",
				Type:     "string",
				Required: true,
			},
		},
	}
	err := db.Create(&singleType).Error
	require.NoError(t, err, "failed to create singleType 'homepage'")

	t.Run("CreateSingleItem - success", func(t *testing.T) {
		content := map[string]interface{}{
			"title": "Welcome to Our Site",
		}
		item, err := storage.CreateSingleItem(&singleType, content)
		assert.NoError(t, err, "should create single item without conflict")
		assert.NotNil(t, item, "item should not be nil")
		assert.NotZero(t, item.ID, "new single item should have an ID")
		assert.Equal(t, singleType.ID, item.SingleTypeID)
		assert.Equal(t, content["title"], item.Data["title"])
	})

	t.Run("CreateSingleItem - conflict", func(t *testing.T) {
		// Attempt to create another item for the same singleType => conflict
		duplicateContent := map[string]interface{}{
			"title": "Another Attempt",
		}
		item, err := storage.CreateSingleItem(&singleType, duplicateContent)
		assert.Error(t, err, "should fail because an item already exists for 'homepage'")
		assert.Nil(t, item)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("GetSingleItemByType - success", func(t *testing.T) {
		item, err := storage.GetSingleItemByType("homepage")
		assert.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "Welcome to Our Site", item.Data["title"])
	})

	t.Run("GetSingleItemByType - single type not found", func(t *testing.T) {
		item, err := storage.GetSingleItemByType("nonexistent")
		assert.Nil(t, item)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "single type 'nonexistent' not found")
	})

	t.Run("UpdateSingleItem - success", func(t *testing.T) {
		// Create new data
		newData := map[string]interface{}{
			"title": "Updated Homepage Title",
		}

		updatedItem, err := storage.UpdateSingleItem("homepage", newData)
		assert.NoError(t, err)
		assert.NotNil(t, updatedItem)
		assert.Equal(t, "Updated Homepage Title", updatedItem.Data["title"])

		// Verify in DB
		check, err := storage.GetSingleItemByType("homepage")
		assert.NoError(t, err)
		assert.Equal(t, "Updated Homepage Title", check.Data["title"])
	})

	t.Run("UpdateSingleItem - single type not found", func(t *testing.T) {
		newData := map[string]interface{}{
			"title": "Won't matter",
		}
		updatedItem, err := storage.UpdateSingleItem("unknown", newData)
		assert.Nil(t, updatedItem)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "single type 'unknown' not found")
	})

	t.Run("UpdateSingleItem - no existing single item", func(t *testing.T) {
		// Create a new single type
		anotherType := models.SingleType{
			Name: "about-us",
			Attributes: []models.Attribute{
				{Name: "title", Type: "string", Required: true},
			},
		}
		err := db.Create(&anotherType).Error
		require.NoError(t, err, "failed to create singleType 'about-us'")

		newData := map[string]interface{}{
			"title": "About Us Page",
		}
		updatedItem, err := storage.UpdateSingleItem("about-us", newData)
		assert.Nil(t, updatedItem)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no existing single item for single type 'about-us'")
	})

	t.Run("DeleteSingleItem - success", func(t *testing.T) {
		// Delete the item for 'homepage'
		err := storage.DeleteSingleItem("homepage")
		assert.NoError(t, err, "should delete the single item successfully")

		// After deletion, retrieval should fail
		item, getErr := storage.GetSingleItemByType("homepage")
		assert.Nil(t, item)
		assert.Error(t, getErr)
		assert.Contains(t, getErr.Error(), "no single item found for single type 'homepage'")
	})

	t.Run("DeleteSingleItem - single type not found", func(t *testing.T) {
		err := storage.DeleteSingleItem("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "single type 'nonexistent' not found")
	})

	t.Run("DeleteSingleItem - item already deleted", func(t *testing.T) {
		err := storage.DeleteSingleItem("homepage")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no single item found for single type 'homepage'")
	})
}

func TestCheckValidationFails(t *testing.T) {
	db := setupTestDB(t)

	// Create SingleType with a required attribute
	st := models.SingleType{
		Name: "contact",
		Attributes: []models.Attribute{
			{Name: "email", Type: "string", Required: true},
		},
	}
	require.NoError(t, db.Create(&st).Error)

	// Attempt creation with missing "email"
	itemData := map[string]interface{}{}
	item, err := storage.CreateSingleItem(&st, itemData)
	assert.Nil(t, item)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required attribute: 'email'")
}

// TODO
// Additional tests can be added for relationships, pattern checks, etc.
//
