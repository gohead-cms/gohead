// internal/models/content_relation_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestContentRelationCRUD(t *testing.T) {
	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the models
	err = db.AutoMigrate(&ContentItem{}, &ContentRelation{})
	assert.NoError(t, err)

	// Create two ContentItems to establish a relation
	contentItem1 := ContentItem{
		ContentType: "articles",
		Data: JSONMap{
			"title":   "Parent Article",
			"content": "This is the parent article.",
		},
	}
	contentItem2 := ContentItem{
		ContentType: "comments",
		Data: JSONMap{
			"author":  "John Doe",
			"content": "This is a comment.",
		},
	}
	err = db.Create(&contentItem1).Error
	assert.NoError(t, err)
	err = db.Create(&contentItem2).Error
	assert.NoError(t, err)

	// Create a ContentRelation
	contentRelation := ContentRelation{
		ContentType:   "articles",
		ContentItemID: contentItem1.ID,
		RelatedType:   "comments",
		RelatedItemID: contentItem2.ID,
		RelationType:  "one-to-many",
		FieldName:     "comments",
	}
	err = db.Create(&contentRelation).Error
	assert.NoError(t, err)
	assert.NotZero(t, contentRelation.ID, "ContentRelation ID should be set after creation")

	// Retrieve the ContentRelation
	var retrievedRelation ContentRelation
	err = db.First(&retrievedRelation, contentRelation.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, contentRelation.ContentType, retrievedRelation.ContentType)
	assert.Equal(t, contentRelation.RelationType, retrievedRelation.RelationType)
	assert.Equal(t, contentRelation.FieldName, retrievedRelation.FieldName)

	// Verify the associated ContentItems
	var parentItem ContentItem
	var relatedItem ContentItem
	err = db.First(&parentItem, retrievedRelation.ContentItemID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Parent Article", parentItem.Data["title"])

	err = db.First(&relatedItem, retrievedRelation.RelatedItemID).Error
	assert.NoError(t, err)
	assert.Equal(t, "This is a comment.", relatedItem.Data["content"])

	// Update the ContentRelation
	err = db.Model(&retrievedRelation).Update("RelationType", "many-to-many").Error
	assert.NoError(t, err)

	// Verify the update
	var updatedRelation ContentRelation
	err = db.First(&updatedRelation, contentRelation.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "many-to-many", updatedRelation.RelationType)

	// Delete the ContentRelation
	err = db.Delete(&ContentRelation{}, contentRelation.ID).Error
	assert.NoError(t, err)

	// Verify deletion
	var deletedRelation ContentRelation
	err = db.First(&deletedRelation, contentRelation.ID).Error
	assert.Error(t, err, "Record should not be found after deletion")
}
