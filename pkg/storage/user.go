package storage

import (
	"errors"
	"fmt"

	"gohead/internal/models"
	"gohead/pkg/database"
	"gohead/pkg/logger"

	"gorm.io/gorm"
)

func CreateUser(user *models.User) error {
	result := database.DB.Create(user)
	if result.Error != nil {
		if result.Error.Error() == "UNIQUE constraint failed: users.username" {
			return &DuplicateEntryError{Field: "username"}
		} else if result.Error.Error() == "UNIQUE constraint failed: users.email" {
			return &DuplicateEntryError{Field: "email"}
		}
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			logger.Log.WithFields(map[string]interface{}{
				"username": user.Username,
				"email":    user.Email,
			}).Warn("Duplicate key error when saving user")

		}

		logger.Log.WithError(result.Error).Error("Failed to save user due to database error")
		return &GeneralDatabaseError{Message: result.Error.Error()}
	}

	// Log success
	logger.Log.WithField("username", user.Username).Info("User created successfully in database")
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
	if err := database.DB.Select("id, username").Find(&users).Error; err != nil {
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
