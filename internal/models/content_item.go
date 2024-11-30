package models

import "gorm.io/gorm"

type ContentItem struct {
	gorm.Model
	ContentType string  `json:"content_type"`
	Data        JSONMap `json:"data" gorm:"type:json"`
}
