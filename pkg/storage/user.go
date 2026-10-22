package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

func CreateUser(user *models.User) error {
	result := database.DB.Create(user)
	if result.Error != nil {
		if field, ok := duplicateField(result.Error); ok {
			logger.Log.WithFields(map[string]interface{}{
				"username": user.Username,
				"email":    user.Email,
				"field":    field,
			}).Warn("Duplicate key error when saving user")
			return &DuplicateEntryError{Field: field}
		}
		logger.Log.WithError(result.Error).Error("Failed to save user due to database error")
		return &GeneralDatabaseError{Message: result.Error.Error()}
	}

	// Log success
	logger.Log.WithField("username", user.Username).Info("User created successfully in database")
	return nil
}

// duplicateField detects unique-constraint violations across SQLite, Postgres
// and MySQL and returns the offending field when it can be identified.
func duplicateField(err error) (string, bool) {
	msg := strings.ToLower(err.Error())
	isDup := errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(msg, "unique constraint failed") || // sqlite
		strings.Contains(msg, "duplicate key value") || // postgres
		strings.Contains(msg, "duplicate entry") // mysql
	if !isDup {
		return "", false
	}
	for _, f := range []string{"username", "email", "slug"} {
		if strings.Contains(msg, f) {
			return f, true
		}
	}
	return "unknown", true
}

// GetUserByID retrieves a user by their ID.
func GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := database.DB.Preload("Role").First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// CountUsers returns the total number of (non-deleted) users.
func CountUsers() (int64, error) {
	var count int64
	if err := database.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
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
	if err := database.DB.Preload("Role").Find(&users).Error; err != nil {
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

// CountUsersWithRole counts the users that have the role.
func CountUsersWithRole(roleName string) (int64, error) {
	var n int64
	err := database.DB.Model(&models.User{}).
		Joins("JOIN user_roles ON user_roles.id = users.user_role_id").
		Where("user_roles.name = ?", roleName).Count(&n).Error
	return n, err
}

// ErrUserNotFound is returned by the lookups of sign-in with a provider.
var ErrUserNotFound = errors.New("user not found")

// GetUserBySSO returns the account linked to an identity provider's subject.
func GetUserBySSO(provider, subject string) (*models.User, error) {
	var user models.User
	err := database.DB.Preload("Role").Where(&models.User{SSOProvider: provider, SSOSubject: subject}).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

// GetUserByEmail returns the account with an email address (any case).
func GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := database.DB.Preload("Role").Where("LOWER(email) = ?", strings.ToLower(email)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

// LinkUserSSO links an account to an identity provider's subject.
func LinkUserSSO(id uint, provider, subject string) error {
	return database.DB.Model(&models.User{}).Where("id = ?", id).
		Updates(models.User{SSOProvider: provider, SSOSubject: subject}).Error
}
