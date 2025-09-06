package models

import (
	"gorm.io/gorm"
)

type Agent struct {
	gorm.Model        // Provides ID, CreatedAt, UpdatedAt, and DeletedAt
	Name       string `gorm:"uniqueIndex"`
	Config     string `gorm:"type:jsonb"` // Store the entire config as JSONB
}
