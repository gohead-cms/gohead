package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
)

func TestSaveContentRelations(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.ContentRelation{}, &models.ContentItem{}, &models.ContentType{}))

	// Prepare content type with relationships
	ct := models.ContentType{
		Name: "articles",
		Relationships: []models.Relationship{
			{
				FieldName:    "author",
				RelationType: "one-to-one",
				RelatedType:  "users",
			},
			{
				FieldName:    "tags",
				RelationType: "many-to-many",
				RelatedType:  "tags",
			},
		},
	}

	// Save test data
	itemID := uint(1)
	itemData := map[string]interface{}{
		"author": float64(42),                  // Simulating a one-to-one relationship with user ID 42
		"tags":   []interface{}{1.0, 2.0, 3.0}, // Simulating a many-to-many relationship with tags
	}

	// Save content relations
	err := SaveContentRelations(&ct, itemID, itemData)
	assert.NoError(t, err)

	// Verify one-to-one relation
	var oneToOneRelation models.ContentRelation
	err = db.Where("content_type = ? AND content_item_id = ? AND field_name = ?", "articles", itemID, "author").First(&oneToOneRelation).Error
	assert.NoError(t, err)
	assert.Equal(t, uint(42), oneToOneRelation.RelatedItemID)
	assert.Equal(t, "one-to-one", oneToOneRelation.RelationType)

	// Verify many-to-many relations
	var manyToManyRelations []models.ContentRelation
	err = db.Where("content_type = ? AND content_item_id = ? AND field_name = ?", "articles", itemID, "tags").Find(&manyToManyRelations).Error
	assert.NoError(t, err)
	assert.Len(t, manyToManyRelations, 3)
	assert.Equal(t, uint(1), manyToManyRelations[0].RelatedItemID)
	assert.Equal(t, uint(2), manyToManyRelations[1].RelatedItemID)
	assert.Equal(t, uint(3), manyToManyRelations[2].RelatedItemID)
}

func TestGetContentRelations(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.ContentRelation{}))

	// Insert content relations
	relations := []models.ContentRelation{
		{
			ContentType:   "articles",
			ContentItemID: 1,
			RelatedType:   "users",
			RelatedItemID: 42,
			RelationType:  "one-to-one",
			FieldName:     "author",
		},
		{
			ContentType:   "articles",
			ContentItemID: 1,
			RelatedType:   "tags",
			RelatedItemID: 1,
			RelationType:  "many-to-many",
			FieldName:     "tags",
		},
		{
			ContentType:   "articles",
			ContentItemID: 1,
			RelatedType:   "tags",
			RelatedItemID: 2,
			RelationType:  "many-to-many",
			FieldName:     "tags",
		},
	}
	assert.NoError(t, db.Create(&relations).Error)

	// Retrieve content relations
	result, err := GetContentRelations("articles", 1)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Validate retrieved relations
	assert.Equal(t, "users", result[0].RelatedType)
	assert.Equal(t, uint(42), result[0].RelatedItemID)
	assert.Equal(t, "tags", result[1].RelatedType)
	assert.Equal(t, uint(1), result[1].RelatedItemID)
	assert.Equal(t, "tags", result[2].RelatedType)
	assert.Equal(t, uint(2), result[2].RelatedItemID)
}
