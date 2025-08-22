package models

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gohead-cms/gohead/internal/types"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

type Collection struct {
	gorm.Model
	ID          uint        `json:"id"`
	Name        string      `json:"name" gorm:"uniqueIndex"`
	Kind        string      `json:"kind" gorm:"type:varchar(50);not null"`
	Description string      `json:"description"`
	Attributes  []Attribute `json:"attributes" gorm:"constraint:OnDelete:CASCADE;"`
}

func ParseCollectionInput(input map[string]any) (Collection, error) {
	// Initialize a Collection struct
	var collection Collection

	// Extract basic fields
	if name, ok := input["name"].(string); ok {
		collection.Name = name
	} else {
		return collection, fmt.Errorf("missing or invalid field 'name'")
	}

	if kind, ok := input["kind"].(string); ok {
		collection.Kind = kind
	} else {
		return collection, fmt.Errorf("missing or invalid field 'kind'")
	}

	if description, ok := input["description"].(string); ok {
		collection.Description = description
	}

	// Extract and transform attributes
	if rawAttributes, ok := input["attributes"].(map[string]interface{}); ok {
		for attrName, rawAttr := range rawAttributes {
			attrMap, ok := rawAttr.(map[string]interface{})
			if !ok {
				return collection, fmt.Errorf("invalid attribute format for '%s'", attrName)
			}

			attribute := Attribute{Name: attrName}
			if err := mapToAttribute(attrMap, &attribute); err != nil {
				return collection, fmt.Errorf("failed to parse attribute '%s': %v", attrName, err)
			}

			collection.Attributes = append(collection.Attributes, attribute)
		}
	} else {
		return collection, fmt.Errorf("missing or invalid field 'attributes'")
	}

	return collection, nil
}

func mapToAttribute(attrMap map[string]interface{}, attribute *Attribute) error {
	if attrType, ok := attrMap["type"].(string); ok {
		attribute.Type = attrType
	} else {
		return fmt.Errorf("missing or invalid field 'type'")
	}

	if required, ok := attrMap["required"].(bool); ok {
		attribute.Required = required
	}

	if unique, ok := attrMap["unique"].(bool); ok {
		attribute.Unique = unique
	}

	if options, ok := attrMap["options"].([]interface{}); ok {
		for _, option := range options {
			if strOption, ok := option.(string); ok {
				attribute.Options = append(attribute.Options, strOption)
			}
		}
	}

	if min, ok := attrMap["min"].(float64); ok {
		minInt := int(min)
		attribute.Min = &minInt
	}

	if max, ok := attrMap["max"].(float64); ok {
		maxInt := int(max)
		attribute.Max = &maxInt
	}

	if pattern, ok := attrMap["pattern"].(string); ok {
		attribute.Pattern = pattern
	}

	// Handle relation-specific fields
	if attribute.Type == "relation" {
		if relation, ok := attrMap["relation"].(string); ok {
			attribute.Relation = relation
		} else {
			return fmt.Errorf("missing or invalid field 'relation' for type 'relation'")
		}

		if target, ok := attrMap["target"].(string); ok {
			attribute.Target = target
		} else {
			return fmt.Errorf("missing or invalid field 'target' for type 'relation'")
		}
	}

	return nil
}

//
//
// -------------------- Schema validator
//

func ValidateCollectionSchema(ct Collection) error {
	if ct.Name == "" {
		return errors.New("missing required attribute: 'name'")
	}

	if len(ct.Attributes) == 0 {
		return errors.New("attributes array cannot be empty")
	}

	attributeNames := make(map[string]bool)
	for _, attribute := range ct.Attributes {
		// Prevent duplicate attributes
		if attributeNames[attribute.Name] {
			return fmt.Errorf("duplicate attribute name: '%s'", attribute.Name)
		}
		attributeNames[attribute.Name] = true

		logger.Log.WithField("attribute", attribute.Name).Debug("Validating attribute type:", attribute.Type)

		// --- Handle relationship type FIRST ---
		if attribute.Type == "relation" {
			logger.Log.WithField("attribute", attribute.Name).Debug("Validating relationship attributes")

			// Ensure required fields exist
			if attribute.Relation == "" || attribute.Target == "" {
				return fmt.Errorf("relationship '%s' must define 'relation' and 'target'", attribute.Name)
			}

			// Verify the target collection exists
			var relatedCollection Collection
			if err := database.DB.Where("name = ?", attribute.Target).First(&relatedCollection).Error; err != nil {
				logger.Log.WithField("collection", attribute.Target).
					WithError(err).
					Warn("Referenced collection does not exist")
				return fmt.Errorf("target collection '%s' for relationship '%s' does not exist", attribute.Target, attribute.Name)
			}

			// Validate relationship types
			allowedRelationTypes := map[string]struct{}{
				"oneToOne":   {},
				"oneToMany":  {},
				"manyToMany": {},
			}
			if _, isValid := allowedRelationTypes[attribute.Relation]; !isValid {
				logger.Log.WithFields(map[string]interface{}{
					"relation_type": attribute.Relation,
					"attribute":     attribute.Name,
				}).Error("Invalid relation_type provided")
				return fmt.Errorf("invalid relation_type '%s' for relationship '%s'; allowed values are: oneToOne, oneToMany, manyToMany", attribute.Relation, attribute.Name)
			}

			continue // relation is done, go to next attribute!
		}

		// --- Only call registry/type check for non-relation ---
		if _, err := types.GetGraphQLType(attribute.Type); err != nil {
			logger.Log.WithField("attribute", attribute.Name).WithError(err).Error("Invalid attribute type")
			return fmt.Errorf("invalid type '%s' for attribute '%s'", attribute.Type, attribute.Name)
		}
	}

	logger.Log.WithField("collection", ct.Name).Info("Collection schema validated successfully")
	return nil
}

//
//
// -------------------- Schema validator helpers
//

// validateField checks the attribute schema for constraints and valid types.
func validateAttributeType(attribute Attribute) error {
	validTypes := map[string]struct{}{
		"text":     {},
		"int":      {},
		"bool":     {},
		"date":     {},
		"richtext": {},
		"enum":     {},
		"relation": {},
	}

	if _, valid := validTypes[attribute.Type]; !valid {
		return fmt.Errorf("invalid attribute type '%s' for attribute '%s'", attribute.Type, attribute.Name)
	}

	if attribute.Type == "enum" && len(attribute.Options) == 0 {
		return fmt.Errorf("attribute '%s' of type 'enum' must have options", attribute.Name)
	}

	if attribute.Type == "int" && attribute.Min != nil && attribute.Max != nil && *attribute.Min > *attribute.Max {
		return fmt.Errorf("attribute '%s': min value cannot be greater than max value", attribute.Name)
	}

	if attribute.Type == "text" && attribute.Pattern != "" {
		if _, err := regexp.Compile(attribute.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern for attribute '%s': %v", attribute.Name, err)
		}
	}

	return nil
}

// GetFieldType returns whether a given attributeName is a "attribute", "relationship", or unknown.
func (c *Collection) GetAttributeType(attributeName string) (string, error) {
	for _, attribute := range c.Attributes {
		if attribute.Name == attributeName {
			return "attribute", nil
		}
	}

	logger.Log.WithField("attribute", attributeName).Warn("Unknown attribute or relationship")
	return "", fmt.Errorf("unknown attribute or relationship: '%s'", attributeName)
}

// ToFlattenedMap converts the Collection into a flat map structure
func (c *Collection) ToFlattenedMap() map[string]interface{} {
	flattened := map[string]interface{}{
		"id":   c.ID,
		"name": c.Name,
	}

	// Flatten attributes
	for _, attr := range c.Attributes {
		flattened[attr.Name] = map[string]interface{}{
			"ID":   attr.ID,
			"name": attr.Name,
			"type": attr.Type,
		}
	}

	return flattened
}

// validateAttributeValue handles validation logic for a single attribute’s value.
func validateAttributeValue(attribute Attribute, value interface{}) error {
	// 1) Confirm the attribute type is recognized in the registry
	if _, err := types.GetGraphQLType(attribute.Type); err != nil {
		// e.g., if "relation" or unregistered type => returns an error
		return fmt.Errorf("unsupported attribute type '%s': %w", attribute.Type, err)
	}

	// 2) Proceed with domain-specific validation logic:
	switch attribute.Type {
	case "string", "text", "richtext":
		strValue, err := convertToType(value, "text")
		if err != nil {
			return err
		}
		// Pattern match if specified
		if attribute.Pattern != "" {
			matched, err := regexp.MatchString(attribute.Pattern, strValue.(string))
			if err != nil {
				return fmt.Errorf("invalid regex pattern for attribute '%s': %v", attribute.Name, err)
			}
			if !matched {
				return fmt.Errorf("attribute '%s' does not match required pattern", attribute.Name)
			}
		}

	case "int":
		intValue, err := convertToType(value, "int")
		if err != nil {
			return err
		}
		iv := intValue.(int)
		if attribute.Min != nil && iv < *attribute.Min {
			return fmt.Errorf("attribute '%s' must be at least %d", attribute.Name, *attribute.Min)
		}
		if attribute.Max != nil && iv > *attribute.Max {
			return fmt.Errorf("attribute '%s' must be at most %d", attribute.Name, *attribute.Max)
		}

	case "bool":
		if _, err := convertToType(value, "bool"); err != nil {
			return err
		}

	case "time", "date":
		if _, err := convertToType(value, "date"); err != nil {
			return err
		}

	case "enum":
		strValue, err := convertToType(value, "text")
		if err != nil {
			return err
		}
		if !sliceContains(attribute.Options, strValue.(string)) {
			return fmt.Errorf("attribute '%s' must be one of %v", attribute.Name, attribute.Options)
		}

	default:
		// If, for example, "relation" or "component" is encountered, we skip or return an error
		return fmt.Errorf("unsupported or special attribute type '%s'; must be handled separately", attribute.Type)
	}

	logger.Log.WithField("attribute", attribute.Name).Info("attribute validated successfully")
	return nil
}

// convertToType attempts to convert 'value' to the desired 'targetType'.
func convertToType(value interface{}, targetType string) (interface{}, error) {
	switch targetType {
	case "text":
		if str, ok := value.(string); ok {
			return str, nil
		}
		// Fallback: format with %v
		return fmt.Sprintf("%v", value), nil

	case "int":
		switch v := value.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		case string:
			if intValue, err := strconv.Atoi(v); err == nil {
				return intValue, nil
			}
		}
		return nil, fmt.Errorf("invalid number format for value: %v", value)

	case "float":
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case string:
			if floatValue, err := strconv.ParseFloat(v, 64); err == nil {
				return floatValue, nil
			}
		}
		return nil, fmt.Errorf("invalid float format for value: %v", value)

	case "bool":
		if boolVal, ok := value.(bool); ok {
			return boolVal, nil
		}
		if str, ok := value.(string); ok {
			switch str {
			case "true":
				return true, nil
			case "false":
				return false, nil
			}
		}
		return nil, fmt.Errorf("invalid boolean format for value: %v", value)

	case "date":
		// Adjust the format to match your needs: "2006-01-02" is YYYY-MM-DD
		if str, ok := value.(string); ok {
			if dateValue, err := time.Parse("2006-01-02", str); err == nil {
				return dateValue, nil
			}
		}
		return nil, fmt.Errorf("invalid date format for value: %v", value)

	default:
		return nil, fmt.Errorf("unsupported target type: %s", targetType)
	}
}

// sliceContains checks if 'item' exists in 'slice'.
func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
