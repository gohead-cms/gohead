// pkg/migrations/migrations.go
package migrations

import (
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gorm.io/gorm"
)

func MigrateDatabase(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Collection{},
		&models.Relationship{},
		&models.Item{},
		&models.Field{},
		&models.User{},
	)
}
