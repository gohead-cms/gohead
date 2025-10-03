package models

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// Singleton represents a content structure that has only one instance.
// e.g., a homepage, site settings, or a header.
type Singleton struct {
	gorm.Model
	ID          uint        `json:"id"`
	Name        string      `json:"name" gorm:"uniqueIndex"`
	Description string      `json:"description"`
	Attributes  []Attribute `json:"attributes" gorm:"constraint:OnDelete:CASCADE;"`
}

// ParseSingletonInput transforms a generic map input into a Singleton struct.
func ParseSingletonInput(input map[string]any) (Singleton, error) {
	var singleton Singleton

	// Extract basic fields
	if name, ok := input["name"].(string); ok {
		singleton.Name = name
	}
	if description, ok := input["description"].(string); ok {
		singleton.Description = description
	}

	// Attributes are expected to be a map of attributeName -> attributeDefinition
	rawAttributes, hasAttributes := input["attributes"].(map[string]any)
	if !hasAttributes {
		return singleton, fmt.Errorf("missing or invalid 'attributes' field")
	}

	for attrName, rawAttr := range rawAttributes {
		attrMap, validMap := rawAttr.(map[string]any)
		if !validMap {
			return singleton, fmt.Errorf("invalid attribute format for '%s'", attrName)
		}
		// NOTE: Assumes the Attribute model will be updated to have a SingletonID field.
		attribute := Attribute{Name: attrName, SingletonID: &singleton.ID}
		if err := mapToAttribute(attrMap, &attribute); err != nil {
			return singleton, fmt.Errorf("failed to parse attribute '%s': %v", attrName, err)
		}

		singleton.Attributes = append(singleton.Attributes, attribute)
	}

	return singleton, nil
}

// ValidateSingletonSchema ensures the structure of a singleton is valid.
func ValidateSingletonSchema(singleton Singleton) error {
	if singleton.Name == "" {
		return errors.New("missing required field: 'name'")
	}

	if len(singleton.Attributes) == 0 {
		return errors.New("attributes array cannot be empty for a singleton")
	}

	seen := make(map[string]bool)
	for _, attr := range singleton.Attributes {
		if seen[attr.Name] {
			return fmt.Errorf("duplicate attribute name: '%s'", attr.Name)
		}
		seen[attr.Name] = true

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

// ValidateSingletonValues checks if provided data conforms to the singleton's schema.
func ValidateSingletonValues(singleton Singleton, data map[string]any) error {
	// 1. Build a set of valid attribute names
	validAttributes := make(map[string]Attribute, len(singleton.Attributes))
	for _, attr := range singleton.Attributes {
		validAttributes[attr.Name] = attr
	}

	// 2. Check for unknown fields in 'data'
	for key := range data {
		if _, ok := validAttributes[key]; !ok {
			logger.Log.WithField("attribute", key).Warn("Validation failed: unknown attribute")
			return fmt.Errorf("unknown attribute: '%s'", key)
		}
	}

	// 3. For each attribute in the schema, check required & apply pattern checks
	for _, attribute := range singleton.Attributes {
		val, exists := data[attribute.Name]

		// Check for required attributes
		if attribute.Required && !exists {
			logger.Log.WithField("attribute", attribute.Name).Warn("Validation failed: missing required attribute")
			return fmt.Errorf("missing required attribute: '%s'", attribute.Name)
		}
		if !exists {
			continue
		}

		if attribute.Pattern != "" && attribute.Type == "string" {
			strVal, err := ensureString(val)
			if err != nil {
				logger.Log.WithField("attribute", attribute.Name).
					Warnf("Validation failed: expected string, got %v", val)
				return fmt.Errorf("attribute '%s' must be a string to check pattern", attribute.Name)
			}

			if matchErr := matchPattern(attribute.Pattern, strVal, attribute.Name); matchErr != nil {
				return matchErr
			}
		}
	}

	logger.Log.WithField("singleton", singleton.Name).Debug("Singleton data validation passed")
	return nil
}

func ensureString(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("not a string value")
	}
}

func matchPattern(pattern string, val string, attrName string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern for attribute '%s': %v", attrName, err)
	}

	if !re.MatchString(val) {
		return fmt.Errorf("attribute '%s' value '%s' does not match pattern '%s'", attrName, val, pattern)
	}
	return nil
}
