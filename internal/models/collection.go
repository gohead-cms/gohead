package models

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/gorm"
)

type Field struct {
	gorm.Model
	Name         string   `json:"name"`
	Type         string   `json:"type"` // e.g., "string", "int", "bool", "date", "richtext", "enum"
	Required     bool     `json:"required"`
	Options      []string `gorm:"type:json" json:"options,omitempty"`
	Min          *int     `json:"min,omitempty"`
	Max          *int     `json:"max,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	CustomErrors JSONMap  `gorm:"type:json" json:"custom_errors,omitempty"`
	CollectionID uint     `json:"-"` // Foreign key to associate with Collection
}

type Collection struct {
	gorm.Model
	Name          string         `json:"name" gorm:"uniqueIndex"`
	Fields        []Field        `json:"fields" gorm:"constraint:OnDelete:CASCADE;"`
	Relationships []Relationship `json:"relationships" gorm:"constraint:OnDelete:CASCADE;"`
}

//
//
// -------------------- Schema validators
//
//

// ValidateCollection validates the schema of a content type.
func ValidateCollectionSchema(ct Collection) error {
	if ct.Name == "" {
		return errors.New("missing required field: 'name'")
	}

	if len(ct.Fields) == 0 {
		return errors.New("fields array cannot be empty")
	}

	fieldNames := make(map[string]bool)
	for _, field := range ct.Fields {
		if fieldNames[field.Name] {
			return fmt.Errorf("duplicate field name: '%s'", field.Name)
		}
		fieldNames[field.Name] = true
		//logger.Log.WithField("field", field).Info("Validate field type")
		if err := validateFieldType(field); err != nil {
			return err
		}
	}

	for _, rel := range ct.Relationships {
		logger.Log.WithField("field", rel).Info("Test field")
		if rel.Field == "" || rel.RelationType == "" {
			return fmt.Errorf("invalid relationship: all fields must be defined (field, collection, relation_type)")
		}
	}

	return nil
}

// validateField checks the field schema for constraints and valid types.
func validateFieldType(field Field) error {
	validTypes := map[string]struct{}{
		"string":   {},
		"int":      {},
		"bool":     {},
		"date":     {},
		"richtext": {},
		"enum":     {},
	}

	if _, valid := validTypes[field.Type]; !valid {
		return fmt.Errorf("invalid field type '%s' for field '%s'", field.Type, field.Name)
	}

	if field.Type == "enum" && len(field.Options) == 0 {
		return fmt.Errorf("field '%s' of type 'enum' must have options", field.Name)
	}

	if field.Type == "int" && field.Min != nil && field.Max != nil && *field.Min > *field.Max {
		return fmt.Errorf("field '%s': min value cannot be greater than max value", field.Name)
	}

	if field.Type == "string" && field.Pattern != "" {
		if _, err := regexp.Compile(field.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern for field '%s': %v", field.Name, err)
		}
	}

	return nil
}

// GetFieldType returns whether a given fieldName is a "field", "relationship", or unknown.
func (c *Collection) GetFieldType(fieldName string) (string, error) {
	for _, field := range c.Fields {
		if field.Name == fieldName {
			return "field", nil
		}
	}
	for _, rel := range c.Relationships {
		if rel.Field == fieldName {
			return "relationship", nil
		}
	}
	logger.Log.WithField("field", fieldName).Warn("Unknown field or relationship")
	return "", fmt.Errorf("unknown field or relationship: '%s'", fieldName)
}

// validateFieldValue handles validation logic for a single field’s value.
func validateFieldValue(field Field, value interface{}) error {
	switch field.Type {
	case "string", "richtext":
		strValue, err := convertToType(value, "string")
		if err != nil {
			return err
		}
		// Pattern match if specified
		if field.Pattern != "" {
			matched, err := regexp.MatchString(field.Pattern, strValue.(string))
			if err != nil {
				return fmt.Errorf("invalid regex pattern for field '%s': %v", field.Name, err)
			}
			if !matched {
				return fmt.Errorf("field '%s' does not match required pattern", field.Name)
			}
		}

	case "int":
		intValue, err := convertToType(value, "int")
		if err != nil {
			return err
		}
		iv := intValue.(int)
		if field.Min != nil && iv < *field.Min {
			return fmt.Errorf("field '%s' must be at least %d", field.Name, *field.Min)
		}
		if field.Max != nil && iv > *field.Max {
			return fmt.Errorf("field '%s' must be at most %d", field.Name, *field.Max)
		}

	case "bool":
		if _, err := convertToType(value, "bool"); err != nil {
			return err
		}

	case "date":
		if _, err := convertToType(value, "date"); err != nil {
			return err
		}

	case "enum":
		strValue, err := convertToType(value, "string")
		if err != nil {
			return err
		}
		if !sliceContains(field.Options, strValue.(string)) {
			return fmt.Errorf("field '%s' must be one of %v", field.Name, field.Options)
		}

	default:
		return fmt.Errorf("unsupported field type: '%s'", field.Type)
	}

	logger.Log.WithField("field", field.Name).Info("Field validated successfully")
	return nil
}

// convertToType attempts to convert 'value' to the desired 'targetType'.
func convertToType(value interface{}, targetType string) (interface{}, error) {
	switch targetType {
	case "string":
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
			if str == "true" {
				return true, nil
			} else if str == "false" {
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
