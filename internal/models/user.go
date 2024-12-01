package models

import (
	"fmt"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/gorm"
)

// UserRole defines the role of a user
type UserRole struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`                         // Role name (e.g., admin, editor, viewer)
	Description string  `json:"description"`                  // Role description
	Permissions JSONMap `json:"permissions" gorm:"type:json"` // Permissions associated with the role
}
type User struct {
	gorm.Model
	Username  string    `json:"username" gorm:"uniqueIndex;size:191"` // Unique username with length limit for MySQL compatibility
	Email     string    `json:"email" gorm:"uniqueIndex;size:191"`    // Unique email with length limit for MySQL compatibility
	Password  string    `json:"password"`                             // Hashed password
	RoleRefer int       `json:"-"`                                    // Foreign key reference (not exposed in JSON)
	Role      UserRole  `json:"role" gorm:"foreignKey:RoleRefer"`     // Associated role
	CreatedAt time.Time `json:"created_at,omitempty"`                 // Auto-managed timestamp
}

// ValidateUser validates the user data
func ValidateUser(user User) error {
	// Check if username is provided
	if user.Username == "" {
		logger.Log.WithFields(logrus.Fields{
			"username": user.Username,
		}).Warn("Validation failed: missing username")
		return fmt.Errorf("username is required")
	}

	// Validate email
	if !isValidEmail(user.Email) {
		logger.Log.WithFields(logrus.Fields{
			"email": user.Email,
		}).Warn("Validation failed: invalid email address")
		return fmt.Errorf("invalid email address")
	}

	// Check if role is provided
	if user.RoleRefer == 0 {
		logger.Log.WithFields(logrus.Fields{
			"username": user.Username,
			"role_id":  user.Role.ID,
		}).Warn("Validation failed: invalid role")
		return fmt.Errorf("role is required")
	}

	// Additional password checks (e.g., length)
	if len(user.Password) < 6 {
		logger.Log.WithFields(logrus.Fields{
			"username": user.Username,
		}).Warn("Validation failed: password too short")
		return fmt.Errorf("password must be at least 6 characters long")
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     user.Role.Name,
	}).Info("User validation passed")
	return nil
}

// isValidEmail checks if the given email address is valid
func isValidEmail(email string) bool {
	regex := `^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`
	re := regexp.MustCompile(regex)
	if !re.MatchString(email) {
		logger.Log.WithFields(logrus.Fields{
			"email": email,
		}).Warn("Validation failed: email does not match regex")
		return false
	}
	return true
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
