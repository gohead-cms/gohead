package handlers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gohead-cms/gohead/internal/accounts"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/ratelimit"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Registration modes for POST /auth/register.
const (
	// RegistrationDisabled refuses self-registration: accounts are created
	// by an admin (POST /admin/users), the configuration (admin.*) or the
	// `gohead admin create` command.
	RegistrationDisabled = "disabled"
	// RegistrationOpen lets anyone create an account with the
	// DefaultRegistrationRole.
	RegistrationOpen = "open"
)

// Registration is the current registration mode, set at startup from
// auth.registration.
var Registration = RegistrationDisabled

// Register creates a new account through the public endpoint, when
// registration is open. The caller never chooses its role: accounts get the
// least-privileged DefaultRegistrationRole, and admins promote users through
// PUT /admin/users/:id. No account is ever made admin here.
func Register(c *gin.Context) {
	if Registration != RegistrationOpen {
		c.Set("status", http.StatusForbidden)
		c.Set("response", "Registration is disabled")
		c.Set("details", "ask an administrator for an account")
		return
	}
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("Register: Invalid input")
		c.Set("status", http.StatusBadRequest)
		c.Set("details", err.Error())
		c.Set("response", "Register: Invalid input")
		return
	}

	if len(input.Password) < models.MinPasswordLength {
		c.Set("status", http.StatusBadRequest)
		c.Set("response", "Register: User validation failed")
		c.Set("details", fmt.Sprintf("password must be at least %d characters long", models.MinPasswordLength))
		return
	}

	roleName := DefaultRegistrationRole
	role, err := storage.GetRoleByName(roleName)
	if err != nil {
		// Roles are seeded at startup, so a missing one is a server-side problem.
		logger.Log.WithError(err).Error("Register: Failed to fetch role")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", fmt.Sprintf("Register: role '%s' is not configured", roleName))
		return
	}

	user, status, err := buildUser(input.Username, input.Email, input.Password, role)
	if err != nil {
		c.Set("status", status)
		c.Set("response", "Register: User validation failed")
		c.Set("details", err.Error())
		return
	}

	if status, msg, details := saveNewUser(user); status != 0 {
		c.Set("status", status)
		c.Set("response", "Register: "+msg)
		c.Set("details", details)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     role.Name,
	}).Info("User registered successfully")
	c.Set("status", http.StatusCreated)
	c.Set("response", "User registered successfully")
}

// DefaultRegistrationRole is the role given to self-registered accounts.
const DefaultRegistrationRole = "viewer"

// buildUser hashes the password and assembles a validated user.
func buildUser(username, email, password string, role *models.UserRole) (*models.User, int, error) {
	user, err := accounts.Build(username, email, password, role)
	if errors.Is(err, accounts.ErrInvalid) {
		return nil, http.StatusBadRequest, err
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return user, 0, nil
}

// saveNewUser persists a user and maps storage errors to HTTP status codes.
// A zero status means success.
func saveNewUser(user *models.User) (int, string, string) {
	err := storage.CreateUser(user)
	if err == nil {
		return 0, "", ""
	}
	var dup *storage.DuplicateEntryError
	if errors.As(err, &dup) {
		return http.StatusConflict, "Duplicate entry", dup.Error()
	}
	logger.Log.WithError(err).Error("Failed to save user")
	return http.StatusInternalServerError, "General database error", ""
}

// LoginAttemptGuard limits failed logins per (username, client IP) within a
// window. Keying on both stops password guessing against an account without
// letting anyone lock a user out from another address.
type LoginAttemptGuard struct {
	Counter     ratelimit.Counter
	MaxFailures int
	Window      time.Duration
}

// LoginGuard is set at startup; nil disables the check.
var LoginGuard *LoginAttemptGuard

func loginGuardKey(username, ip string) string {
	return "login-fail:" + strings.ToLower(strings.TrimSpace(username)) + ":" + ip
}

func (g *LoginAttemptGuard) recordFailure(ctx context.Context, key string) {
	if g == nil {
		return
	}
	if _, _, err := g.Counter.Hit(ctx, key, g.Window); err != nil {
		logger.Log.WithError(err).Warn("Login guard: cannot record failure")
	}
}

func (g *LoginAttemptGuard) reset(ctx context.Context, key string) {
	if g == nil {
		return
	}
	_ = g.Counter.Reset(ctx, key)
}

// dummyPasswordHash is compared against when the user does not exist, so
// unknown usernames take as long as wrong passwords.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("gohead-timing-equalizer"), bcrypt.DefaultCost)

func Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("Login: Invalid input")
		c.Set("status", http.StatusBadRequest)
		c.Set("response", "Invalid input")
		return
	}

	// Too many recent failures for this account from this client?
	guardKey := loginGuardKey(input.Username, c.ClientIP())
	if LoginGuard != nil {
		if n, reset, err := LoginGuard.Counter.Count(c.Request.Context(), guardKey); err == nil && n >= int64(LoginGuard.MaxFailures) {
			logger.Log.WithField("username", input.Username).Warn("Login: too many failed attempts")
			c.Header("Retry-After", strconv.Itoa(max(int(math.Ceil(reset.Seconds())), 1)))
			c.Set("status", http.StatusTooManyRequests)
			c.Set("response", "Too many failed login attempts, please retry later")
			return
		}
	}

	// Fetch the user with their role. When the user does not exist, still
	// run a bcrypt comparison so response time does not reveal whether the
	// username exists.
	user, err := storage.GetUserByUsername(input.Username)
	hash := dummyPasswordHash
	if err == nil {
		hash = []byte(user.Password)
	}
	if cmpErr := bcrypt.CompareHashAndPassword(hash, []byte(input.Password)); err != nil || cmpErr != nil {
		LoginGuard.recordFailure(c.Request.Context(), guardKey)
		logger.Log.WithField("username", input.Username).Warn("Login: Invalid username or password")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", "Invalid username or password")
		return
	}
	LoginGuard.reset(c.Request.Context(), guardKey)

	// Validate the Role
	if user.Role.Name == "" {
		logger.Log.WithField("username", user.Username).Warn("Login: User role is not assigned")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", "User role is not assigned")
		return
	}

	// Generate JWT token with the user's role
	tokenString, err := auth.GenerateJWT(user.Username, user.Role.Name)
	if err != nil {
		logger.Log.WithError(err).Error("Login: Failed to generate token")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", "Failed to generate token")
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     user.Role.Name,
	}).Info("User logged in successfully")

	c.Set("status", http.StatusOK)
	c.Set("response", gin.H{"token": tokenString})
}
