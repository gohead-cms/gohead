package seed

import (
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

func SeedRoles() {
	roles := []models.UserRole{
		{Name: "admin", Description: "Administrator with full access", Permissions: models.JSONMap{"manage_users": true, "manage_content": true}},
		{Name: "editor", Description: "Editor with content management access", Permissions: models.JSONMap{"manage_content": true}},
		{Name: "viewer", Description: "Viewer with read-only access", Permissions: models.JSONMap{"read_content": true}},
	}

	for _, role := range roles {
		if err := database.DB.FirstOrCreate(&models.UserRole{}, role).Error; err != nil {
			logger.Log.WithFields(logrus.Fields{
				"role": role.Name,
			}).Warn("Failed to seed role : ", err)
		}
	}
	logger.Log.Info("Default roles seeded successfully.")
}
