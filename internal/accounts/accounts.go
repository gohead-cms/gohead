// Package accounts creates user accounts and the first administrator.
package accounts

import (
	"errors"
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalid matches validation errors (bad username, email or password)
// with errors.Is.
var ErrInvalid = errors.New("invalid account")

// invalid is a validation error: its message is the reason alone.
type invalid struct{ err error }

func (e *invalid) Error() string        { return e.err.Error() }
func (e *invalid) Unwrap() error        { return e.err }
func (e *invalid) Is(target error) bool { return target == ErrInvalid }

func invalidf(format string, args ...any) error {
	return &invalid{fmt.Errorf(format, args...)}
}

// MinAdminPasswordLength applies to administrators created from the
// configuration or the command line.
const MinAdminPasswordLength = 12

// Build validates a new account and hashes its password. Validation errors
// wrap ErrInvalid.
func Build(username, email, password string, role *models.UserRole) (*models.User, error) {
	user := models.User{
		Username: username,
		Email:    email,
		Password: password, // validated in plain text first, hashed below
		Role:     *role,
		Slug:     utils.GenerateSlug(username),
	}
	if err := models.ValidateUser(user); err != nil {
		return nil, &invalid{err}
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashed)
	return &user, nil
}

// CreateAdmin creates an account with the admin role.
func CreateAdmin(username, email, password string) (*models.User, error) {
	if len(password) < MinAdminPasswordLength {
		return nil, invalidf("an administrator password needs at least %d characters", MinAdminPasswordLength)
	}
	role, err := storage.GetRoleByName("admin")
	if err != nil {
		return nil, fmt.Errorf("admin role missing (are the roles seeded?): %w", err)
	}
	user, err := Build(username, email, password, role)
	if err != nil {
		return nil, err
	}
	if err := storage.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

// BootstrapAdmin creates the configured administrator when the instance has
// no users yet. It does nothing when username is empty or users exist, so it
// is safe to run at every start and from several replicas (a concurrent
// creation fails on the unique username and is ignored).
func BootstrapAdmin(username, email, password string) (created bool, err error) {
	if username == "" {
		return false, nil
	}
	count, err := storage.CountUsers()
	if err != nil {
		return false, fmt.Errorf("could not count users: %w", err)
	}
	if count > 0 {
		return false, nil
	}
	if password == "" || email == "" {
		return false, invalidf("admin.username is set: admin.email and admin.password are required too")
	}
	if _, err := CreateAdmin(username, email, password); err != nil {
		var dup *storage.DuplicateEntryError
		if errors.As(err, &dup) {
			return false, nil // another replica created it first
		}
		return false, err
	}
	logger.Log.WithField("username", username).Info("Created the first administrator from the configuration")
	return true, nil
}

// SetPassword replaces a user's password.
func SetPassword(username, password string) error {
	user, err := storage.GetUserByUsername(username)
	if err != nil {
		return err
	}
	min := models.MinPasswordLength
	if user.Role.Name == "admin" {
		min = MinAdminPasswordLength
	}
	if len(password) < min {
		return invalidf("the password needs at least %d characters", min)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return storage.UpdateUser(user.ID, map[string]any{"password": string(hashed)})
}
