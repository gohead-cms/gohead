package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
)

func TestSaveRelationships(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Relationship{}, &models.Item{}, &models.Collection{}))

	// Prepare content type with Relationships
	ct := models.Collection{
		Name: "articles",
		Relationships: []models.Relationship{
			{
				FieldName:         "author",
				RelationType:      "one-to-one",
				RelatedCollection: 2,
			},
			{
				FieldName:         "tags",
				RelationType:      "many-to-many",
				RelatedCollection: 3,
			},
		},
	}

	// Save test data
	itemID := uint(1)
	itemData := map[string]interface{}{
		"author": float64(42),                  // Simulating a one-to-one Relationship with user ID 42
		"tags":   []interface{}{1.0, 2.0, 3.0}, // Simulating a many-to-many Relationship with tags
	}

	// Save content relations
	err := SaveRelationship(&ct, itemID, itemData)
	assert.NoError(t, err)

	// Verify one-to-one relation
	var oneToOneRelation models.Relationship
	err = db.Where("content_type = ? AND content_item_id = ? AND field_name = ?", "articles", itemID, "author").First(&oneToOneRelation).Error
	assert.NoError(t, err)
	assert.Equal(t, uint(42), oneToOneRelation.RelatedItemID)
	assert.Equal(t, "one-to-one", oneToOneRelation.RelationType)

	// Verify many-to-many relations
	var manyToManyRelations []models.Relationship
	err = db.Where("content_type = ? AND content_item_id = ? AND field_name = ?", "articles", itemID, "tags").Find(&manyToManyRelations).Error
	assert.NoError(t, err)
	assert.Len(t, manyToManyRelations, 3)
	assert.Equal(t, uint(1), manyToManyRelations[0].RelatedItemID)
	assert.Equal(t, uint(2), manyToManyRelations[1].RelatedItemID)
	assert.Equal(t, uint(3), manyToManyRelations[2].RelatedItemID)
}

func TestGetRelationships(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Relationship{}))

	// Insert content relations
	relations := []models.Relationship{
		{
			CollectionID:      1,
			ItemID:            1,
			RelatedCollection: 2,
			RelatedItemID:     42,
			RelationType:      "one-to-one",
			FieldName:         "author",
		},
		{
			CollectionID:      1,
			ItemID:            1,
			RelatedCollection: 3,
			RelatedItemID:     1,
			RelationType:      "many-to-many",
			FieldName:         "tags",
		},
		{
			CollectionID:      1,
			ItemID:            1,
			RelatedCollection: 3,
			RelatedItemID:     2,
			RelationType:      "many-to-many",
			FieldName:         "tags",
		},
	}
	assert.NoError(t, db.Create(&relations).Error)

	// Retrieve content relations
	result, err := GetRelationships("articles", 1)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Validate retrieved relations
	assert.Equal(t, "users", result[0].RelatedCollection)
	assert.Equal(t, uint(42), result[0].RelatedItemID)
	assert.Equal(t, "tags", result[1].RelatedCollection)
	assert.Equal(t, uint(1), result[1].RelatedItemID)
	assert.Equal(t, "tags", result[2].RelatedCollection)
	assert.Equal(t, uint(2), result[2].RelatedItemID)
}
