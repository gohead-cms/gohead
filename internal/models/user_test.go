package models_test

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestValidateUser(t *testing.T) {
	t.Run("Valid User", func(t *testing.T) {
		user := models.User{
			Username:     "testuser",
			Email:        "test@example.com",
			Password:     "securepassword",
			Slug:         "testuser",
			ProfileImage: "https://example.com/profile.jpg",
			Website:      "https://example.com",
		}
		err := models.ValidateUser(user)
		assert.NoError(t, err)
	})

	t.Run("Missing Username", func(t *testing.T) {
		user := models.User{
			Email:    "test@example.com",
			Password: "securepassword",
			Slug:     "testuser",
		}
		err := models.ValidateUser(user)
		assert.EqualError(t, err, "username is required")
	})

	t.Run("Invalid Email", func(t *testing.T) {
		user := models.User{
			Username: "testuser",
			Email:    "invalid-email",
			Password: "securepassword",
			Slug:     "testuser",
		}
		err := models.ValidateUser(user)
		assert.EqualError(t, err, "invalid email address")
	})

	t.Run("Short Password", func(t *testing.T) {
		user := models.User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "123",
			Slug:     "testuser",
		}
		err := models.ValidateUser(user)
		assert.EqualError(t, err, "password must be at least 6 characters long")
	})

	t.Run("Invalid Profile Image URL", func(t *testing.T) {
		user := models.User{
			Username:     "testuser",
			Email:        "test@example.com",
			Password:     "securepassword",
			Slug:         "testuser",
			ProfileImage: "not-a-valid-url",
		}
		err := models.ValidateUser(user)
		assert.EqualError(t, err, "invalid profile image URL")
	})

	t.Run("Invalid Website URL", func(t *testing.T) {
		user := models.User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "securepassword",
			Slug:     "testuser",
			Website:  "invalid-website",
		}
		err := models.ValidateUser(user)
		assert.EqualError(t, err, "invalid website URL")
	})
}

func TestValidateUserRole(t *testing.T) {
	t.Run("Valid Role", func(t *testing.T) {
		role := models.UserRole{
			Name:        "admin",
			Description: "Administrator role",
			Permissions: models.JSONMap{"manage_users": true, "manage_content": true},
		}
		err := models.ValidateUserRole(role)
		assert.NoError(t, err)
	})

	t.Run("Missing Role Name", func(t *testing.T) {
		role := models.UserRole{
			Description: "Role with no name",
			Permissions: models.JSONMap{"manage_users": true},
		}
		err := models.ValidateUserRole(role)
		assert.EqualError(t, err, "role name is required")
	})

	t.Run("No Permissions", func(t *testing.T) {
		role := models.UserRole{
			Name:        "editor",
			Description: "Editor role",
		}
		err := models.ValidateUserRole(role)
		assert.EqualError(t, err, "at least one permission is required for the role")
	})
}

func TestValidateUserUpdates(t *testing.T) {
	t.Run("Valid Updates", func(t *testing.T) {
		updates := map[string]interface{}{
			"username": "newusername",
			"email":    "newemail@example.com",
			"password": "newpassword",
			"role": models.UserRole{
				Name:        "editor",
				Description: "Editor role",
				Permissions: models.JSONMap{"edit_articles": true},
			},
		}
		err := models.ValidateUserUpdates(updates)
		assert.NoError(t, err)
	})

	t.Run("Invalid Username", func(t *testing.T) {
		updates := map[string]interface{}{
			"username": "",
		}
		err := models.ValidateUserUpdates(updates)
		assert.EqualError(t, err, "invalid username")
	})

	t.Run("Invalid Email", func(t *testing.T) {
		updates := map[string]interface{}{
			"email": "invalid-email",
		}
		err := models.ValidateUserUpdates(updates)
		assert.EqualError(t, err, "invalid email address")
	})

	t.Run("Short Password", func(t *testing.T) {
		updates := map[string]interface{}{
			"password": "123",
		}
		err := models.ValidateUserUpdates(updates)
		assert.EqualError(t, err, "password must be at least 6 characters long")
	})

	t.Run("Invalid Role Format", func(t *testing.T) {
		updates := map[string]interface{}{
			"role": "invalid-role-format",
		}
		err := models.ValidateUserUpdates(updates)
		assert.EqualError(t, err, "invalid role format")
	})

	t.Run("Unsupported Field", func(t *testing.T) {
		updates := map[string]interface{}{
			"unsupported_field": "some value",
		}
		err := models.ValidateUserUpdates(updates)
		assert.EqualError(t, err, "unsupported field for update: unsupported_field")
	})
}
