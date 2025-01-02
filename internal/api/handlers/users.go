package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// CreateUser handles creating a new user.
func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Log.WithError(err).Warn("Failed to bind JSON for user creation")
		c.Set("response", "Invalid input")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Use ValidateUser from models package
	if err := models.ValidateUser(user); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"username": user.Username,
			"email":    user.Email,
		}).Warn("User validation failed")
		c.Set("response", "User validation failed")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := storage.SaveUser(&user); err != nil {
		logger.Log.WithError(err).Error("Failed to save user")
		c.Set("response", "Failed to create user")
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     user.Role.Name,
	}).Info("User created successfully")
	c.Set("response", "User created successfully")
	c.Set("status", http.StatusCreated)
}

// GetAllUsers handles fetching all users.
func GetAllUsers(c *gin.Context) {
	users, err := storage.GetAllUsers()
	if err != nil {
		logger.Log.WithError(err).Error("Failed to fetch users")
		c.Set("response", "Failed to fetch users")
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Fetched all users successfully")
	c.Set("response", gin.H{"users": users})
	c.Set("status", http.StatusOK)
}

// GetUser handles fetching a single user by ID.
func GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Log.WithError(err).Warn("Invalid user ID in request")
		c.Set("response", "Invalid user ID")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	user, err := storage.GetUserByID(uint(id))
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"user_id": id,
		}).Warn("User not found")
		c.Set("response", "User not found")
		c.Set("details", err.Error())
		c.Set("status", http.StatusNotFound)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"user_id": id,
	}).Info("Fetched user successfully")
	c.Set("response", user)
	c.Set("details", err.Error())
	c.Set("status", http.StatusOK)
}

// UpdateUser handles updating a user's details.
func UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Log.WithError(err).Warn("Invalid user ID in request")
		c.Set("response", "Invalid user ID")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Log.WithError(err).Warn("Failed to bind JSON for user updates")
		c.Set("response", "Invalid input")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Use ValidateUserUpdates from models package
	if err := models.ValidateUserUpdates(updates); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"user_id": id,
			"updates": updates,
		}).Warn("User updates validation failed")
		c.Set("response", "User updates validation failed")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := storage.UpdateUser(uint(id), updates); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"user_id": id,
		}).Error("Failed to update user")
		c.Set("response", "Failed to update user")
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"user_id": id,
		"updates": updates,
	}).Info("User updated successfully")
	c.Set("response", gin.H{"message": "User updated successfully"})
	c.Set("status", http.StatusOK)
}

// DeleteUser handles deleting a user.
func DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Log.WithError(err).Warn("Invalid user ID in request")
		c.Set("response", "Invalid user ID")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := storage.DeleteUser(uint(id)); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"user_id": id,
		}).Error("Failed to delete user")
		c.Set("response", "Failed to delete user")
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"user_id": id,
	}).Info("User deleted successfully")
	c.Set("response", gin.H{"message": "User deleted successfully"})
	c.Set("status", http.StatusOK)
}
