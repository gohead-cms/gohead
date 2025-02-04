// internal/models/component.go
package models

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type Component struct {
	gorm.Model
	ID          uint        `json:"id"`
	Name        string      `json:"name" gorm:"uniqueIndex"`
	Description string      `json:"description"`
	Attributes  []Attribute `json:"attributes" gorm:"constraint:OnDelete:CASCADE;"`
}

func ParseComponentInput(input map[string]interface{}) (Component, error) {
	var component Component

	// Extract basic fields
	if name, ok := input["name"].(string); ok {
		component.Name = name
	} else {
		return component, fmt.Errorf("missing or invalid field 'name'")
	}

	if description, ok := input["description"].(string); ok {
		component.Description = description
	}

	// Extract and transform attributes
	if rawAttributes, ok := input["attributes"].(map[string]interface{}); ok {
		for attrName, rawAttr := range rawAttributes {
			attrMap, ok := rawAttr.(map[string]interface{})
			if !ok {
				return component, fmt.Errorf("invalid attribute format for '%s'", attrName)
			}

			attribute := Attribute{Name: attrName}
			if err := mapToAttribute(attrMap, &attribute); err != nil {
				return component, fmt.Errorf("failed to parse attribute '%s': %v", attrName, err)
			}

			component.Attributes = append(component.Attributes, attribute)
		}
	} else {
		return component, fmt.Errorf("missing or invalid field 'attributes'")
	}

	return component, nil
}

// ValidateComponentSchema ensures the component schema is valid:
// - 'Name' is present
// - At least one attribute
// - No duplicate attributes
// - Valid attribute types (including nested component references)
func ValidateComponentSchema(cmp Component) error {
	if cmp.Name == "" {
		return errors.New("component must have a 'name'")
	}

	if len(cmp.Attributes) == 0 {
		return errors.New("a component must have at least one attribute")
	}

	seen := make(map[string]bool)
	for _, attr := range cmp.Attributes {
		// Check duplicate attribute names
		if seen[attr.Name] {
			return fmt.Errorf("duplicate attribute name: '%s' in component '%s'", attr.Name, cmp.Name)
		}
		seen[attr.Name] = true

		// Validate the attribute type (string, int, bool, component, etc.)
		if err := validateAttributeType(attr); err != nil {
			return fmt.Errorf("invalid attribute '%s' in component '%s': %v", attr.Name, cmp.Name, err)
		}

		// If the attribute type is "component", verify the nested component name is valid
		if attr.Type == "component" {
			if attr.Component == nil {
				return fmt.Errorf("attribute '%s' in component '%s' has type 'component' but no 'component' name", attr.Name, cmp.Name)
			}

			// TODO RELATION CHECK
			// Optionally check the DB to confirm the referenced component actually exists
			// var nestedCmp Component
			// if err := database.DB.Where("name = ?", attr.Component).First(&nestedCmp).Error; err != nil {
			// 	if errors.Is(err, gorm.ErrRecordNotFound) {
			// 		return fmt.Errorf("referenced component '%s' (for attribute '%s') does not exist", attr.Component, attr.Name)
			// 	}
			// 	return fmt.Errorf("failed to check nested component '%s' for attribute '%s': %w", attr.Component, attr.Name, err)
			// }
		}
	}

	return nil
}
