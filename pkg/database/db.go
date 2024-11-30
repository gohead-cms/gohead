package database

import (
	"gitlab.com/sudo.bngz/gohead/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDatabase() error {
	var err error
	DB, err = gorm.Open(sqlite.Open("cms.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	// Migrate the schema
	return DB.AutoMigrate(&models.ContentItem{}, &models.User{})
}
