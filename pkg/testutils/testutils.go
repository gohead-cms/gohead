package testutils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// JSONMap is a custom type that implements driver.Valuer and sql.Scanner.
// testutils cannot import internal/models (models tests import testutils),
// so it carries its own minimal copy.
type JSONMap map[string]interface{}

// Value serializes the map to JSON for storage.
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

// Scan deserializes JSON from the database.
func (m *JSONMap) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	case nil:
		*m = JSONMap{}
		return nil
	default:
		return fmt.Errorf("unsupported type for JSONMap: %T", value)
	}
	return json.Unmarshal(b, m)
}

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
	// Tests use an in-memory SQLite database by default. Set
	// GOHEAD_TEST_DATABASE_URL to a postgres:// URL to run the same tests
	// against PostgreSQL; its public schema is wiped before each setup.
	dbURL := os.Getenv("GOHEAD_TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "sqlite://:memory:"
	}
	db, err := database.InitDatabase(dbURL, gormlogger.Info)
	if err != nil {
		panic("Failed to initialize test database: " + err.Error())
	}
	if strings.HasPrefix(dbURL, "postgres://") {
		if err := db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public").Error; err != nil {
			panic("Failed to reset test schema: " + err.Error())
		}
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
		if err := database.DB.Where(UserRole{Name: role.Name}).Attrs(role).FirstOrCreate(&UserRole{}).Error; err != nil {
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
