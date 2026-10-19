package migrations

import (
	"time"

	"github.com/gohead-cms/gohead/internal/models"

	"gorm.io/gorm"
)

// Tables and columns as migration 0003 creates them (copies, so later model
// changes need their own migration).
type locale0003 struct {
	ID        uint   `gorm:"primaryKey"`
	Code      string `gorm:"size:35;not null;uniqueIndex:idx_locales_code"`
	Name      string `gorm:"size:191"`
	IsDefault bool   `gorm:"not null;default:false"`
	CreatedAt time.Time
}

func (locale0003) TableName() string { return "locales" }

type attribute0003 struct {
	Localized bool `gorm:"not null;default:false"`
}

func (attribute0003) TableName() string { return "attributes" }

type translations0003 struct {
	Translations models.JSONMap `gorm:"type:json"`
}

// i18nColumns are the tables that get a translations column.
var i18nColumns = []string{"items", "single_items", "revisions"}

func i18nUp(tx *gorm.DB) error {
	if err := tx.AutoMigrate(&locale0003{}); err != nil {
		return err
	}
	if !tx.Migrator().HasColumn(&attribute0003{}, "Localized") {
		if err := tx.Migrator().AddColumn(&attribute0003{}, "Localized"); err != nil {
			return err
		}
	}
	for _, table := range i18nColumns {
		t := tx.Table(table)
		if !t.Migrator().HasColumn(&translations0003{}, "Translations") {
			if err := tx.Table(table).Migrator().AddColumn(&translations0003{}, "Translations"); err != nil {
				return err
			}
		}
	}
	return nil
}

func i18nDown(tx *gorm.DB) error {
	for _, table := range i18nColumns {
		if tx.Table(table).Migrator().HasColumn(&translations0003{}, "Translations") {
			if err := tx.Table(table).Migrator().DropColumn(&translations0003{}, "Translations"); err != nil {
				return err
			}
		}
	}
	if tx.Migrator().HasColumn(&attribute0003{}, "Localized") {
		if err := tx.Migrator().DropColumn(&attribute0003{}, "Localized"); err != nil {
			return err
		}
	}
	return tx.Migrator().DropTable(&locale0003{})
}
