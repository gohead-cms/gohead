// single_type_test.go
package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// setupTestDB initializes an in-memory SQLite database for testing Singleton storage.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to in-memory database")
	// ":memory:" is per-connection: pin the pool to a single connection.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	// Assign test DB to the global package variable
	database.DB = db

	// Automigrate the necessary models
	err = db.AutoMigrate(&models.Singleton{}, &models.Attribute{}, &models.SingleItem{})
	require.NoError(t, err, "Failed to migrate database")

	return db
}

func TestSingletonStorage(t *testing.T) {
	logger.InitLogger("silent") // Mute logs or set desired level

	db := setupTestDB(t)
	assert.NotNil(t, db, "Database instance should not be nil")

	t.Run("SaveSingleton - create new single type", func(t *testing.T) {
		st := &models.Singleton{
			Name:        "homepage",
			Description: "Main homepage settings",
			Attributes: []models.Attribute{
				{
					Name:        "title",
					Type:        "string",
					Required:    true,
					SingletonID: nil,
				},
				{
					Name:        "heroText",
					Type:        "richtext",
					SingletonID: nil,
				},
			},
		}

		err := storage.SaveOrUpdateSingleton(st)
		assert.NoError(t, err, "Should create single type without errors")
		assert.NotZero(t, st.ID, "Newly created single type should have an ID")

		// Check DB
		var fetched models.Singleton
		err = db.Preload("Attributes").First(&fetched, "name = ?", "homepage").Error
		assert.NoError(t, err, "Should fetch newly created single type from DB")
		assert.Equal(t, "homepage", fetched.Name)
		assert.Len(t, fetched.Attributes, 2)
	})

	t.Run("SaveSingleton - existing name updates in place", func(t *testing.T) {
		// SaveOrUpdateSingleton is an upsert: saving under an existing name
		// updates that singleton instead of failing.
		before, err := storage.GetSingletonByName("homepage")
		require.NoError(t, err)

		st := &models.Singleton{
			Name:        "homepage",
			Description: "Updated description",
			Attributes:  before.Attributes,
		}
		require.NoError(t, storage.SaveOrUpdateSingleton(st))

		after, err := storage.GetSingletonByName("homepage")
		require.NoError(t, err)
		assert.Equal(t, before.ID, after.ID, "upsert must keep the same record")
		assert.Equal(t, "Updated description", after.Description)
		assert.Len(t, after.Attributes, 2)
	})

	t.Run("GetSingletonByName - success", func(t *testing.T) {
		st, err := storage.GetSingletonByName("homepage")
		assert.NoError(t, err, "Should retrieve single type by name")
		assert.Equal(t, "homepage", st.Name)
		assert.Equal(t, 2, len(st.Attributes))
	})

	t.Run("GetSingletonByName - not found", func(t *testing.T) {
		st, err := storage.GetSingletonByName("does-not-exist")
		assert.Nil(t, st)
		assert.Error(t, err, "Should return an error for non-existent single type")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("UpdateSingleton - success", func(t *testing.T) {
		// Prepare updated data
		update := &models.Singleton{
			Name:        "homepage",
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

		err := storage.SaveOrUpdateSingleton(update)
		assert.NoError(t, err, "Should update single type without errors")

		// Validate changes
		st, err := storage.GetSingletonByName("homepage")
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

	t.Run("SaveSingleton - missing name", func(t *testing.T) {
		update := &models.Singleton{
			Description: "Should not matter",
		}
		err := storage.SaveOrUpdateSingleton(update)
		require.Error(t, err, "Should refuse a single type without a name")
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("DeleteSingleton - success", func(t *testing.T) {
		// Fetch existing single type
		st, err := storage.GetSingletonByName("homepage")
		require.NoError(t, err, "Should retrieve single type to delete")

		err = storage.DeleteSingleton(st.ID)
		assert.NoError(t, err, "Should delete single type without errors")

		// Verify it no longer exists
		check, err := storage.GetSingletonByName("homepage")
		assert.Nil(t, check, "Should not retrieve deleted single type")
		assert.Error(t, err, "Should report an error for deleted single type")
		assert.Contains(t, err.Error(), "not found")
	})
}
