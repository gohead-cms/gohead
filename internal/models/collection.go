package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
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
	if rawAttributes, ok := input["attributes"].(map[string]any); ok {
		for attrName, rawAttr := range rawAttributes {
			attrMap, ok := rawAttr.(map[string]any)
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

func mapToAttribute(attrMap map[string]any, attribute *Attribute) error {
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

	if options, ok := attrMap["options"].([]any); ok {
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

// -------------------- Schema validator --------------------

// ValidateCollectionSchema is the single source of truth for validating a collection's structure.
// It correctly uses the TypeRegistry for all type checks.
func ValidateCollectionSchema(ct Collection) error {
	if ct.Name == "" {
		return errors.New("missing required attribute: 'name'")
	}
	if len(ct.Attributes) == 0 {
		return errors.New("attributes array cannot be empty")
	}

	attributeNames := make(map[string]bool)
	for _, attribute := range ct.Attributes {
		if attributeNames[attribute.Name] {
			return fmt.Errorf("duplicate attribute name: '%s'", attribute.Name)
		}
		attributeNames[attribute.Name] = true

		// --- Handle the "relation" type as a special case FIRST ---
		if attribute.Type == "relation" {
			if attribute.Relation == "" || attribute.Target == "" {
				return fmt.Errorf("relationship '%s' must define 'relation' and 'target'", attribute.Name)
			}
			var relatedCollection Collection
			if err := database.DB.Where("name = ?", attribute.Target).First(&relatedCollection).Error; err != nil {
				return fmt.Errorf("target collection '%s' for relationship '%s' does not exist", attribute.Target, attribute.Name)
			}
			allowedRelationTypes := map[string]struct{}{"oneToOne": {}, "oneToMany": {}, "manyToMany": {}}
			if _, isValid := allowedRelationTypes[attribute.Relation]; !isValid {
				return fmt.Errorf("invalid relation_type '%s' for relationship '%s'", attribute.Relation, attribute.Name)
			}
			continue // Relation is valid, move to the next attribute.
		}

		// --- For all other types, use the centralized TypeRegistry ---
		if _, err := types.GetGraphQLType(attribute.Type); err != nil {
			return fmt.Errorf("invalid type '%s' for attribute '%s': %w", attribute.Type, attribute.Name, err)
		}
	}

	logger.Log.WithField("collection", ct.Name).Info("Collection schema validated successfully")
	return nil
}

// -------------------- Schema validator helpers --------------------

// GetAttributeType correctly identifies an attribute as a "attribute" or "relationship".
func (c *Collection) GetAttributeType(attributeName string) (string, error) {
	for _, attribute := range c.Attributes {
		if attribute.Name == attributeName {
			// FIX: Check the attribute's type to determine its kind.
			if attribute.Type == "relation" {
				return "relationship", nil
			}
			return "attribute", nil
		}
	}

	logger.Log.WithField("attribute", attributeName).Warn("Unknown attribute or relationship")
	return "", fmt.Errorf("unknown attribute or relationship: '%s'", attributeName)
}

// ToFlattenedMap converts the Collection into a flat map structure
func (c *Collection) ToFlattenedMap() map[string]any {
	flattened := map[string]any{
		"id":   c.ID,
		"name": c.Name,
	}

	// Flatten attributes
	for _, attr := range c.Attributes {
		flattened[attr.Name] = map[string]any{
			"ID":   attr.ID,
			"name": attr.Name,
			"type": attr.Type,
		}
	}

	return flattened
}

// validateAttributeValue handles validation logic for a single attribute’s value.
func validateAttributeValue(attribute Attribute, value any) error {
	// 1) Confirm the attribute type is recognized in the registry
	if _, err := types.GetGraphQLType(attribute.Type); err != nil {
		// e.g., if "relation" or unregistered type => returns an error
		return fmt.Errorf("unsupported attribute type '%s': %w", attribute.Type, err)
	}

	// 2) Proceed with domain-specific validation logic:
	switch attribute.Type {
	case "string", "text", "richtext", "email":
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

	case "boolean", "bool":
		if _, err := convertToType(value, "bool"); err != nil {
			return err
		}

	case "time", "date":
		if _, err := convertToType(value, "date"); err != nil {
			return err
		}
	case "datetime":
		if _, err := convertToType(value, "datetime"); err != nil {
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
	case "uid":
		if _, err := convertToType(value, "text"); err != nil {
			return err
		}
	case "media":
		if _, err := convertToType(value, "text"); err != nil {
			return err
		}
	case "json":
		if !isValidJSON(value) {
			return fmt.Errorf("attribute '%s' must be a valid JSON object", attribute.Name)
		}
	default:
		// If, for example, "relation" or "component" is encountered, we skip or return an error
		return fmt.Errorf("unsupported or special attribute type '%s'; must be handled separately", attribute.Type)
	}

	logger.Log.WithField("attribute", attribute.Name).Info("attribute validated successfully")
	return nil
}

// convertToType attempts to convert 'value' to the desired 'targetType'.
func convertToType(value any, targetType string) (any, error) {
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
	case "datetime":
		if str, ok := value.(string); ok {
			if dateTimeValue, err := time.Parse(time.RFC3339, str); err == nil {
				return dateTimeValue, nil
			}
		}
		return nil, fmt.Errorf("invalid datetime format for value: %v (expected ISO 8601 format)", value)

	default:
		return nil, fmt.Errorf("unsupported target type: %s", targetType)
	}
}

// sliceContains checks if 'item' exists in 'slice'.
func sliceContains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

func isValidJSON(value any) bool {
	// If it's a string, we check if the string itself is valid JSON.
	if str, ok := value.(string); ok {
		return json.Valid([]byte(str))
	}

	// If it's another type (like map[string]any or []any from the request body),
	// we try to marshal it. If there's no error, it's a valid JSON structure.
	_, err := json.Marshal(value)
	return err == nil
}
