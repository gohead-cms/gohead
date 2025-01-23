package models

import (
	"fmt"

	"gorm.io/gorm"
)

// Attribute defines the structure for fields in a collection.
type Attribute struct {
	gorm.Model
	ID           uint     `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"` // e.g., "text", "int", "bool", "date", "richtext", "enum", "relation"
	Required     bool     `json:"required"`
	Unique       bool     `json:"unique,omitempty"`
	Options      []string `gorm:"type:json" json:"options,omitempty"`
	Min          *int     `json:"min,omitempty"`
	Max          *int     `json:"max,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	CustomErrors JSONMap  `gorm:"type:json" json:"custom_errors,omitempty"`
	Target       string   `json:"target,omitempty"`   // Target collection for relationships
	Relation     string   `json:"relation,omitempty"` // e.g., "oneToOne", "oneToMany", "manyToMany"

	// Foreign keys
	CollectionID *uint       `json:"collection_id"` // Nullable foreign key
	Collection   *Collection `json:"-" gorm:"constraint:OnDelete:CASCADE;"`

	SingleTypeID *uint       `json:"single_type_id"` // Nullable foreign key
	SingleType   *SingleType `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

func (attr *Attribute) ValidateParent() error {
	if attr.CollectionID != nil && attr.SingleTypeID != nil {
		return fmt.Errorf("attribute '%s' cannot belong to both a collection and a single type", attr.Name)
	}
	if attr.CollectionID == nil && attr.SingleTypeID == nil {
		return fmt.Errorf("attribute '%s' must belong to either a collection or a single type", attr.Name)
	}
	return nil
}
