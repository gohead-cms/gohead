package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"` // Optional
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.Warn("Register: Invalid input", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Set default role if none provided
	role := input.Role
	if role == "" {
		role = "viewer"
	}

	// Validate role
	validRoles := []string{"admin", "editor", "viewer"}
	if !contains(validRoles, role) {
		logger.Log.Warn("Register: Invalid role", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	user := models.User{
		Username: input.Username,
		Password: string(hashedPassword),
		Role:     role,
	}

	// Save the user
	if err := database.DB.Create(&user).Error; err != nil {
		logger.Log.Warn("Register: Username already exists", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     user.Role,
	}).Info("User registered successfully")

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.Warn("Login: Error bad request ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		logger.Log.Warn("Login: Invalid username or password", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		logger.Log.Warn("Login: Invalid username or password", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Generate JWT token with role
	tokenString, err := auth.GenerateJWT(user.Username, user.Role)
	if err != nil {
		logger.Log.Warn("Login: Failed to generate token", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     user.Role,
	}).Info("User logged in successfully")

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
