package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gohead/pkg/logger"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// UserRole defines the role of a user
type UserRole struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`                                                // Role name (e.g., admin, editor, viewer)
	Description string  `json:"description"`                                         // Role description
	Permissions JSONMap `json:"permissions" gorm:"type:jsonb;default:'[]';not null"` // Use 'jsonb' for PostgreSQL // Permissions associated with the role
}

// User represents a user in the system with extended profile information.
type User struct {
	gorm.Model
	Username        string    `json:"username" gorm:"uniqueIndex;size:191"` // Unique username with length limit for MySQL compatibility
	Email           string    `json:"email" gorm:"uniqueIndex;size:191"`    // Unique email with length limit for MySQL compatibility
	Password        string    `json:"password"`                             // Hashed password
	UserRoleID      int       `json:"-"`                                    // Foreign key reference (not exposed in JSON)
	Role            UserRole  `json:"role" gorm:"foreignKey:UserRoleID"`    // Associated role
	Slug            string    `json:"slug" gorm:"uniqueIndex"`              // Unique slug for the user
	ProfileImage    string    `json:"profile_image"`                        // URL to the user's profile image
	CoverImage      *string   `json:"cover_image"`                          // URL to the user's cover image (optional)
	Bio             string    `json:"bio"`                                  // Short biography
	Website         string    `json:"website"`                              // Personal or professional website
	Location        string    `json:"location"`                             // User's location
	Facebook        string    `json:"facebook"`                             // Facebook username or handle
	Twitter         string    `json:"twitter"`                              // Twitter handle
	MetaTitle       *string   `json:"meta_title"`                           // SEO meta title (optional)
	MetaDescription *string   `json:"meta_description"`                     // SEO meta description (optional)
	URL             string    `json:"url"`                                  // Full URL to the user's profile
	CreatedAt       time.Time `json:"created_at,omitempty"`                 // Auto-managed timestamp
}

func ValidateUser(user User) error {
	// Check if username is provided and valid
	if strings.TrimSpace(user.Username) == "" {
		logger.Log.WithFields(logrus.Fields{
			"field": "username",
		}).Warn("Validation failed: missing username")
		return errors.New("username is required")
	}

	// Validate email format
	if !isValidEmail(user.Email) {
		logger.Log.WithFields(logrus.Fields{
			"field": "email",
			"value": user.Email,
		}).Warn("Validation failed: invalid email address")
		return errors.New("invalid email address")
	}

	// Validate password length
	minPasswordLength := 6
	if len(user.Password) < minPasswordLength {
		logger.Log.WithFields(logrus.Fields{
			"field":   "password",
			"length":  len(user.Password),
			"minimum": minPasswordLength,
		}).Warn("Validation failed: password too short")
		return fmt.Errorf("password must be at least %d characters long", minPasswordLength)
	}

	// Check if slug is unique and provided
	if strings.TrimSpace(user.Slug) == "" {
		logger.Log.WithFields(logrus.Fields{
			"field": "slug",
		}).Warn("Validation failed: missing slug")
		return errors.New("slug is required")
	}

	// Validate profile image (if provided)
	if user.ProfileImage != "" && !isValidURL(user.ProfileImage) {
		logger.Log.WithFields(logrus.Fields{
			"field": "profile_image",
			"value": user.ProfileImage,
		}).Warn("Validation failed: invalid profile image URL")
		return errors.New("invalid profile image URL")
	}

	// Validate website (if provided)
	if user.Website != "" && !isValidURL(user.Website) {
		logger.Log.WithFields(logrus.Fields{
			"field": "website",
			"value": user.Website,
		}).Warn("Validation failed: invalid website URL")
		return errors.New("invalid website URL")
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
	}).Info("User validation passed")
	return nil
}

// isValidEmail checks if the given email is valid
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// isValidURL checks if the given URL is valid
func isValidURL(url string) bool {
	re := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return re.MatchString(url)
}

// ValidateUserRole validates the user role data
func ValidateUserRole(role UserRole) error {
	// Check if role name is provided
	if role.Name == "" {
		logger.Log.WithFields(logrus.Fields{
			"role": role.Name,
		}).Warn("Validation failed: role name missing")
		return fmt.Errorf("role name is required")
	}

	// Validate permissions
	if len(role.Permissions) == 0 {
		logger.Log.WithFields(logrus.Fields{
			"role": role.Name,
		}).Warn("Validation failed: no permissions assigned")
		return fmt.Errorf("at least one permission is required for the role")
	}

	logger.Log.WithFields(logrus.Fields{
		"role": role.Name,
	}).Info("Role validation passed")
	return nil
}

// ValidateUserUpdates validates updates to a user's fields
func ValidateUserUpdates(updates map[string]interface{}) error {
	for key, value := range updates {
		switch key {
		case "username":
			if username, ok := value.(string); !ok || username == "" {
				logger.Log.WithFields(logrus.Fields{
					"field": "username",
					"value": value,
				}).Warn("Validation failed: invalid username")
				return fmt.Errorf("invalid username")
			}
		case "email":
			if email, ok := value.(string); !ok || !isValidEmail(email) {
				logger.Log.WithFields(logrus.Fields{
					"field": "email",
					"value": value,
				}).Warn("Validation failed: invalid email")
				return fmt.Errorf("invalid email address")
			}
		case "password":
			if password, ok := value.(string); !ok || len(password) < 6 {
				logger.Log.WithFields(logrus.Fields{
					"field": "password",
				}).Warn("Validation failed: password too short")
				return fmt.Errorf("password must be at least 6 characters long")
			}
		case "role":
			if role, ok := value.(UserRole); !ok {
				logger.Log.WithFields(logrus.Fields{
					"field": "role",
					"value": value,
				}).Warn("Validation failed: invalid role")
				return fmt.Errorf("invalid role format")
			} else {
				// Validate the role object itself
				if err := ValidateUserRole(role); err != nil {
					return err
				}
			}
		default:
			logger.Log.WithFields(logrus.Fields{
				"field": key,
				"value": value,
			}).Warn("Validation failed: unsupported field")
			return fmt.Errorf("unsupported field for update: %s", key)
		}
	}

	logger.Log.Info("User updates validation passed")
	return nil
}
