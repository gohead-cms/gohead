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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch the role
	role, err := storage.GetRoleByName(input.RoleName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("role_name", input.RoleName).Warn("Register: Role not found")
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Role '%s' does not exist", input.RoleName)})
			return
		}
		logger.Log.WithError(err).Error("Register: Failed to fetch role")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role"})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Log.WithError(err).Error("Register: Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save the user
	err = storage.SaveUser(&user)
	switch e := err.(type) {
	case nil:
		logger.Log.WithFields(logrus.Fields{
			"username": user.Username,
			"role":     role.Name,
		}).Info("User registered successfully")
		c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
		return
	case *storage.DuplicateEntryError:
		logger.Log.WithError(err).Warn("Register: Duplicate entry")
		c.JSON(http.StatusBadRequest, gin.H{"error": e.Error()})
		return
	case *storage.GeneralDatabaseError:
		logger.Log.WithError(err).Error("Register: General database error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	default:
		logger.Log.WithError(err).Error("Register: Unknown error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An unknown error occurred"})
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
		c.Set("response", gin.H{"error": "Invalid input"})
		return
	}

	// Fetch the user with their role
	user, err := storage.GetUserByUsername(input.Username)
	if err != nil {
		logger.Log.WithError(err).Warn("Login: Invalid username or password")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", gin.H{"error": "Invalid username or password"})
		return
	}

	// Compare the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		logger.Log.WithError(err).Warn("Login: Invalid username or password")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", gin.H{"error": "Invalid username or password"})
		return
	}

	// Validate the Role
	if user.Role.Name == "" {
		logger.Log.WithField("username", user.Username).Warn("Login: User role is not assigned")
		c.Set("status", http.StatusUnauthorized)
		c.Set("response", gin.H{"error": "User role is not assigned"})
		return
	}

	// Generate JWT token with the user's role
	tokenString, err := auth.GenerateJWT(user.Username, user.Role.Name)
	if err != nil {
		logger.Log.WithError(err).Error("Login: Failed to generate token")
		c.Set("status", http.StatusInternalServerError)
		c.Set("response", gin.H{"error": "Failed to generate token"})
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
