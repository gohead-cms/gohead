package models_test

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

// Initialize logger for testing
func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
	// Initialize in-memory test database

}

func TestValidateUser(t *testing.T) {

	validUser := models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "securepassword",
		RoleRefer: 1,
	}

	err := models.ValidateUser(validUser)
	assert.NoError(t, err)

	invalidUser := models.User{
		Username:  "",
		Email:     "invalid-email",
		Password:  "123",
		RoleRefer: 0,
	}

	err = models.ValidateUser(invalidUser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")

	invalidUser.Username = "testuser"
	err = models.ValidateUser(invalidUser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email address")

	invalidUser.Email = "test@example.com"
	err = models.ValidateUser(invalidUser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role is required")

	invalidUser.Password = "securepassword"
	err = models.ValidateUser(invalidUser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role is required")
}

func TestValidateUserRole(t *testing.T) {
	validRole := models.UserRole{
		Name:        "admin",
		Permissions: models.JSONMap{"manage_users": true},
	}

	err := models.ValidateUserRole(validRole)
	assert.NoError(t, err)

	invalidRole := models.UserRole{
		Name:        "",
		Permissions: models.JSONMap{},
	}

	err = models.ValidateUserRole(invalidRole)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role name is required")

	invalidRole.Name = "admin"
	err = models.ValidateUserRole(invalidRole)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one permission is required for the role")
}

func TestValidateUserUpdates(t *testing.T) {
	validUpdates := map[string]interface{}{
		"username": "newuser",
		"email":    "new@example.com",
		"password": "newpassword",
	}

	err := models.ValidateUserUpdates(validUpdates)
	assert.NoError(t, err)

	invalidUpdates := map[string]interface{}{
		"username": "test",
		"email":    "invalid-email",
		"password": "123456g",
	}

	err = models.ValidateUserUpdates(invalidUpdates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email address")

	invalidUpdates["email"] = "new@example.com"
	invalidUpdates["username"] = ""
	err = models.ValidateUserUpdates(invalidUpdates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid username")

	err = models.ValidateUserUpdates(invalidUpdates)
	assert.Error(t, err)

	validRole := models.UserRole{Name: "editor", Permissions: models.JSONMap{"edit_content": true}}
	validUpdates["role"] = validRole
	err = models.ValidateUserUpdates(validUpdates)
	assert.NoError(t, err)

	invalidRole := models.UserRole{Name: "", Permissions: models.JSONMap{}}
	validUpdates["role"] = invalidRole
	err = models.ValidateUserUpdates(validUpdates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role name is required")
}
