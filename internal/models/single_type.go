package models

import (
	"errors"
	"fmt"
	"regexp"

	"gohead/pkg/database"
	"gohead/pkg/logger"

	"gorm.io/gorm"
)

type SingleType struct {
	gorm.Model
	ID          uint        `json:"id"`
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

func ValidateSingleTypeValues(st SingleType, data map[string]interface{}) error {
	// 1. Build a set of valid attribute names
	validAttributes := make(map[string]Attribute, len(st.Attributes))
	for _, attr := range st.Attributes {
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
	for _, attribute := range st.Attributes {
		val, exists := data[attribute.Name]

		// Check for required attributes
		if attribute.Required && !exists {
			logger.Log.WithField("attribute", attribute.Name).Warn("Validation failed: missing required attribute")
			return fmt.Errorf("missing required attribute: '%s'", attribute.Name)
		}
		if !exists {
			// Not required & not provided => skip further checks
			continue
		}

		// If there's a pattern, validate only if the attribute's type is "string"
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

		// If you have other checks (e.g., type checks, min/max length, etc.), do them here.
	}

	logger.Log.WithField("singleType", st.Name).Debug("SingleType data validation passed")
	return nil
}

func ensureString(val interface{}) (string, error) {
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
