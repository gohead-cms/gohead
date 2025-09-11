package models

import (
	"fmt"

	"gorm.io/gorm"
)

// Attribute defines the structure for fields in a collection, single type, or component.
type Attribute struct {
	gorm.Model

	// Basic fields
	Name         string   `json:"name"`
	Type         string   `json:"type"` // e.g., "text", "int", "bool", "date", "richtext", "enum", "relation", "component"
	Required     bool     `json:"required"`
	Unique       bool     `json:"unique,omitempty"`
	Options      []string `gorm:"type:json" json:"options,omitempty"`
	Min          *int     `json:"min,omitempty"`
	Max          *int     `json:"max,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	CustomErrors JSONMap  `gorm:"type:json" json:"custom_errors,omitempty"`

	// Relationship-specific fields
	Target       string `json:"target,omitempty"`    // Target collection for relationships, or component name
	Relation     string `json:"relation,omitempty"`  // e.g., "oneToOne", "oneToMany", "manyToMany", for relationships
	ComponentRef string `json:"component,omitempty"` // If Type="component", which component is referenced

	// Foreign keys (an attribute can belong to either a Collection, a SingleType, or a Component)
	CollectionID *uint       `json:"collection_id,omitempty"`
	Collection   *Collection `json:"-" gorm:"constraint:OnDelete:CASCADE;"`

	SingletonID *uint      `json:"single_type_id,omitempty"`
	Singleton   *Singleton `json:"-" gorm:"constraint:OnDelete:CASCADE;"`

	ComponentID *uint      `json:"component_id,omitempty"`
	Component   *Component `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

// ValidateParent ensures an attribute belongs exactly to one parent: Collection, SingleType, or Component.
func (attr *Attribute) ValidateParent() error {
	parentsSet := 0

	if attr.CollectionID != nil {
		parentsSet++
	}
	if attr.SingletonID != nil {
		parentsSet++
	}
	if attr.ComponentID != nil {
		parentsSet++
	}

	switch parentsSet {
	case 0:
		return fmt.Errorf("attribute '%s' must belong to either a collection, single type, or component", attr.Name)
	case 1:
		return nil // OK
	default:
		return fmt.Errorf("attribute '%s' cannot belong to more than one parent (collection, single type, or component)", attr.Name)
	}
}
