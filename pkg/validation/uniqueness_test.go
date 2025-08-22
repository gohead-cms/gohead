package validation

import (
	"bytes"
	"testing"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

// SetupTestDB initializes a test database using SQLite.
func SetupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	// Mock schema for items table
	db.Exec(`
		CREATE TABLE items (
			id INTEGER PRIMARY KEY,
			collection_id INTEGER,
			data JSON
		);
	`)

	return db
}

func TestCheckFieldUniqueness(t *testing.T) {
	// Initialize the test database
	testDB := SetupTestDB()
	database.DB = testDB

	// Seed the database with mock data
	testDB.Exec(`INSERT INTO items (collection_id, data) VALUES
		(1, '{"field_name": "existing_value"}'),
		(1, '{"field_name": "another_value"}');`)

	t.Run("Unique field value", func(t *testing.T) {
		err := CheckFieldUniqueness(1, "field_name", "unique_value")
		assert.NoError(t, err)
	})

	t.Run("Non-unique field value", func(t *testing.T) {
		err := CheckFieldUniqueness(1, "field_name", "existing_value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value for field 'field_name' must be unique")
	})

	t.Run("Different collection - should not conflict", func(t *testing.T) {
		err := CheckFieldUniqueness(2, "field_name", "existing_value")
		assert.NoError(t, err)
	})

	t.Run("Error during validation - malformed JSON", func(t *testing.T) {
		// Seed malformed data
		testDB.Exec(`INSERT INTO items (collection_id, data) VALUES
			(1, '{malformed_json}');`)

		err := CheckFieldUniqueness(1, "field_name", "unique_value")
		assert.Error(t, err) // The malformed row should not impact other checks
	})
}
