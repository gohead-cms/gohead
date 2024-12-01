package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// CreateUser handles creating a new user.
func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Log.WithError(err).Warn("Failed to bind JSON for user creation")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Use ValidateUser from models package
	if err := models.ValidateUser(user); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"username": user.Username,
			"email":    user.Email,
		}).Warn("User validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := storage.SaveUser(&user); err != nil {
		logger.Log.WithError(err).Error("Failed to save user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     user.Role.Name,
	}).Info("User created successfully")
	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// GetAllUsers handles fetching all users.
func GetAllUsers(c *gin.Context) {
	users, err := storage.GetAllUsers()
	if err != nil {
		logger.Log.WithError(err).Error("Failed to fetch users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	logger.Log.Info("Fetched all users successfully")
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// GetUserByUsername fetches a user by their username, including their associated role.
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

// GetUser handles fetching a single user by ID.
func GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Log.WithError(err).Warn("Invalid user ID in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := storage.GetUserByID(uint(id))
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"user_id": id,
		}).Warn("User not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"user_id": id,
	}).Info("Fetched user successfully")
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateUser handles updating a user's details.
func UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Log.WithError(err).Warn("Invalid user ID in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Log.WithError(err).Warn("Failed to bind JSON for user updates")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Use ValidateUserUpdates from models package
	if err := models.ValidateUserUpdates(updates); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"user_id": id,
			"updates": updates,
		}).Warn("User updates validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := storage.UpdateUser(uint(id), updates); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"user_id": id,
		}).Error("Failed to update user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"user_id": id,
		"updates": updates,
	}).Info("User updated successfully")
	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser handles deleting a user.
func DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Log.WithError(err).Warn("Invalid user ID in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := storage.DeleteUser(uint(id)); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"user_id": id,
		}).Error("Failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"user_id": id,
	}).Info("User deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
