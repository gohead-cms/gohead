package storage_test

import (
	"errors"
	"testing"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGetRoleByName(t *testing.T) {
	// Set up test database
	db := testutils.SetupTestDB()
	defer testutils.CleanupTestDB()

	// Apply migrations
	assert.NoError(t, db.AutoMigrate(&models.UserRole{}), "Failed to apply migrations")

	// Seed test roles
	testRoles := []models.UserRole{
		{Name: "admin", Description: "Administrator with full access", Permissions: models.JSONMap{"manage_users": true, "manage_content": true}},
		{Name: "editor", Description: "Editor with content management access", Permissions: models.JSONMap{"manage_content": true}},
		{Name: "viewer", Description: "Viewer with read-only access", Permissions: models.JSONMap{"read_content": true}},
	}

	for _, role := range testRoles {
		err := db.Create(&role).Error
		assert.NoError(t, err, "Failed to seed role: %s", role.Name)
	}

	t.Run("Role exists", func(t *testing.T) {
		// Retrieve an existing role
		role, err := storage.GetRoleByName("admin")
		assert.NoError(t, err, "Expected no error when retrieving an existing role")
		assert.NotNil(t, role, "Expected role to be non-nil")
		assert.Equal(t, "admin", role.Name, "Expected role name to match")
		assert.Equal(t, "Administrator with full access", role.Description, "Expected role description to match")
		assert.Equal(t, true, role.Permissions["manage_users"], "Expected permissions to include 'manage_users'")
	})

	t.Run("Role does not exist", func(t *testing.T) {
		// Attempt to retrieve a non-existent role
		role, err := storage.GetRoleByName("nonexistent")
		assert.Nil(t, role, "Expected role to be nil when not found")
		assert.Error(t, err, "Expected error when role does not exist")
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound), "Expected error to wrap gorm.ErrRecordNotFound")
		assert.Contains(t, err.Error(), "role 'nonexistent' not found", "Expected error message to contain role name")
	})
}
