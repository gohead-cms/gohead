// internal/models/user_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserModel(t *testing.T) {
	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&User{})
	assert.NoError(t, err)

	// Create a User
	user := User{
		Username: "testuser",
		Password: "securepassword",
		Role:     "admin",
	}
	err = db.Create(&user).Error
	assert.NoError(t, err)
	assert.NotZero(t, user.ID, "User ID should be set after creation")

	// Retrieve the User
	var retrievedUser User
	err = db.First(&retrievedUser, user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "testuser", retrievedUser.Username)
	assert.Equal(t, "securepassword", retrievedUser.Password)
	assert.Equal(t, "admin", retrievedUser.Role)

	// Test Unique Constraint
	duplicateUser := User{
		Username: "testuser",
		Password: "anotherpassword",
		Role:     "viewer",
	}
	err = db.Create(&duplicateUser).Error
	assert.Error(t, err, "Should fail to create a user with a duplicate username")

	// Update the User
	err = db.Model(&retrievedUser).Update("Role", "editor").Error
	assert.NoError(t, err)

	// Verify the update
	var updatedUser User
	err = db.First(&updatedUser, user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "editor", updatedUser.Role)

	// Delete the User
	err = db.Delete(&User{}, user.ID).Error
	assert.NoError(t, err)

	// Verify deletion
	var deletedUser User
	err = db.First(&deletedUser, user.ID).Error
	assert.Error(t, err, "Record should not be found after deletion")
}
