package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CreateUser lets an admin create an account with any existing role.
func CreateUser(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required"`
		RoleName string `json:"role_name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("Failed to bind JSON for user creation")
		c.Set("response", "Invalid input")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}
	if input.RoleName == "" {
		input.RoleName = DefaultRegistrationRole
	}

	role, err := storage.AssignableRole(input.RoleName)
	if err != nil {
		c.Set("response", "User validation failed")
		c.Set("details", roleProblem(err, input.RoleName))
		c.Set("status", http.StatusBadRequest)
		return
	}

	user, status, err := buildUser(input.Username, input.Email, input.Password, role)
	if err != nil {
		c.Set("response", "User validation failed")
		c.Set("details", err.Error())
		c.Set("status", status)
		return
	}

	if status, msg, details := saveNewUser(user); status != 0 {
		c.Set("response", msg)
		c.Set("details", details)
		c.Set("status", status)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"username": user.Username,
		"role":     role.Name,
	}).Info("User created successfully")
	c.Set("response", user)
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
	c.Set("status", http.StatusOK)
}

// UpdateUser lets an admin change a user's username, email, password or role.
// Unknown fields are rejected so that arbitrary columns cannot be written.
func UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Set("response", "Invalid user ID")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.Set("response", "Invalid input")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// role_name is resolved to a foreign key; everything else goes through
	// the model's field whitelist.
	updates := map[string]interface{}{}
	var roleName string
	for k, v := range raw {
		if k == "role_name" {
			name, ok := v.(string)
			if !ok || name == "" {
				c.Set("response", "User updates validation failed")
				c.Set("details", "role_name must be a non-empty string")
				c.Set("status", http.StatusBadRequest)
				return
			}
			roleName = name
			continue
		}
		updates[k] = v
	}
	delete(updates, "role") // roles change only through role_name

	if err := models.ValidateUserUpdates(updates); err != nil {
		c.Set("response", "User updates validation failed")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if pw, ok := updates["password"].(string); ok {
		hashed, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			c.Set("response", "Failed to update user")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		updates["password"] = string(hashed)
	}
	if roleName != "" {
		role, err := storage.AssignableRole(roleName)
		if err != nil {
			c.Set("response", "User updates validation failed")
			c.Set("details", roleProblem(err, roleName))
			c.Set("status", http.StatusBadRequest)
			return
		}
		if role.Name != "admin" && !adminCanStepDown(c, uint(id)) {
			return
		}
		updates["user_role_id"] = role.ID
	}
	if len(updates) == 0 {
		c.Set("response", "User updates validation failed")
		c.Set("details", "no fields to update")
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := storage.UpdateUser(uint(id), updates); err != nil {
		logger.Log.WithError(err).WithField("user_id", id).Error("Failed to update user")
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.Set("response", "Failed to update user")
		c.Set("details", err.Error())
		c.Set("status", status)
		return
	}

	logger.Log.WithField("user_id", id).Info("User updated successfully")
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

	// Prevent an admin from locking themselves out.
	if target, err := storage.GetUserByID(uint(id)); err == nil {
		if current, ok := c.Get("username"); ok && current == target.Username {
			c.Set("response", "Cannot delete your own account")
			c.Set("status", http.StatusBadRequest)
			return
		}
	}
	if !adminCanStepDown(c, uint(id)) {
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

// adminCanStepDown checks that user id may lose the admin role (deletion or
// demotion): never your own account, never the last admin. It answers the
// request (409) and returns false when not.
func adminCanStepDown(c *gin.Context, id uint) bool {
	target, err := storage.GetUserByID(id)
	if err != nil || target.Role.Name != "admin" {
		return true // not an admin (or unknown: the caller reports it)
	}
	if current, ok := c.Get("username"); ok && current == target.Username {
		c.Set("response", "Cannot remove your own admin role")
		c.Set("details", "ask another admin")
		c.Set("status", http.StatusConflict)
		return false
	}
	admins, err := storage.CountUsersWithRole("admin")
	if err != nil {
		c.Set("response", "Failed to check the remaining admins")
		c.Set("status", http.StatusInternalServerError)
		return false
	}
	if admins <= 1 {
		c.Set("response", "Cannot remove the last admin")
		c.Set("details", "promote another user to admin first")
		c.Set("status", http.StatusConflict)
		return false
	}
	return true
}

// roleProblem explains why a role cannot be given to an account.
func roleProblem(err error, name string) string {
	if errors.Is(err, storage.ErrPublicRole) {
		return err.Error()
	}
	return "unknown role: " + name
}
