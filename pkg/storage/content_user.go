package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

// SaveUser creates a new user in the database.
func SaveUser(user *models.User) error {
	if err := database.DB.Create(user).Error; err != nil {
		logger.Log.Warn("Failed to save user", err)
	}
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
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
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
