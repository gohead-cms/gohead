// pkg/migrations/migrations.go
package migrations

import (
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gorm.io/gorm"
)

func MigrateDatabase(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.ContentType{},
		&models.ContentRelation{},
		&models.ContentItem{},
		&models.Field{},
		&models.Relationship{},
		&models.User{},
	)
}
