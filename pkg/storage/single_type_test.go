// single_type_test.go
package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// setupTestDB initializes an in-memory SQLite database for testing SingleType storage.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to in-memory database")

	// Assign test DB to the global package variable
	database.DB = db

	// Automigrate the necessary models
	err = db.AutoMigrate(&models.SingleType{}, &models.Attribute{})
	require.NoError(t, err, "Failed to migrate database")

	return db
}

func TestSingleTypeStorage(t *testing.T) {
	logger.InitLogger("silent") // Mute logs or set desired level

	db := setupTestDB(t)
	assert.NotNil(t, db, "Database instance should not be nil")

	t.Run("SaveSingleType - create new single type", func(t *testing.T) {
		st := &models.SingleType{
			Name:        "homepage",
			Description: "Main homepage settings",
			Attributes: []models.Attribute{
				{
					Name:         "title",
					Type:         "string",
					Required:     true,
					SingleTypeID: nil,
				},
				{
					Name:         "heroText",
					Type:         "richtext",
					SingleTypeID: nil,
				},
			},
		}

		err := storage.SaveOrUpdateSingleType(st)
		assert.NoError(t, err, "Should create single type without errors")
		assert.NotZero(t, st.ID, "Newly created single type should have an ID")

		// Check DB
		var fetched models.SingleType
		err = db.Preload("Attributes").First(&fetched, "name = ?", "homepage").Error
		assert.NoError(t, err, "Should fetch newly created single type from DB")
		assert.Equal(t, "homepage", fetched.Name)
		assert.Len(t, fetched.Attributes, 2)
	})

	t.Run("SaveSingleType - conflict with existing name", func(t *testing.T) {
		st := &models.SingleType{
			Name:        "homepage", // already created in previous test
			Description: "Attempting duplicate creation",
		}

		err := storage.SaveOrUpdateSingleType(st)
		assert.Error(t, err, "Should fail when creating single type with existing non-deleted name")
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("GetSingleTypeByName - success", func(t *testing.T) {
		st, err := storage.GetSingleTypeByName("homepage")
		assert.NoError(t, err, "Should retrieve single type by name")
		assert.Equal(t, "homepage", st.Name)
		assert.Equal(t, 2, len(st.Attributes))
	})

	t.Run("GetSingleTypeByName - not found", func(t *testing.T) {
		st, err := storage.GetSingleTypeByName("does-not-exist")
		assert.Nil(t, st)
		assert.Error(t, err, "Should return an error for non-existent single type")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("UpdateSingleType - success", func(t *testing.T) {
		// Prepare updated data
		update := &models.SingleType{
			Description: "Updated homepage description",
			Attributes: []models.Attribute{
				{
					Name: "title", // existing attribute -> will be updated
					Type: "string",
				},
				{
					Name: "subtitle", // new attribute -> will be inserted
					Type: "string",
				},
			},
		}

		err := storage.SaveOrUpdateSingleType(update)
		assert.NoError(t, err, "Should update single type without errors")

		// Validate changes
		st, err := storage.GetSingleTypeByName("homepage")
		assert.NoError(t, err, "Should fetch updated single type")
		assert.Equal(t, "Updated homepage description", st.Description)
		assert.Len(t, st.Attributes, 2, "Should have 2 attributes after update")

		var hasTitle, hasSubtitle bool
		for _, attr := range st.Attributes {
			if attr.Name == "title" {
				hasTitle = true
			}
			if attr.Name == "subtitle" {
				hasSubtitle = true
			}
		}
		assert.True(t, hasTitle, "Should still have updated 'title'")
		assert.True(t, hasSubtitle, "Should have inserted new 'subtitle' attribute")
	})

	t.Run("UpdateSingleType - single type not found", func(t *testing.T) {
		update := &models.SingleType{
			Description: "Should not matter",
		}
		err := storage.SaveOrUpdateSingleType(update)
		assert.Error(t, err, "Should fail for non-existent single type name")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteSingleType - success", func(t *testing.T) {
		// Fetch existing single type
		st, err := storage.GetSingleTypeByName("homepage")
		require.NoError(t, err, "Should retrieve single type to delete")

		err = storage.DeleteSingleType(st.ID)
		assert.NoError(t, err, "Should delete single type without errors")

		// Verify it no longer exists
		check, err := storage.GetSingleTypeByName("homepage")
		assert.Nil(t, check, "Should not retrieve deleted single type")
		assert.Error(t, err, "Should report an error for deleted single type")
		assert.Contains(t, err.Error(), "not found")
	})
}
