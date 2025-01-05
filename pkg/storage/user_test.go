package storage_test

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"gitlab.com/sudo.bngz/gohead/pkg/testutils"
	"gitlab.com/sudo.bngz/gohead/pkg/utils"
)

// Initialize logger for testing
func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestCreateUser(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	err := db.AutoMigrate(
		&models.UserRole{},
	)

	// Create a user role with permissions
	role := models.UserRole{
		Name:        "admin",
		Description: "Administrator",
		Permissions: models.JSONMap{
			"manage_users":   true,
			"manage_content": true,
		},
	}
	assert.NoError(t, db.Create(&role).Error)

	err = db.AutoMigrate(
		&models.User{},
	)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role: models.UserRole{
			Name:        "admin",
			Description: "Administrator",
			Permissions: models.JSONMap{
				"manage_users":   true,
				"manage_content": true,
			},
		},
	}

	err = storage.CreateUser(user)
	assert.NoError(t, err, "Failed to save user")

	// Test duplicate username
	duplicateUser := &models.User{
		Username: "testuser",
		Email:    "duplicate@example.com",
		Password: "hashedpassword",
		Role: models.UserRole{
			Name:        "admin",
			Description: "Administrator",
			Permissions: models.JSONMap{
				"manage_users":   true,
				"manage_content": true,
			},
		},
	}
	err = storage.CreateUser(duplicateUser)
	assert.Error(t, err, "Expected error for duplicate username")
	assert.IsType(t, &storage.DuplicateEntryError{}, err)

	// Test duplicate email
	duplicateUserEmail := &models.User{
		Username: "uniqueuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role: models.UserRole{
			Name:        "admin",
			Description: "Administrator",
			Permissions: models.JSONMap{
				"manage_users":   true,
				"manage_content": true,
			},
		},
	}
	err = storage.CreateUser(duplicateUserEmail)
	assert.Error(t, err, "Expected error for duplicate email")
	assert.IsType(t, &storage.DuplicateEntryError{}, err)
}

func TestGetUserByID(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	err := db.AutoMigrate(
		&models.UserRole{},
		&models.User{},
	)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role: models.UserRole{
			Name:        "admin",
			Description: "Administrator",
			Permissions: models.JSONMap{"manage_users": true, "manage_content": true},
		},
	}
	err = storage.CreateUser(user)
	assert.NoError(t, err, "Failed to save user")

	retrievedUser, err := storage.GetUserByID(user.ID)
	assert.NoError(t, err, "Failed to retrieve user by ID")
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, user.Email, retrievedUser.Email)
}

func TestGetUserByUsername(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	err := db.AutoMigrate(
		&models.UserRole{},
		&models.User{},
	)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role: models.UserRole{
			Name:        "admin",
			Description: "Administrator",
			Permissions: models.JSONMap{"manage_users": true, "manage_content": true},
		},
	}
	err = storage.CreateUser(user)
	assert.NoError(t, err, "Failed to save user")

	retrievedUser, err := storage.GetUserByUsername(user.Username)
	assert.NoError(t, err, "Failed to retrieve user by username")
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, user.Email, retrievedUser.Email)
}

func TestGetAllUsers(t *testing.T) {

	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	err := db.AutoMigrate(
		&models.UserRole{},
		&models.User{},
	)

	user1 := &models.User{
		Username: "user1",
		Email:    "user1@example.com",
		Password: "hashedpassword",
		Slug:     utils.GenerateSlug("user1"),
		Role: models.UserRole{
			Name:        "editor",
			Description: "Content editor",
			Permissions: models.JSONMap{"manage_content": true, "edit_posts": true},
		},
	}
	user2 := &models.User{
		Username: "user2",
		Email:    "user2@example.com",
		Password: "hashedpassword",
		Slug:     utils.GenerateSlug("user2"),
		Role: models.UserRole{
			Name:        "viewer",
			Description: "Content viewer",
			Permissions: models.JSONMap{"read_content": true},
		},
	}
	err = storage.CreateUser(user1)
	assert.NoError(t, err, "Failed to save user1")
	err = storage.CreateUser(user2)
	assert.NoError(t, err, "Failed to save user2")

	users, err := storage.GetAllUsers()
	assert.NoError(t, err, "Failed to retrieve all users")
	assert.Len(t, users, 2)
}

func TestUpdateUser(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	err := db.AutoMigrate(
		&models.UserRole{},
		&models.User{},
	)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role: models.UserRole{
			Name:        "admin",
			Description: "Administrator",
			Permissions: models.JSONMap{
				"manage_users":   true,
				"manage_content": true,
			},
		},
	}
	err = storage.CreateUser(user)
	assert.NoError(t, err, "Failed to save user")

	updates := map[string]interface{}{
		"username": "updateduser",
		"email":    "updated@example.com",
	}
	err = storage.UpdateUser(user.ID, updates)
	assert.NoError(t, err, "Failed to update user")

	updatedUser, err := storage.GetUserByID(user.ID)
	assert.NoError(t, err, "Failed to retrieve updated user")
	assert.Equal(t, "updateduser", updatedUser.Username)
	assert.Equal(t, "updated@example.com", updatedUser.Email)
}

func TestDeleteUser(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Apply migrations
	err := db.AutoMigrate(
		&models.UserRole{},
		&models.User{},
	)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role: models.UserRole{
			Name:        "admin",
			Description: "Administrator",
			Permissions: models.JSONMap{
				"manage_users":   true,
				"manage_content": true,
			},
		},
	}
	err = storage.CreateUser(user)
	assert.NoError(t, err, "Failed to save user")

	err = storage.DeleteUser(user.ID)
	assert.NoError(t, err, "Failed to delete user")

	deletedUser, err := storage.GetUserByID(user.ID)
	assert.Error(t, err, "Expected error for deleted user")
	assert.Nil(t, deletedUser)
}
