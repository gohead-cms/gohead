// single_item_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gohead/pkg/logger"
)

func TestValidateSingleItemValues(t *testing.T) {
	logger.InitLogger("info") // or "debug", depending on how much log noise you want
	db := setupDatabase(t)
	assert.NotNil(t, db, "DB instance should not be nil")

	//
	// 1. Create a 'target' collection for relationship testing
	//
	targetCollection := Collection{
		Name: "articles",
	}
	err := db.Create(&targetCollection).Error
	require.NoError(t, err, "failed to create target collection 'articles'")

	// Insert a couple of Items in 'articles' so we can reference them
	item1 := Item{CollectionID: targetCollection.ID}
	item2 := Item{CollectionID: targetCollection.ID}
	require.NoError(t, db.Create(&item1).Error)
	require.NoError(t, db.Create(&item2).Error)

	//
	// 2. Define a SingleType (schema) with various attributes
	//
	singleType := SingleType{
		Name: "homepage",
		Attributes: []Attribute{
			{
				Name:     "title",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "featuredArticle",
				Type:     "relation",
				Relation: "oneToOne",
				Target:   "articles",
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
		data := map[string]any{
			// "title" is missing
			"featuredArticle": float64(item1.ID), // 1
		}
		err := ValidateSingleItemValues(singleType, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required attribute: 'title'")
	})

	t.Run("Valid data with single (oneToOne) relationship", func(t *testing.T) {
		data := map[string]any{
			"title":           "Hello World",
			"featuredArticle": float64(item1.ID), // referencing an existing item in 'articles'
		}
		err := ValidateSingleItemValues(singleType, data)
		assert.NoError(t, err, "should pass validation for correct single reference")
	})

	t.Run("Invalid single relationship reference", func(t *testing.T) {
		data := map[string]any{
			"title":           "Hello World",
			"featuredArticle": float64(999), // non-existent item ID
		}
		err := ValidateSingleItemValues(singleType, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "referenced item with ID '999' in collection 'articles' does not exist")
	})

	t.Run("Many-to-many relationship with valid IDs", func(t *testing.T) {
		data := map[string]any{
			"title":           "Hello World",
			"featuredArticle": float64(item1.ID),
			"relatedArticles": []any{float64(item1.ID), float64(item2.ID)},
		}
		err := ValidateSingleItemValues(singleType, data)
		assert.NoError(t, err, "should pass with valid ID references in array")
	})

	t.Run("Many-to-many relationship with invalid element", func(t *testing.T) {
		data := map[string]any{
			"title":           "Hello World",
			"featuredArticle": float64(item1.ID),
			"relatedArticles": []any{
				float64(item1.ID),
				map[string]any{"id": 999}, // not a pure ID or float
			},
		}
		err := ValidateSingleItemValues(singleType, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid element in relationship array for 'relatedArticles'")
	})

	t.Run("Many-to-many relationship with missing item", func(t *testing.T) {
		data := map[string]any{
			"title":           "Hello World",
			"featuredArticle": float64(item1.ID),
			"relatedArticles": []any{float64(item1.ID), float64(999)},
		}
		err := ValidateSingleItemValues(singleType, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "referenced item with ID '999' in collection 'articles' does not exist")
	})

	t.Run("Valid data with no references (only required field)", func(t *testing.T) {
		data := map[string]any{
			"title": "Minimal",
		}
		err := ValidateSingleItemValues(singleType, data)
		assert.NoError(t, err, "should pass validation with only the required attribute set")
	})
}

func TestCheckItemExists_NoSuchCollection(t *testing.T) {

	err := checkItemExists(999, 1)
	assert.Error(t, err, "collection 999 does not exist, so it can't have item 1")
}
