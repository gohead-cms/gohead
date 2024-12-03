package models

import "gorm.io/gorm"

type Item struct {
	gorm.Model
	CollectionID uint    `json:"collection"`
	Data         JSONMap `json:"data" gorm:"type:json"`
}
