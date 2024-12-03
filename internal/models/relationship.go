// internal/models/content_relation.go
package models

import "gorm.io/gorm"

type Relationship struct {
	gorm.Model
	FieldName         string `json:"field_name"`
	RelationType      string `json:"relation_type"`
	CollectionID      uint   `json:"-"`
	RelatedCollection uint   `json:"related_collection"`
	ItemID            uint   `json:"item_id"`
	RelatedItemID     uint   `json:"related_item_id"`
}
