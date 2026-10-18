package models

import (
	"fmt"
	"regexp"

	"github.com/gohead-cms/gohead/internal/types"
	"gorm.io/gorm"
)

// Attribute defines the structure for fields in a collection, single type, or component.
type BaseAttribute struct {
	gorm.Model
	// Basic fields
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	Unique       bool     `json:"unique,omitempty"`
	Options      []string `gorm:"type:json;serializer:json" json:"options,omitempty"`
	Min          *int     `json:"min,omitempty"`
	Max          *int     `json:"max,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	CustomErrors JSONMap  `gorm:"type:json" json:"custom_errors,omitempty"`
	// Relationship-specific fields
	Target   string `json:"target,omitempty"`   // Target collection name for relationships
	Relation string `json:"relation,omitempty"` // e.g., "oneToOne", "oneToMany"

	// Component-specific fields
	ComponentRef string `json:"component,omitempty"`  // Name of the referenced component if Type="component"
	Repeatable   bool   `json:"repeatable,omitempty"` // Type="component": the value is a list of components

	// Media-specific fields
	Multiple     bool     `json:"multiple,omitempty"`                                       // Type="media": a list of files
	AllowedTypes []string `gorm:"type:json;serializer:json" json:"allowed_types,omitempty"` // Type="media": MIME patterns, e.g. "image/*"
}

// Attribute defines the structure for fields in a collection, single type, or component.
type Attribute struct {
	gorm.Model

	// Basic fields
	Name         string   `json:"name"`
	Type         string   `json:"type"` // e.g., "text", "int", "bool", "date", "richtext", "enum", "relation", "component"
	Required     bool     `json:"required"`
	Unique       bool     `json:"unique,omitempty"`
	Options      []string `gorm:"type:json;serializer:json" json:"options,omitempty"`
	Min          *int     `json:"min,omitempty"`
	Max          *int     `json:"max,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	CustomErrors JSONMap  `gorm:"type:json" json:"custom_errors,omitempty"`

	// Relationship-specific fields
	Target       string `json:"target,omitempty"`     // Target collection for relationships, or component name
	Relation     string `json:"relation,omitempty"`   // e.g., "oneToOne", "oneToMany", "manyToMany", for relationships
	ComponentRef string `json:"component,omitempty"`  // If Type="component", which component is referenced
	Repeatable   bool   `json:"repeatable,omitempty"` // Type="component": the value is a list of components

	// Media-specific fields
	Multiple     bool     `json:"multiple,omitempty"`                                       // Type="media": a list of files
	AllowedTypes []string `gorm:"type:json;serializer:json" json:"allowed_types,omitempty"` // Type="media": MIME patterns, e.g. "image/*"

	// Foreign keys (an attribute can belong to either a Collection, a SingleType, or a Component)
	CollectionID *uint       `json:"collection_id,omitempty"`
	Collection   *Collection `json:"-" gorm:"constraint:OnDelete:CASCADE;"`

	SingletonID *uint      `json:"single_type_id,omitempty"`
	Singleton   *Singleton `json:"-" gorm:"constraint:OnDelete:CASCADE;"`

	ComponentID *uint      `json:"component_id,omitempty"`
	Component   *Component `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

// mapToBaseAttribute populates a BaseAttribute (used by components) from a
// map, accepting the same keys as collection attributes.
func mapToBaseAttribute(data map[string]interface{}, attr *BaseAttribute) error {
	if _, ok := data["type"].(string); !ok {
		return fmt.Errorf("attribute is missing 'type'")
	}
	full := Attribute{Name: attr.Name}
	if err := mapToAttribute(data, &full); err != nil {
		return err
	}
	attr.Type = full.Type
	attr.Required = full.Required
	attr.Unique = full.Unique
	attr.Options = full.Options
	attr.Min = full.Min
	attr.Max = full.Max
	attr.Pattern = full.Pattern
	attr.Target = full.Target
	attr.Relation = full.Relation
	attr.ComponentRef = full.ComponentRef
	attr.Repeatable = full.Repeatable
	attr.Multiple = full.Multiple
	attr.AllowedTypes = full.AllowedTypes
	return nil
}

// ToAttribute converts a component attribute so it can go through the same
// value validation as collection attributes.
func (b BaseAttribute) ToAttribute() Attribute {
	return Attribute{
		Name:         b.Name,
		Type:         b.Type,
		Required:     b.Required,
		Options:      b.Options,
		Min:          b.Min,
		Max:          b.Max,
		Pattern:      b.Pattern,
		ComponentRef: b.ComponentRef,
		Repeatable:   b.Repeatable,
		Multiple:     b.Multiple,
		AllowedTypes: b.AllowedTypes,
	}
}

// validateAttributeType checks the attribute schema for constraints and valid types
// using the central TypeRegistry.
func validateBaseAttributeType(attribute BaseAttribute) error {
	// Check against the central type registry
	if _, exists := types.TypeRegistry[attribute.Type]; !exists {
		return fmt.Errorf("invalid attribute type '%s'", attribute.Type)
	}

	// Specific validation for 'enum' type
	if attribute.Type == "enum" && len(attribute.Options) == 0 {
		return fmt.Errorf("attribute '%s' of type 'enum' must have options", attribute.Name)
	}

	// Specific validation for 'int' type
	if (attribute.Type == "int" || attribute.Type == "integer") && attribute.Min != nil && attribute.Max != nil && *attribute.Min > *attribute.Max {
		return fmt.Errorf("for attribute '%s', min value cannot be greater than max value", attribute.Name)
	}

	// Specific validation for 'text' type pattern
	if attribute.Type == "text" && attribute.Pattern != "" {
		if _, err := regexp.Compile(attribute.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern for attribute '%s': %v", attribute.Name, err)
		}
	}

	return nil
}
