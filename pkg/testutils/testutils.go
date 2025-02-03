package testutils

import (
	"gohead/internal/models"
	"gohead/pkg/database"
	"gohead/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupTestServer initializes a Gin router and database for testing.
func SetupTestServer() (*gin.Engine, *gorm.DB) {
	// Initialize the logger
	logger.InitLogger("debug")
	// Initialize in-memory test database
	db, err := database.InitDatabase("sqlite://:memory:", gormlogger.Info)
	if err != nil {
		panic("Failed to initialize in-memory database: " + err.Error())
	}
	database.DB = db

	err = db.AutoMigrate(&models.UserRole{})
	if err != nil {
		logger.Log.Error("Failed to migrate : ", err.Error())
		panic("Failed to migrate :  " + err.Error())
	}

	// Seed default roles
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

	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Create a Gin router
	router := gin.Default()

	return router, db
}

func CleanupTestDB() {
	sqlDB, err := database.DB.DB()
	if err == nil {
		sqlDB.Close()
	}
}
