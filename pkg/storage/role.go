package storage

import (
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
)

// CreateRole creates a new role.
func SaveRole(role *models.UserRole) error {
	if err := database.DB.Create(role).Error; err != nil {
		logger.Log.WithError(err).WithField("role", role.Name).Error("Failed to create role")
		return fmt.Errorf("failed to create role: %w", err)
	}
	logger.Log.WithField("role", role.Name).Info("Role created successfully")
	return nil
}

// GetRoleByName retrieves a role by its name.
func GetRoleByName(name string) (*models.UserRole, error) {
	var role models.UserRole
	err := database.DB.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, fmt.Errorf("role '%s' not found: %w", name, err)
	}
	return &role, nil
}

// GetRoleByID retrieves a role by its ID.
func GetRoleByID(id uint) (*models.UserRole, error) {
	var role models.UserRole
	if err := database.DB.First(&role, id).Error; err != nil {
		logger.Log.WithError(err).WithField("id", id).Warn("Role not found")
		return nil, fmt.Errorf("role with ID %d not found: %w", id, err)
	}
	return &role, nil
}

// GetAllRoles retrieves all roles from the database.
func GetAllRoles() ([]models.UserRole, error) {
	var roles []models.UserRole
	if err := database.DB.Find(&roles).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to retrieve roles")
		return nil, fmt.Errorf("failed to retrieve roles: %w", err)
	}
	return roles, nil
}

// UpdateRole updates an existing role by ID.
func UpdateRole(id uint, updates map[string]interface{}) error {
	var role models.UserRole
	if err := database.DB.First(&role, id).Error; err != nil {
		logger.Log.WithError(err).WithField("id", id).Warn("Role not found")
		return fmt.Errorf("role with ID %d not found: %w", id, err)
	}

	if err := database.DB.Model(&role).Updates(updates).Error; err != nil {
		logger.Log.WithError(err).WithField("id", id).Error("Failed to update role")
		return fmt.Errorf("failed to update role: %w", err)
	}
	logger.Log.WithField("role", role.Name).Info("Role updated successfully")
	return nil
}

// DeleteRole deletes a role by ID.
func DeleteRole(id uint) error {
	if err := database.DB.Delete(&models.UserRole{}, id).Error; err != nil {
		logger.Log.WithError(err).WithField("id", id).Error("Failed to delete role")
		return fmt.Errorf("failed to delete role: %w", err)
	}
	logger.Log.WithField("id", id).Info("Role deleted successfully")
	return nil
}
