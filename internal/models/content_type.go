package models

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"gorm.io/gorm"
)

type Relationship struct {
	gorm.Model
	FieldName     string `json:"field_name"`
	RelatedType   string `json:"related_type"`
	RelationType  string `json:"relation_type"`
	ContentTypeID uint   `json:"-"` // Foreign key to associate with ContentType
}

type Field struct {
	gorm.Model
	Name          string            `json:"name"`
	Type          string            `json:"type"` // e.g., "string", "int", "bool", "date", "richtext", "enum"
	Required      bool              `json:"required"`
	Options       []string          `gorm:"type:json" json:"options,omitempty"`
	Min           *int              `json:"min,omitempty"`
	Max           *int              `json:"max,omitempty"`
	Pattern       string            `json:"pattern,omitempty"`
	CustomErrors  map[string]string `gorm:"type:json" json:"custom_errors,omitempty"`
	ContentTypeID uint              `json:"-"` // Foreign key to associate with ContentType
}

type ContentType struct {
	gorm.Model
	Name          string         `json:"name" gorm:"uniqueIndex"`
	Fields        []Field        `json:"fields" gorm:"constraint:OnDelete:CASCADE;"`
	Relationships []Relationship `json:"relationships" gorm:"constraint:OnDelete:CASCADE;"`
}

// ValidateItemData validates the item data against the content type's schema.
func ValidateItemData(ct ContentType, data map[string]interface{}) error {
	for _, field := range ct.Fields {
		value, exists := data[field.Name]

		// Check if field is required
		if field.Required && !exists {
			return fmt.Errorf("missing required field: '%s'", field.Name)
		}

		// Skip validation for non-required fields that are not present
		if !exists {
			continue
		}

		// Validate based on field type
		if err := validateFieldValue(field, value); err != nil {
			return err
		}
	}

	return nil
}

// ValidateContentType validates the schema of a content type.
func ValidateContentType(ct ContentType) error {
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
		if rel.FieldName == "" || rel.RelatedType == "" || rel.RelationType == "" {
			return fmt.Errorf("invalid relationship: all fields must be defined (fieldName, relatedType, relationType)")
		}
	}

	return nil
}

// validateFieldValue checks the value of a field against its type and constraints.
func validateFieldValue(field Field, value interface{}) error {
	switch field.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a string", field.Name)
		}
		if field.Pattern != "" {
			if matched, err := regexp.MatchString(field.Pattern, str); err != nil || !matched {
				return fmt.Errorf("field '%s' does not match required pattern", field.Name)
			}
		}
	case "int":
		num, ok := value.(float64) // JSON numbers are parsed as float64
		if !ok {
			return fmt.Errorf("field '%s' must be a number", field.Name)
		}
		intValue := int(num)
		if field.Min != nil && intValue < *field.Min {
			return fmt.Errorf("field '%s' must be at least %d", field.Name, *field.Min)
		}
		if field.Max != nil && intValue > *field.Max {
			return fmt.Errorf("field '%s' must be at most %d", field.Name, *field.Max)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean", field.Name)
		}
	case "date":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a string in date format", field.Name)
		}
		if _, err := time.Parse("2006-01-02", str); err != nil {
			return fmt.Errorf("field '%s' must be a valid date (YYYY-MM-DD)", field.Name)
		}
	case "enum":
		str, ok := value.(string)
		if !ok || !containsMap(field.Options, str) {
			return fmt.Errorf("field '%s' must be one of %v", field.Name, field.Options)
		}
	case "richtext":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string", field.Name)
		}
	default:
		return fmt.Errorf("unsupported field type: '%s'", field.Type)
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
