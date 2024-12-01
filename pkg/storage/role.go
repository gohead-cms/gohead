package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
)

// GetRoleByName retrieves a role by its name.
func GetRoleByName(name string) (*models.UserRole, error) {
	var role models.UserRole
	err := database.DB.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, fmt.Errorf("role '%s' not found: %w", name, err)
	}
	return &role, nil
}
