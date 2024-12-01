package handlers

import (
	"fmt"
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
	// Parse input
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

	// Fetch the 'viewer' role
	role, err := storage.GetRoleByName("admin")
	if err != nil {
		logger.Log.WithError(err).Error("Register: Failed to fetch 'viewer' role")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch default role"})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Log.WithError(err).Error("Register: Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user instance
	user := models.User{
		Username:  input.Username,
		Password:  string(hashedPassword),
		RoleRefer: role.ID,
		Email:     input.Email,
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
	case *storage.DuplicateEntryError:
		logger.Log.WithError(err).Warn("Register: Duplicate entry")
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s", e.Error())})
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

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     role.Name,
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
