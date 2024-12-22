package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
	"gitlab.com/sudo.bngz/gohead/pkg/validation"
)

func TestSaveItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}, &models.Collection{}))

	// Create a sample collection
	collection := &models.Collection{Name: "articles"}
	assert.NoError(t, db.Create(collection).Error)

	// Create a sample content item
	item := &models.Item{
		CollectionID: collection.ID,
		Data: models.JSONMap{
			"title":   "Sample Article",
			"content": "This is a test article.",
		},
	}

	// Save the content item
	err := SaveItem(item)
	assert.NoError(t, err)
	assert.NotZero(t, item.ID)

	// Verify the content item exists in the database
	var result models.Item
	err = db.First(&result, item.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Sample Article", result.Data["title"])
}

func TestGetItemByID(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}, &models.Collection{}))

	// Create a collection and content item
	collection := &models.Collection{Name: "articles"}
	assert.NoError(t, db.Create(collection).Error)

	item := &models.Item{
		CollectionID: collection.ID,
		Data: models.JSONMap{
			"title":   "Test Article",
			"content": "This is a test.",
		},
	}
	assert.NoError(t, db.Create(item).Error)

	// Fetch the content item by ID
	result, err := GetItemByID(item.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Article", result.Data["title"])
}

func TestGetItems(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}, &models.Collection{}))

	// Create a collection and multiple items
	collection := &models.Collection{Name: "articles"}
	assert.NoError(t, db.Create(collection).Error)

	items := []models.Item{
		{CollectionID: collection.ID, Data: models.JSONMap{"title": "Article 1"}},
		{CollectionID: collection.ID, Data: models.JSONMap{"title": "Article 2"}},
	}
	assert.NoError(t, db.Create(&items).Error)

	// Fetch all content items
	results, err := GetItems(collection.ID)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Article 1", results[0].Data["title"])
	assert.Equal(t, "Article 2", results[1].Data["title"])
}

func TestUpdateItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}, &models.Relationship{}, &models.Collection{}))

	// Create a collection
	collection := &models.Collection{Name: "articles"}
	assert.NoError(t, db.Create(collection).Error)

	// Create and save a content item
	item := &models.Item{
		CollectionID: collection.ID,
		Data:         models.JSONMap{"title": "Old Title", "content": "Old Content"},
	}
	assert.NoError(t, db.Create(item).Error)

	// Create and save relationships for the item
	relationship := &models.Relationship{
		SourceItemID: &item.ID,
		RelationType: "one-to-one",
		Field:        "related_field",
	}
	assert.NoError(t, db.Create(relationship).Error)

	// Update the content item
	updatedData := models.JSONMap{
		"title":         "New Title",
		"content":       "New Content",
		"related_field": map[string]interface{}{"title": "Nested Title"},
	}
	err := UpdateItem(collection, item.ID, updatedData)
	assert.NoError(t, err)

	// Verify the content item is updated
	var updatedItem models.Item
	assert.NoError(t, db.First(&updatedItem, item.ID).Error)
	assert.Equal(t, "New Title", updatedItem.Data["title"])
	assert.Equal(t, "New Content", updatedItem.Data["content"])

	// Verify relationships are updated
	var updatedRelationships []models.Relationship
	assert.NoError(t, db.Where("source_item_id = ?", item.ID).Find(&updatedRelationships).Error)
	assert.Len(t, updatedRelationships, 1)
	assert.Equal(t, "related_field", updatedRelationships[0].Field)

	// Verify nested item creation
	var nestedItem models.Item
	assert.NoError(t, db.Where("data->>'title' = ?", "Nested Title").First(&nestedItem).Error)
	assert.NotZero(t, nestedItem.ID)
}

func TestDeleteItem(t *testing.T) {
	// Initialize in-memory test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.Item{}, &models.Relationship{}, &models.Collection{}))

	// Create and save a collection
	collection := &models.Collection{Name: "articles"}
	assert.NoError(t, db.Create(collection).Error)

	// Create and save a content item
	item := &models.Item{
		CollectionID: collection.ID,
		Data:         models.JSONMap{"title": "Test Title", "content": "Test Content"},
	}
	assert.NoError(t, db.Create(item).Error)

	// Delete the content item
	err := DeleteItem(item.ID)
	assert.NoError(t, err)

	// Verify the content item is deleted
	var deletedItem models.Item
	err = db.First(&deletedItem, item.ID).Error
	assert.Error(t, err) // Should return an error because the item no longer exists

	// Verify relationships are deleted
	var deletedRelationships []models.Relationship
	err = db.Where("source_item_id = ?", item.ID).Find(&deletedRelationships).Error
	assert.NoError(t, err)
	assert.Len(t, deletedRelationships, 0)
}

func TestCheckFieldUniqueness(t *testing.T) {
	// Setup the test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	err := db.AutoMigrate(&models.Collection{},
		&models.Field{},
		&models.Relationship{},
		&models.Item{},
	)
	assert.NoError(t, err)

	// Insert test data
	assert.NoError(t, db.Exec(`
		INSERT INTO items (collection_id, data) VALUES
		(1, '{"email": "existing@example.com"}'),
		(1, '{"email": "duplicate@example.com"}'),
		(2, '{"email": "unique@example.com"}');
	`).Error)

	// Test cases
	t.Run("Unique Value in Collection", func(t *testing.T) {
		err := validation.CheckFieldUniqueness(1, "email", "new@example.com")
		assert.NoError(t, err, "Expected no error for unique value")
	})

	t.Run("Duplicate Value in Same Collection", func(t *testing.T) {
		err := validation.CheckFieldUniqueness(1, "email", "duplicate@example.com")
		assert.Error(t, err, "Expected an error for duplicate value")
		assert.Equal(t, "value for field 'email' must be unique", err.Error())
	})

	t.Run("Duplicate Value in Different Collection", func(t *testing.T) {
		err := validation.CheckFieldUniqueness(2, "email", "duplicate@example.com")
		assert.NoError(t, err, "Expected no error for duplicate value in a different collection")
	})

	t.Run("Non-Existent Field", func(t *testing.T) {
		err := validation.CheckFieldUniqueness(1, "nonexistent_field", "value")
		assert.NoError(t, err, "Expected no error for non-existent field")
	})
}
