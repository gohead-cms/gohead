// pkg/database/db.go
package database

import (
	"fmt"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDatabase initializes the database connection.
func InitDatabase(databaseURL string) (*gorm.DB, error) {
	var err error

	if strings.HasPrefix(databaseURL, "sqlite://") {
		dbPath := strings.TrimPrefix(databaseURL, "sqlite://")
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	} else {
		return nil, fmt.Errorf("unsupported database type: %s", databaseURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return DB, nil
}
