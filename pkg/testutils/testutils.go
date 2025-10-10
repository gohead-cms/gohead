package testutils

import (
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// JSONMap is a custom type that implements driver.Valuer and sql.Scanner
type JSONMap map[string]interface{}

type UserRole struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`                                                // Role name (e.g., admin, editor, viewer)
	Description string  `json:"description"`                                         // Role description
	Permissions JSONMap `json:"permissions" gorm:"type:jsonb;default:'[]';not null"` // Use 'jsonb' for PostgreSQL // Permissions associated with the role
}

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

	err = db.AutoMigrate(&UserRole{})
	if err != nil {
		logger.Log.Error("Failed to migrate : ", err.Error())
		panic("Failed to migrate :  " + err.Error())
	}

	// Seed default roles
	roles := []UserRole{
		{Name: "admin", Description: "Administrator with full access", Permissions: JSONMap{"manage_users": true, "manage_content": true}},
		{Name: "editor", Description: "Editor with content management access", Permissions: JSONMap{"manage_content": true}},
		{Name: "viewer", Description: "Viewer with read-only access", Permissions: JSONMap{"read_content": true}},
	}

	for _, role := range roles {
		if err := database.DB.FirstOrCreate(&UserRole{}, role).Error; err != nil {
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
