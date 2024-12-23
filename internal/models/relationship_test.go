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
	err = db.AutoMigrate(&Collection{}, &Item{}, &Relationship{})
	assert.NoError(t, err)

	// Create Collections
	collection1 := Collection{Name: "articles"}
	collection2 := Collection{Name: "comments"}
	err = db.Create(&collection1).Error
	assert.NoError(t, err)
	err = db.Create(&collection2).Error
	assert.NoError(t, err)

	// Create Items in Collections
	item1 := Item{
		CollectionID: collection1.ID,
		Data: JSONMap{
			"title":   "Parent Article",
			"content": "This is the parent article.",
		},
	}
	item2 := Item{
		CollectionID: collection2.ID,
		Data: JSONMap{
			"author":  "John Doe",
			"content": "This is a comment.",
		},
	}
	err = db.Create(&item1).Error
	assert.NoError(t, err)
	err = db.Create(&item2).Error
	assert.NoError(t, err)

	// Create a Relationship
	relationship := Relationship{
		Attribute:    "comments",
		RelationType: "one-to-many",
		CollectionID: collection1.ID,
		SourceItemID: &item1.ID,
	}
	err = db.Create(&relationship).Error
	assert.NoError(t, err)
	assert.NotZero(t, relationship.ID, "Relationship ID should be set after creation")

	// Retrieve the Relationship
	var retrievedRelation Relationship
	err = db.Preload("SourceItem").Preload("TargetItem").First(&retrievedRelation, relationship.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, relationship.RelationType, retrievedRelation.RelationType)
	assert.Equal(t, relationship.Attribute, retrievedRelation.Attribute)
	assert.Equal(t, item1.ID, retrievedRelation.SourceItem.ID)

	// Verify the associated Items
	var parentItem Item
	var relatedItem Item
	err = db.First(&parentItem, retrievedRelation.SourceItemID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Parent Article", parentItem.Data["title"])

	err = db.First(&relatedItem, retrievedRelation.SourceItemID).Error
	assert.NoError(t, err)
	assert.Equal(t, "This is a comment.", relatedItem.Data["content"])

	// Update the Relationship
	err = db.Model(&retrievedRelation).Update("RelationType", "many-to-many").Error
	assert.NoError(t, err)

	// Verify the update
	var updatedRelation Relationship
	err = db.First(&updatedRelation, relationship.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "many-to-many", updatedRelation.RelationType)

	// Delete the Relationship
	err = db.Delete(&Relationship{}, relationship.ID).Error
	assert.NoError(t, err)

	// Verify deletion
	var deletedRelation Relationship
	err = db.First(&deletedRelation, relationship.ID).Error
	assert.Error(t, err, "Record should not be found after deletion")
}
