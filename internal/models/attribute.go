package models

import "gorm.io/gorm"

// Attribute defines the structure for fields in a collection.
type Attribute struct {
	gorm.Model
	Name         string   `json:"name"`
	Type         string   `json:"type"` // e.g., "string", "int", "bool", "date", "richtext", "enum", "relation"
	Required     bool     `json:"required"`
	Unique       bool     `json:"unique,omitempty"`
	Options      []string `gorm:"type:json" json:"options,omitempty"`
	Min          *int     `json:"min,omitempty"`
	Max          *int     `json:"max,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	CustomErrors JSONMap  `gorm:"type:json" json:"custom_errors,omitempty"`
	Target       string   `json:"target,omitempty"`   // Target collection for relationships
	Relation     string   `json:"relation,omitempty"` // e.g., "oneToOne", "oneToMany", "manyToMany"
	CollectionID uint     `json:"-"`                  // Foreign key to associate with Collection
}
