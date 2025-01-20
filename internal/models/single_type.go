package models

import (
	"errors"
	"fmt"

	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/gorm"
)

type SingleType struct {
	gorm.Model
	Name        string      `json:"name" gorm:"uniqueIndex"`
	Description string      `json:"description"`
	Attributes  []Attribute `json:"attributes" gorm:"constraint:OnDelete:CASCADE;"`
}

// ParseSingleTypeInput transforms a generic map input into a SingleType struct.
// It mirrors the logic from ParseCollectionInput but adapts for single types.
func ParseSingleTypeInput(input map[string]interface{}) (SingleType, error) {
	var st SingleType

	// Extract basic fields
	if name, ok := input["name"].(string); ok {
		st.Name = name
	}
	if description, ok := input["description"].(string); ok {
		st.Description = description
	}

	// Attributes are expected to be a map of attributeName -> attributeDefinition
	rawAttributes, hasAttributes := input["attributes"].(map[string]interface{})
	if !hasAttributes {
		return st, fmt.Errorf("missing or invalid 'attributes' field")
	}

	for attrName, rawAttr := range rawAttributes {
		attrMap, validMap := rawAttr.(map[string]interface{})
		if !validMap {
			return st, fmt.Errorf("invalid attribute format for '%s'", attrName)
		}

		attribute := Attribute{Name: attrName, SingleTypeID: &st.ID}
		if err := mapToAttribute(attrMap, &attribute); err != nil {
			return st, fmt.Errorf("failed to parse attribute '%s': %v", attrName, err)
		}

		st.Attributes = append(st.Attributes, attribute)
	}

	return st, nil
}

func ValidateSingleTypeSchema(st SingleType) error {
	if st.Name == "" {
		return errors.New("missing required field: 'name'")
	}

	if len(st.Attributes) == 0 {
		return errors.New("attributes array cannot be empty for a single type")
	}

	seen := make(map[string]bool)
	for _, attr := range st.Attributes {
		if seen[attr.Name] {
			return fmt.Errorf("duplicate attribute name: '%s'", attr.Name)
		}
		seen[attr.Name] = true

		if err := validateAttributeType(attr); err != nil {
			return err
		}

		if attr.Type == "relation" {
			if attr.Relation == "" || attr.Target == "" {
				return fmt.Errorf("relationship '%s' must define 'relation' and 'target'", attr.Name)
			}
			var relatedCollection Collection
			if err := database.DB.Where("name = ?", attr.Target).First(&relatedCollection).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("target collection '%s' for relationship '%s' does not exist", attr.Target, attr.Name)
				}
				return fmt.Errorf("failed to check target collection '%s': %w", attr.Target, err)
			}
			allowed := map[string]struct{}{
				"oneToOne":   {},
				"oneToMany":  {},
				"manyToMany": {},
			}
			if _, ok := allowed[attr.Relation]; !ok {
				return fmt.Errorf("invalid relation_type '%s' for relationship '%s'; allowed: oneToOne, oneToMany, manyToMany", attr.Relation, attr.Name)
			}
		}
	}
	return nil
}

// ValidateSingleTypeValues validates the input data against the SingleType schema (attributes).
func ValidateSingleTypeValues(st SingleType, data map[string]interface{}) error {
	for _, attribute := range st.Attributes {
		_, exists := data[attribute.Name]

		// Check for required attributes
		if attribute.Required && !exists {
			logger.Log.WithField("attribute", attribute.Name).Warn("Validation failed: missing required attribute")
			return fmt.Errorf("missing required attribute: '%s'", attribute.Name)
		}
		if !exists {
			continue // Skip validation for optional fields not provided
		}
	}

	logger.Log.WithField("singleType", st.Name).Info("SingleType data validation passed")
	return nil
}
