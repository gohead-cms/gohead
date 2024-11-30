// pkg/database/db_test.go
package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitDatabase(t *testing.T) {
	t.Run("Initialize In-Memory SQLite Database", func(t *testing.T) {
		// Initialize an in-memory SQLite database
		db, err := InitDatabase("sqlite://:memory:")
		assert.NoError(t, err, "Should not return an error for in-memory SQLite database")
		assert.NotNil(t, db, "DB instance should not be nil for in-memory SQLite")

		// Verify the connection
		sqlDB, err := db.DB()
		assert.NoError(t, err, "Should not return an error for getting underlying SQL DB")
		assert.NotNil(t, sqlDB, "Underlying SQL DB should not be nil")
		assert.NoError(t, sqlDB.Ping(), "Should successfully ping the in-memory SQLite database")
	})

	t.Run("Initialize File-Based SQLite Database", func(t *testing.T) {
		// Define a temporary database file
		dbFilePath := "test_database.db"
		defer os.Remove(dbFilePath) // Clean up after test

		// Initialize SQLite database using the file
		db, err := InitDatabase("sqlite://" + dbFilePath)
		assert.NoError(t, err, "Should not return an error for file-based SQLite database")
		assert.NotNil(t, db, "DB instance should not be nil for file-based SQLite")

		// Verify the database file exists
		_, err = os.Stat(dbFilePath)
		assert.NoError(t, err, "Database file should exist")

		// Verify the connection
		sqlDB, err := db.DB()
		assert.NoError(t, err, "Should not return an error for getting underlying SQL DB")
		assert.NoError(t, sqlDB.Ping(), "Should successfully ping the file-based SQLite database")
	})

	t.Run("Unsupported Database Type", func(t *testing.T) {
		// Try initializing an unsupported database type
		db, err := InitDatabase("mysql://user:password@tcp(127.0.0.1:3306)/dbname")
		assert.Error(t, err, "Should return an error for unsupported database type")
		assert.Nil(t, db, "DB instance should be nil for unsupported database type")
		assert.Contains(t, err.Error(), "unsupported database type", "Error message should indicate unsupported database type")
	})
}
