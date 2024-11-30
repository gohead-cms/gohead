// internal/models/content_relation.go
package models

import "gorm.io/gorm"

type ContentRelation struct {
	gorm.Model
	ContentType   string `json:"content_type"`
	ContentItemID uint   `json:"content_item_id"`
	RelatedType   string `json:"related_type"`
	RelatedItemID uint   `json:"related_item_id"`
	RelationType  string `json:"relation_type"` // "one-to-one", "one-to-many", "many-to-many"
	FieldName     string `json:"field_name"`
}
