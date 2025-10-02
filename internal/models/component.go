package models

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type Component struct {
	gorm.Model
	ID          uint                 `json:"id"`
	Name        string               `json:"name" gorm:"uniqueIndex"`
	Description string               `json:"description"`
	Attributes  []ComponentAttribute `json:"attributes" gorm:"foreignKey:ComponentID;constraint:OnDelete:CASCADE;"`
}

type ComponentAttribute struct {
	BaseAttribute
	ComponentID uint `json:"-"`
}

func ParseComponentInput(input map[string]any) (Component, error) {
	var component Component

	// Extract basic fields
	if name, ok := input["name"].(string); ok && name != "" {
		component.Name = name
	} else {
		return component, fmt.Errorf("missing or invalid field 'name'")
	}

	if description, ok := input["description"].(string); ok {
		component.Description = description
	}

	// Extract and transform attributes
	if rawAttributes, ok := input["attributes"].(map[string]any); ok {
		for attrName, rawAttr := range rawAttributes {
			attrMap, ok := rawAttr.(map[string]any)
			if !ok {
				return component, fmt.Errorf("invalid attribute format for '%s'", attrName)
			}

			// Correctly create a ComponentAttribute
			var attribute ComponentAttribute
			attribute.Name = attrName // Set name from the key

			// Pass the embedded BaseAttribute to be populated
			if err := mapToBaseAttribute(attrMap, &attribute.BaseAttribute); err != nil {
				return component, fmt.Errorf("failed to parse attribute '%s': %v", attrName, err)
			}

			// Append the correctly typed attribute
			component.Attributes = append(component.Attributes, attribute)
		}
	}

	return component, nil
}

// ValidateComponentSchema ensures the component schema is valid.
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

		// Validate the attribute by passing the embedded BaseAttribute
		if err := validateBaseAttributeType(attr.BaseAttribute); err != nil {
			return fmt.Errorf("invalid attribute '%s' in component '%s': %v", attr.Name, cmp.Name, err)
		}

		// If the attribute type is "component", verify the nested component reference is valid.
		// It now checks `ComponentRef` from the embedded BaseAttribute.
		if attr.Type == "component" {
			if attr.ComponentRef == "" {
				return fmt.Errorf("attribute '%s' in component '%s' has type 'component' but is missing the 'component' reference name", attr.Name, cmp.Name)
			}
		}
	}

	return nil
}
