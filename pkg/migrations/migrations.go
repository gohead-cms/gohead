// pkg/migrations/migrations.go
package migrations

import (
	"github.com/gohead-cms/gohead/internal/models"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"gorm.io/gorm"
)

func MigrateDatabase(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Collection{},
		&models.Attribute{},
		&models.Singleton{},
		&models.SingleItem{},
		&models.Item{},
		&models.User{},
		&agents.Agent{},
	)
}
