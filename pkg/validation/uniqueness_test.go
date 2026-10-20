package validation

import (
	"bytes"
	"testing"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/ncruces/go-sqlite3/gormlite"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	db, err := gorm.Open(gormlite.Open(database.SQLiteDSN(":memory:")), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	// Mock schema for items table
	db.Exec(`
		CREATE TABLE items (
			id INTEGER PRIMARY KEY,
			collection_id INTEGER,
			data JSON,
			deleted_at DATETIME
		);
	`)

	return db
}

func TestCheckFieldUniqueness(t *testing.T) {
	// Initialize the test database
	testDB := SetupTestDB()
	database.DB = testDB

	// Seed the database with mock data
	testDB.Exec(`INSERT INTO items (id, collection_id, data, deleted_at) VALUES
		(1, 1, '{"field_name": "existing_value", "n": 5, "big": 1000000, "small": 0.00001, "flag": true, "my-field": "x"}', NULL),
		(2, 1, '{"field_name": "another_value"}', NULL),
		(3, 1, '{"field_name": "deleted_value"}', '2026-01-01 00:00:00');`)

	t.Run("Unique field value", func(t *testing.T) {
		err := CheckFieldUniqueness(nil, 1, "field_name", "unique_value", 0)
		assert.NoError(t, err)
	})

	t.Run("Non-unique field value", func(t *testing.T) {
		err := CheckFieldUniqueness(nil, 1, "field_name", "existing_value", 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value for field 'field_name' must be unique")
	})

	t.Run("Different collection - should not conflict", func(t *testing.T) {
		err := CheckFieldUniqueness(nil, 2, "field_name", "existing_value", 0)
		assert.NoError(t, err)
	})

	t.Run("The item being updated keeps its own value", func(t *testing.T) {
		assert.NoError(t, CheckFieldUniqueness(nil, 1, "field_name", "existing_value", 1))
		assert.Error(t, CheckFieldUniqueness(nil, 1, "field_name", "existing_value", 2))
	})

	t.Run("Soft-deleted items do not count", func(t *testing.T) {
		assert.NoError(t, CheckFieldUniqueness(nil, 1, "field_name", "deleted_value", 0))
	})

	t.Run("Numbers compare by value", func(t *testing.T) {
		assert.Error(t, CheckFieldUniqueness(nil, 1, "n", float64(5), 0))
		assert.NoError(t, CheckFieldUniqueness(nil, 1, "n", 6, 0))
		assert.Error(t, CheckFieldUniqueness(nil, 1, "big", float64(1000000), 0), "no exponent formatting")
		assert.Error(t, CheckFieldUniqueness(nil, 1, "small", 0.00001, 0))
	})

	t.Run("Booleans and unusual names", func(t *testing.T) {
		assert.Error(t, CheckFieldUniqueness(nil, 1, "flag", true, 0))
		assert.NoError(t, CheckFieldUniqueness(nil, 1, "flag", false, 0))
		assert.Error(t, CheckFieldUniqueness(nil, 1, "my-field", "x", 0))
	})

	t.Run("Error during validation - malformed JSON", func(t *testing.T) {
		// Seed malformed data
		testDB.Exec(`INSERT INTO items (collection_id, data) VALUES
			(1, '{malformed_json}');`)

		err := CheckFieldUniqueness(nil, 1, "field_name", "unique_value", 0)
		assert.Error(t, err) // The malformed row should not impact other checks
	})
}
