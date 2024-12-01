package testutils

import (
	"github.com/gin-gonic/gin"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestServer initializes a Gin router and database for testing.
func SetupTestServer() (*gin.Engine, *gorm.DB) {
	// Initialize in-memory test database
	db, err := database.InitDatabase("sqlite://:memory:")
	if err != nil {
		panic("Failed to initialize in-memory database: " + err.Error())
	}
	database.DB = db

	// Initialize the logger
	logger.InitLogger("debug")

	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Create a Gin router
	router := gin.Default()

	return router, db
}

func SetupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to in-memory database")
	}
	database.DB = db // Assign it to your global database instance
	return db
}

func CleanupTestDB() {
	sqlDB, err := database.DB.DB()
	if err == nil {
		sqlDB.Close()
	}
}
