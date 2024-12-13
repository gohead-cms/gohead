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

	// Prepare a collection with relationships
	collection := models.Collection{
		Name: "articles",
		Relationships: []models.Relationship{
			{
				Field:        "author",
				RelationType: "one-to-one",
			},
			{
				Field:        "tags",
				RelationType: "many-to-many",
			},
		},
	}
	assert.NoError(t, db.Create(&collection).Error)

	// Define the content item ID and data
	itemID := uint(1)
	itemData := models.JSONMap{
		"author": float64(42),                  // Simulating a one-to-one relationship with user ID 42
		"tags":   []interface{}{1.0, 2.0, 3.0}, // Simulating a many-to-many relationship with tag IDs
	}

	// Save relationships
	err := SaveRelationship(&collection, itemID, itemData)
	assert.NoError(t, err)

	// Verify one-to-one relationship
	var oneToOneRelation models.Relationship
	err = db.Where("collection_id = ? AND source_item_id = ? AND field_name = ?", collection.ID, itemID, "author").First(&oneToOneRelation).Error
	assert.NoError(t, err)
	assert.Equal(t, uint(42), oneToOneRelation.SourceItemID)
	assert.Equal(t, "one-to-one", oneToOneRelation.RelationType)

	// Verify many-to-many relationships
	var manyToManyRelations []models.Relationship
	err = db.Where("collection_id = ? AND source_item_id = ? AND field_name = ?", collection.ID, itemID, "tags").Find(&manyToManyRelations).Error
	assert.NoError(t, err)
	assert.Len(t, manyToManyRelations, 3)
	assert.Equal(t, uint(1), manyToManyRelations[0].SourceItemID)
	assert.Equal(t, uint(2), manyToManyRelations[1].SourceItemID)
	assert.Equal(t, uint(3), manyToManyRelations[2].SourceItemID)
}

func TestGetRelationships(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Relationship{}))

	var source_item_id uint
	source_item_id = 1
	// Insert relationships
	relations := []models.Relationship{
		{
			CollectionID: 1,
			SourceItemID: &source_item_id,
			RelationType: "one-to-one",
			Field:        "author",
		},
		{
			CollectionID: 1,
			SourceItemID: &source_item_id,
			RelationType: "many-to-many",
			Field:        "tags",
		},
		{
			CollectionID: 1,
			SourceItemID: &source_item_id,
			RelationType: "many-to-many",
			Field:        "tags",
		},
	}
	assert.NoError(t, db.Create(&relations).Error)

	// Retrieve relationships
	result, err := GetRelationships(1, 1)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Validate retrieved relationships
	assert.Equal(t, uint(42), result[0].SourceItemID)
	assert.Equal(t, uint(1), result[1].SourceItemID)
	assert.Equal(t, uint(2), result[2].SourceItemID)
	assert.Equal(t, "one-to-one", result[0].RelationType)
	assert.Equal(t, "many-to-many", result[1].RelationType)
	assert.Equal(t, "many-to-many", result[2].RelationType)
}
