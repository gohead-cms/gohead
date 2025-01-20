// single_item_test.go
package models_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to in-memory database")

	database.DB = db // point our package-wide DB reference to this test DB

	// Migrate only what's needed for the test: Collections, Items, SingleTypes, Attributes
	err = db.AutoMigrate(&models.Collection{}, &models.Item{},
		&models.SingleType{}, &models.SingleItem{}, &models.Attribute{})
	require.NoError(t, err, "failed to auto-migrate schema")

	return db
}

func TestValidateSingleItemValues(t *testing.T) {
	logger.InitLogger("silent") // or "debug", depending on how much log noise you want
	db := setupTestDB(t)
	assert.NotNil(t, db, "DB instance should not be nil")

	//
	// 1. Create a 'target' collection for relationship testing
	//
	targetCollection := models.Collection{
		Name: "articles",
	}
	err := db.Create(&targetCollection).Error
	require.NoError(t, err, "failed to create target collection 'articles'")

	// Insert a couple of Items in 'articles' so we can reference them
	item1 := models.Item{CollectionID: targetCollection.ID}
	item2 := models.Item{CollectionID: targetCollection.ID}
	require.NoError(t, db.Create(&item1).Error)
	require.NoError(t, db.Create(&item2).Error)

	//
	// 2. Define a SingleType (schema) with various attributes
	//
	singleType := models.SingleType{
		Name: "homepage",
		Attributes: []models.Attribute{
			{
				Name:     "title",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "featuredArticle",
				Type:     "relation",
				Relation: "oneToOne",
				Target:   "articles", // referencing the "articles" collection
			},
			{
				Name:     "relatedArticles",
				Type:     "relation",
				Relation: "manyToMany",
				Target:   "articles",
			},
		},
	}
	require.NoError(t, db.Create(&singleType).Error, "failed to create singleType in DB")

	//
	// 3. Test Cases
	//
	t.Run("Missing required attribute", func(t *testing.T) {
		data := map[string]interface{}{
			// "title" is missing
			"featuredArticle": float64(item1.ID), // 1
		}
		err := models.ValidateSingleItemValues(singleType, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required attribute: 'title'")
	})

	t.Run("Valid data with single (oneToOne) relationship", func(t *testing.T) {
		data := map[string]interface{}{
			"title":           "Hello World",
			"featuredArticle": float64(item1.ID), // referencing an existing item in 'articles'
		}
		err := models.ValidateSingleItemValues(singleType, data)
		assert.NoError(t, err, "should pass validation for correct single reference")
	})

	t.Run("Invalid single relationship reference", func(t *testing.T) {
		data := map[string]interface{}{
			"title":           "Hello World",
			"featuredArticle": float64(999), // non-existent item ID
		}
		err := models.ValidateSingleItemValues(singleType, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "referenced item with ID '999' in collection 'articles' does not exist")
	})

	t.Run("Many-to-many relationship with valid IDs", func(t *testing.T) {
		data := map[string]interface{}{
			"title":           "Hello World",
			"featuredArticle": float64(item1.ID),
			"relatedArticles": []interface{}{float64(item1.ID), float64(item2.ID)},
		}
		err := models.ValidateSingleItemValues(singleType, data)
		assert.NoError(t, err, "should pass with valid ID references in array")
	})

	t.Run("Many-to-many relationship with invalid element", func(t *testing.T) {
		data := map[string]interface{}{
			"title":           "Hello World",
			"featuredArticle": float64(item1.ID),
			"relatedArticles": []interface{}{
				float64(item1.ID),
				map[string]interface{}{"id": 999}, // not a pure ID or float
			},
		}
		err := models.ValidateSingleItemValues(singleType, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid element in relationship array for 'relatedArticles'")
	})

	t.Run("Many-to-many relationship with missing item", func(t *testing.T) {
		data := map[string]interface{}{
			"title":           "Hello World",
			"featuredArticle": float64(item1.ID),
			"relatedArticles": []interface{}{float64(item1.ID), float64(999)},
		}
		err := models.ValidateSingleItemValues(singleType, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "referenced item with ID '999' in collection 'articles' does not exist")
	})

	t.Run("Valid data with no references (only required field)", func(t *testing.T) {
		data := map[string]interface{}{
			"title": "Minimal",
		}
		err := models.ValidateSingleItemValues(singleType, data)
		assert.NoError(t, err, "should pass validation with only the required attribute set")
	})
}

func TestCheckItemExists_NoSuchCollection(t *testing.T) {
	// If you'd like to test checkItemExists separately, you can do so,
	// but typically it's covered by the relationship tests above.

	// For example:
	err := checkItemExists(999, 1)
	assert.Error(t, err, "collection 999 does not exist, so it can't have item 1")
}

// If you want to test checkItemExists directly, you need to replicate the function
// from your code or make it exported for direct usage.
func checkItemExists(collectionID uint, itemID uint) error {
	var count int64
	err := database.DB.Model(&models.Item{}).
		Where("collection_id = ? AND id = ?", collectionID, itemID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("item not found in the specified collection")
	}
	return nil
}
