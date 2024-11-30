// internal/models/content_item.go
package models

import (
	"gorm.io/gorm"
)

type ContentItem struct {
	gorm.Model
	ContentType string                 `json:"content_type"`
	Data        map[string]interface{} `json:"data" gorm:"type:json"`
}
