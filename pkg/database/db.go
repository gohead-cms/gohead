// pkg/database/db.go
package database

import (
	"fmt"
	"strings"

	"gitlab.com/sudo.bngz/gohead/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDatabase(databaseURL string) error {
	var err error

	// Parse the database URL
	if strings.HasPrefix(databaseURL, "sqlite://") {
		// Remove the 'sqlite://' prefix
		dbPath := strings.TrimPrefix(databaseURL, "sqlite://")
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	} else {
		return fmt.Errorf("unsupported database type")
	}

	if err != nil {
		return err
	}
	// Migrate the schema
	return DB.AutoMigrate(
		&models.ContentItem{},
		&models.User{},
		&models.ContentRelation{},
	)
}
