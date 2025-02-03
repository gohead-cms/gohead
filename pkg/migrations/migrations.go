// pkg/migrations/migrations.go
package migrations

import (
	"gohead/internal/models"

	"gorm.io/gorm"
)

func MigrateDatabase(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Collection{},
		&models.Attribute{},
		&models.SingleType{},
		&models.SingleItem{},
		&models.Item{},
		&models.User{},
	)
}
