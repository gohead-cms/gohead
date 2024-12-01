package database

import (
	"fmt"
	"strings"

	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// InitDatabase initializes the database connection
func InitDatabase(databaseURL string, logLevel gormlogger.LogLevel) (*gorm.DB, error) {
	// Create a custom GORM logger
	gormLogger := logger.NewGormLogger(logLevel)

	var db *gorm.DB
	var err error

	// Check database type from the URL prefix
	if strings.HasPrefix(databaseURL, "sqlite://") {
		// SQLite configuration
		dbPath := strings.TrimPrefix(databaseURL, "sqlite://")
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: gormLogger,
		})
	} else if strings.HasPrefix(databaseURL, "mysql://") {
		// MySQL configuration
		dsn := strings.TrimPrefix(databaseURL, "mysql://") // Remove the prefix to extract the DSN
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: gormLogger,
		})
	} else {
		return nil, fmt.Errorf("unsupported database type: %s", databaseURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Assign the initialized database to the global DB
	DB = db
	return DB, nil
}
