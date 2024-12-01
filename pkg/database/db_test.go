package database_test

import (
	"testing"

	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"

	"github.com/stretchr/testify/assert"
	gormlogger "gorm.io/gorm/logger"
)

func TestInitDatabase(t *testing.T) {
	// Initialize logger for testing
	logger.InitLogger("debug")

	t.Run("Initialize SQLite Database", func(t *testing.T) {
		db, err := database.InitDatabase("sqlite://:memory:", gormlogger.Silent)
		assert.NoError(t, err, "Expected no error for SQLite initialization")
		assert.NotNil(t, db, "Expected DB instance to be non-nil")
	})

	t.Run("Initialize MySQL Database", func(t *testing.T) {
		// Example DSN for a test MySQL database
		// Replace with actual credentials if running against a real MySQL instance
		dsn := "mysql://root:password@tcp(127.0.0.1:3306)/test_db?charset=utf8mb4&parseTime=True&loc=Local"

		db, err := database.InitDatabase(dsn, gormlogger.Silent)
		if err != nil {
			t.Log("Skipping MySQL test as no MySQL server is available.")
			t.SkipNow()
		}

		assert.NoError(t, err, "Expected no error for MySQL initialization")
		assert.NotNil(t, db, "Expected DB instance to be non-nil")
	})

	t.Run("Unsupported Database Type", func(t *testing.T) {
		db, err := database.InitDatabase("unsupported://db_path", gormlogger.Silent)
		assert.Error(t, err, "Expected an error for unsupported database type")
		assert.Nil(t, db, "Expected DB instance to be nil for unsupported database type")
		assert.Contains(t, err.Error(), "unsupported database type", "Error message should mention unsupported type")
	})
}
