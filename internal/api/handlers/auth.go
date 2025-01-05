package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"gitlab.com/sudo.bngz/gohead/pkg/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Register(c *gin.Context) {
	// Parse input
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required"`
		RoleName string `json:"role_name" binding:"required"` // Role provided by the client
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("Register: Invalid input")
		c.Set("status", http.StatusBadRequest)
		c.Set("details", err.Error())
		c.Set("response", "Register: Invalid input")
		return
	}

	logger.Log.WithField("request_body", c.Request.Body).Debug("Request received")

	// Fetch the role
	role, err := storage.GetRoleByName(input.RoleName)
	if err != nil {
		logger.Log.WithError(err).Error("Register: Failed to fetch role")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Set("status", http.StatusNotFound)
			c.Set("response", fmt.Sprintf("Role '%s' does not exist", input.RoleName))
			c.Set("details", err.Error())
			return
		}
		c.Set("status", http.StatusNotFound)
		c.Set("response", fmt.Sprintf("Role '%s' does not exist", input.RoleName))
		c.Set("details", err.Error())
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Log.WithError(err).Error("Register: Failed to hash password")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", "Register: Failed to hash password")
		c.Set("details", err.Error())
		return
	}

	// Generate slug for the user
	slug := utils.GenerateSlug(input.Username)

	// Create user instance
	user := models.User{
		Username:     input.Username,
		Password:     string(hashedPassword),
		Role:         *role,
		Email:        input.Email,
		Slug:         slug,
		ProfileImage: "", // Default profile image can be set if needed
	}

	// Validate user
	if err := models.ValidateUser(user); err != nil {
		logger.Log.WithError(err).Warn("Register: User validation failed")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", "Register: User validation failed")
		c.Set("details", err.Error())
		return
	}

	// Save the user
	err = storage.CreateUser(&user)
	switch e := err.(type) {
	case nil:
		logger.Log.WithFields(logrus.Fields{
			"username": user.Username,
			"role":     role.Name,
		}).Info("User registered successfully")
		c.Set("status", http.StatusCreated)
		c.Set("response", "User registered successfully")
		return
	case *storage.DuplicateEntryError:
		logger.Log.WithError(err).Warn("Register: Duplicate entry")
		c.Set("status", http.StatusBadRequest)
		c.Set("response", "Register: Duplicate entry")
		c.Set("details", e.Error())
		return
	case *storage.GeneralDatabaseError:
		logger.Log.WithError(err).Error("Register: General database error")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", "Register: General database error")
		c.Set("details", e.Error())
		return
	default:
		logger.Log.WithError(err).Error("Register: Unknown error")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", "Register: Unknown error")
		c.Set("details", e.Error())
		return
	}
}

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

	// Fetch the user with their role
	user, err := storage.GetUserByUsername(input.Username)
	if err != nil {
		logger.Log.WithError(err).Warn("Login: Invalid username or password")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", "Invalid username or password")
		return
	}

	// Compare the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		logger.Log.WithError(err).Warn("Login: Invalid username or password")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", "Invalid username or password")
		return
	}

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
		"token":    tokenString,
	}).Info("User logged in successfully")

	c.Set("status", http.StatusOK)
	c.Set("response", gin.H{"token": tokenString})
}
