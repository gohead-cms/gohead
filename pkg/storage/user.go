package storage

import (
	"fmt"
	"strings"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

func SaveUser(user *models.User) error {
	err := database.DB.Create(user).Error
	if err != nil {
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
			logger.Log.WithError(err).Warn("Duplicate entry for user email")
			return &DuplicateEntryError{Field: "email"}
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.username") {
			logger.Log.WithError(err).Warn("Duplicate entry for user username")
			return &DuplicateEntryError{Field: "username"}
		}

		// General database error
		logger.Log.WithError(err).Error("Failed to save user")
		return &GeneralDatabaseError{Message: err.Error()}
	}

	logger.Log.WithField("username", user.Username).Info("User saved successfully")
	return nil
}

// GetUserByID retrieves a user by their ID.
func GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by their username.
func GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := database.DB.Preload("Role").Where("username = ?", username).First(&user).Error; err != nil {
		if err.Error() == "record not found" {
			logger.Log.WithField("username", username).Warn("User not found")
			return nil, fmt.Errorf("user not found")
		}
		logger.Log.WithField("username", username).Error("Failed to fetch user: ", err)
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	return &user, nil
}

// GetAllUsers retrieves all users from the database.
func GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := database.DB.Select("id, username, role").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve users: %w", err)
	}
	return users, nil
}

// UpdateUser updates the details of an existing user by ID.
func UpdateUser(id uint, updates map[string]interface{}) error {
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser removes a user from the database by ID.
func DeleteUser(id uint) error {
	if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
