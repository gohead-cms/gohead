package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("Register: Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Log.WithError(err).Error("Register: Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	role := "viewer"

	// Validate the user role
	userRole := models.UserRole{
		Name:        role,
		Description: "Default user role",
		Permissions: models.JSONMap{"perms": "read"}, // Example default permissions
	}

	if err := models.ValidateUserRole(userRole); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"role": role,
		}).Warn("Register: Invalid role", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create user instance
	user := models.User{
		Username: input.Username,
		Password: string(hashedPassword),
		Role:     userRole,
		Email:    input.Email,
	}

	// Validate user
	if err := models.ValidateUser(user); err != nil {
		logger.Log.WithError(err).Warn("Register: User validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save the user
	if err := storage.SaveUser(&user); err != nil {
		logger.Log.WithError(err).Warn("Register: Username already exists")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     user.Role.Name,
	}).Info("User registered successfully")

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
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
	tokenString, err := auth.GenerateJWT(user.Username, user.Role.Name)
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
