package storage_test

import (
	"bytes"
	"errors"
	"testing"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

func TestGetRoleByName(t *testing.T) {
	// Set up test database
	_, db := testutils.SetupTestServer()
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

func TestRoleCRUD(t *testing.T) {

	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	assert.NoError(t, db.AutoMigrate(&models.UserRole{}), "Failed to apply migrations")
	// Test CreateRole
	role := &models.UserRole{
		Name:        "admin",
		Description: "Administrator role",
		Permissions: models.JSONMap{"manage_users": true, "manage_content": true},
	}
	err := storage.SaveRole(role)
	assert.NoError(t, err, "CreateRole should not return an error")
	assert.NotZero(t, role.ID, "Role ID should be set after creation")

	// Test GetRoleByID
	fetchedRole, err := storage.GetRoleByID(role.ID)
	assert.NoError(t, err, "GetRoleByID should not return an error")
	assert.Equal(t, role.Name, fetchedRole.Name, "Fetched role name should match created role")

	// Test GetRoleByName
	fetchedRoleByName, err := storage.GetRoleByName("admin")
	assert.NoError(t, err, "GetRoleByName should not return an error")
	assert.Equal(t, role.ID, fetchedRoleByName.ID, "Fetched role by name should match created role ID")

	// Test GetAllRoles
	roles, err := storage.GetAllRoles()
	assert.NoError(t, err, "GetAllRoles should not return an error")
	assert.Len(t, roles, 1, "There should be one role in the database")

	// Test UpdateRole
	updates := map[string]interface{}{
		"description": "Updated Administrator role",
	}
	err = storage.UpdateRole(role.ID, updates)
	assert.NoError(t, err, "UpdateRole should not return an error")

	updatedRole, err := storage.GetRoleByID(role.ID)
	assert.NoError(t, err, "GetRoleByID should not return an error")
	assert.Equal(t, "Updated Administrator role", updatedRole.Description, "Role description should be updated")

	// Test DeleteRole
	err = storage.DeleteRole(role.ID)
	assert.NoError(t, err, "DeleteRole should not return an error")

	deletedRole, err := storage.GetRoleByID(role.ID)
	assert.Error(t, err, "GetRoleByID should return an error for deleted role")
	assert.Nil(t, deletedRole, "Deleted role should not be retrievable")
}
