// internal/models/content_relation_test.go
package models

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Initialize logger for testing
func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestRelationshipCRUD(t *testing.T) {
	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the models
	err = db.AutoMigrate(&Item{}, &Relationship{})
	assert.NoError(t, err)

	// Create two Items to establish a relation
	Item1 := Item{
		CollectionID: 1,
		Data: JSONMap{
			"title":   "Parent Article",
			"content": "This is the parent article.",
		},
	}
	Item2 := Item{
		CollectionID: 2,
		Data: JSONMap{
			"author":  "John Doe",
			"content": "This is a comment.",
		},
	}
	err = db.Create(&Item1).Error
	assert.NoError(t, err)
	err = db.Create(&Item2).Error
	assert.NoError(t, err)

	// Create a Relationship
	Relationship1 := Relationship{
		CollectionID:      1,
		ItemID:            Item1.ID,
		RelatedCollection: 2,
		RelatedItemID:     Item2.ID,
		RelationType:      "one-to-many",
		FieldName:         "comments",
	}
	err = db.Create(&Relationship1).Error
	assert.NoError(t, err)
	assert.NotZero(t, Relationship1.ID, "Relationship ID should be set after creation")

	// Retrieve the Relationship
	var retrievedRelation Relationship
	err = db.First(&retrievedRelation, Relationship1.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, Relationship1.CollectionID, retrievedRelation.CollectionID)
	assert.Equal(t, Relationship1.RelationType, retrievedRelation.RelationType)
	assert.Equal(t, Relationship1.FieldName, retrievedRelation.FieldName)

	// Verify the associated Items
	var parentItem Item
	var relatedItem Item
	err = db.First(&parentItem, retrievedRelation.ItemID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Parent Article", parentItem.Data["title"])

	err = db.First(&relatedItem, retrievedRelation.RelatedItemID).Error
	assert.NoError(t, err)
	assert.Equal(t, "This is a comment.", relatedItem.Data["content"])

	// Update the Relationship
	err = db.Model(&retrievedRelation).Update("RelationType", "many-to-many").Error
	assert.NoError(t, err)

	// Verify the update
	var updatedRelation Relationship
	err = db.First(&updatedRelation, Relationship1.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "many-to-many", updatedRelation.RelationType)

	// Delete the Relationship
	err = db.Delete(&Relationship{}, Relationship1.ID).Error
	assert.NoError(t, err)

	// Verify deletion
	var deletedRelation Relationship
	err = db.First(&deletedRelation, Relationship1.ID).Error
	assert.Error(t, err, "Record should not be found after deletion")
}
