package models

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
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

// ValidateItemData validates the item data against the content type's schema.
func ValidateItemData(ct Collection, data map[string]interface{}) error {
	for _, field := range ct.Fields {
		value, exists := data[field.Name]

		// Check if the field is required
		if field.Required && !exists {
			logger.Log.WithField("field", field.Name).Warn("Validation failed: missing required field")
			return fmt.Errorf("missing required field: '%s'", field.Name)
		}

		// Skip validation for non-required fields that are not present
		if !exists {
			continue
		}

		// Validate based on field type
		if err := validateFieldValue(field, value); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"field": field.Name,
				"type":  field.Type,
				"value": value,
			}).Warn("Validation failed for field")
			return fmt.Errorf("validation failed for field '%s': %w", field.Name, err)
		}
	}

	// Additional validation for unknown fields
	for key := range data {
		isValidField := false
		for _, field := range ct.Fields {
			if key == field.Name {
				isValidField = true
				break
			}
		}
		if !isValidField {
			logger.Log.WithField("field", key).Warn("Validation failed: unknown field")
			return fmt.Errorf("unknown field: '%s'", key)
		}
	}

	logger.Log.WithField("collection", ct.Name).Info("Item data validation passed")
	return nil
}

func validateFieldValue(field Field, value interface{}) error {
	switch field.Type {
	case "string":
		strValue, err := convertToType(value, "string")
		if err != nil {
			logger.Log.WithField("field", field.Name).WithError(err).Warn("String validation failed")
			return err
		}
		if field.Pattern != "" {
			if matched, err := regexp.MatchString(field.Pattern, strValue.(string)); err != nil || !matched {
				logger.Log.WithFields(logrus.Fields{
					"field":   field.Name,
					"pattern": field.Pattern,
				}).Warn("String pattern validation failed")
				return fmt.Errorf("field '%s' does not match required pattern", field.Name)
			}
		}

	case "int":
		intValue, err := convertToType(value, "int")
		if err != nil {
			logger.Log.WithField("field", field.Name).WithError(err).Warn("Integer validation failed")
			return err
		}
		if field.Min != nil && intValue.(int) < *field.Min {
			logger.Log.WithFields(logrus.Fields{
				"field": field.Name,
				"min":   *field.Min,
			}).Warn("Integer value below minimum")
			return fmt.Errorf("field '%s' must be at least %d", field.Name, *field.Min)
		}
		if field.Max != nil && intValue.(int) > *field.Max {
			logger.Log.WithFields(logrus.Fields{
				"field": field.Name,
				"max":   *field.Max,
			}).Warn("Integer value above maximum")
			return fmt.Errorf("field '%s' must be at most %d", field.Name, *field.Max)
		}

	case "bool":
		if _, err := convertToType(value, "bool"); err != nil {
			logger.Log.WithField("field", field.Name).WithError(err).Warn("Boolean validation failed")
			return err
		}

	case "date":
		if _, err := convertToType(value, "date"); err != nil {
			logger.Log.WithField("field", field.Name).WithError(err).Warn("Date validation failed")
			return err
		}

	case "enum":
		strValue, err := convertToType(value, "string")
		if err != nil {
			logger.Log.WithField("field", field.Name).WithError(err).Warn("Enum validation failed")
			return err
		}
		if !contains(field.Options, strValue.(string)) {
			logger.Log.WithFields(logrus.Fields{
				"field":    field.Name,
				"expected": field.Options,
			}).Warn("Enum value not in options")
			return fmt.Errorf("field '%s' must be one of %v", field.Name, field.Options)
		}

	default:
		logger.Log.WithField("field", field.Name).Error("Unsupported field type")
		return fmt.Errorf("unsupported field type: '%s'", field.Type)
	}
	logger.Log.WithField("field", field.Name).Info("Field validated successfully")
	return nil
}

// contains checks if a string exists in a slice.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// convertToType attempts to reliably determine and convert the input value to the desired type.
func convertToType(value interface{}, targetType string) (interface{}, error) {
	switch targetType {
	case "string":
		if str, ok := value.(string); ok {
			return str, nil
		}
		logger.Log.WithField("value", value).Info("Converting to string")
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
		logger.Log.WithField("value", value).Warn("Failed to convert to int")
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
		logger.Log.WithField("value", value).Warn("Failed to convert to float")
		return nil, fmt.Errorf("invalid float format for value: %v", value)

	case "bool":
		if boolValue, ok := value.(bool); ok {
			return boolValue, nil
		}
		if str, ok := value.(string); ok {
			if str == "true" {
				return true, nil
			} else if str == "false" {
				return false, nil
			}
		}
		logger.Log.WithField("value", value).Warn("Failed to convert to bool")
		return nil, fmt.Errorf("invalid boolean format for value: %v", value)

	case "date":
		if str, ok := value.(string); ok {
			if dateValue, err := time.Parse("2006-01-02", str); err == nil {
				return dateValue, nil
			}
		}
		logger.Log.WithField("value", value).Warn("Failed to convert to date")
		return nil, fmt.Errorf("invalid date format for value: %v", value)

	default:
		logger.Log.WithField("targetType", targetType).Error("Unsupported target type")
		return nil, fmt.Errorf("unsupported target type: %s", targetType)
	}
}

// ValidateCollection validates the schema of a content type.
func ValidateCollection(ct Collection) error {
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

		if err := validateField(field); err != nil {
			return err
		}
	}

	for _, rel := range ct.Relationships {
		if rel.FieldName == "" || rel.RelatedCollection == 0 || rel.RelationType == "" {
			return fmt.Errorf("invalid relationship: all fields must be defined (fieldName, relatedType, relationType)")
		}
	}

	return nil
}

// validateField checks the field schema for constraints and valid types.
func validateField(field Field) error {
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

// containsMap checks if a string exists in a slice using a map for O(1) lookups.
func containsMap(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	_, exists := set[item]
	return exists
}
