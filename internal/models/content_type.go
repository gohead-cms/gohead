// internal/models/content_type.go
package models

import (
	"fmt"
	"regexp"
	"time"
)

type Relationship struct {
	FieldName    string `json:"field_name"`
	RelatedType  string `json:"related_type"`
	RelationType string `json:"relation_type"` // "one-to-one", "one-to-many", "many-to-many"
}

type ContentType struct {
	Name          string         `json:"name"`
	Fields        []Field        `json:"fields"`
	Relationships []Relationship `json:"relationships"`
}

type Field struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"` // e.g., "string", "int", "bool", "date", "richtext", "enum"
	Required     bool              `json:"required"`
	Options      []string          `json:"options,omitempty"`       // For enums
	Min          *int              `json:"min,omitempty"`           // For numeric fields
	Max          *int              `json:"max,omitempty"`           // For numeric fields
	Pattern      string            `json:"pattern,omitempty"`       // For regex validation
	CustomErrors map[string]string `json:"custom_errors,omitempty"` // Custom error messages
}

// ValidateItemData validates the item data against the content type's schema
func ValidateItemData(ct ContentType, data map[string]interface{}) error {
	for _, field := range ct.Fields {
		value, exists := data[field.Name]

		// Check if field is required
		if field.Required && !exists {
			return fmt.Errorf("missing required field: %s", field.Name)
		}

		if !exists {
			continue // Skip validation for non-required fields that are not present
		}

		// Validate based on field type
		switch field.Type {
		case "string":
			strValue, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' must be a string", field.Name)
			}
			if field.Pattern != "" {
				matched, err := regexp.MatchString(field.Pattern, strValue)
				if err != nil {
					return fmt.Errorf("invalid regex pattern for field '%s': %v", field.Name, err)
				}
				if !matched {
					return fmt.Errorf("field '%s' does not match required pattern", field.Name)
				}
			}
		case "int":
			numValue, ok := value.(float64) // JSON numbers are float64
			if !ok {
				return fmt.Errorf("field '%s' must be a number", field.Name)
			}
			intValue := int(numValue)
			if field.Min != nil && intValue < *field.Min {
				return fmt.Errorf("field '%s' must be at least %d", field.Name, *field.Min)
			}
			if field.Max != nil && intValue > *field.Max {
				return fmt.Errorf("field '%s' must be at most %d", field.Name, *field.Max)
			}
		case "bool":
			_, ok := value.(bool)
			if !ok {
				return fmt.Errorf("field '%s' must be a boolean", field.Name)
			}
		case "date":
			strValue, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' must be a string in date format", field.Name)
			}
			// Validate date format, e.g., YYYY-MM-DD
			if _, err := time.Parse("2006-01-02", strValue); err != nil {
				return fmt.Errorf("field '%s' must be a valid date (YYYY-MM-DD)", field.Name)
			}
		case "enum":
			strValue, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' must be a string", field.Name)
			}
			if !contains(field.Options, strValue) {
				return fmt.Errorf("field '%s' must be one of %v", field.Name, field.Options)
			}
		case "richtext":
			// Assuming richtext is stored as a string (e.g., HTML or Markdown)
			_, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' must be a string", field.Name)
			}
			// Additional validation can be added here
		default:
			return fmt.Errorf("unsupported field type: %s", field.Type)
		}
	}

	return nil
}

func ValidateContentType(ct ContentType) error {
	fieldNames := make(map[string]bool)
	for _, field := range ct.Fields {
		// Check for duplicate field names
		if _, exists := fieldNames[field.Name]; exists {
			return fmt.Errorf("duplicate field name: %s", field.Name)
		}
		fieldNames[field.Name] = true

		// Validate field type
		validTypes := []string{"string", "int", "bool", "date", "richtext", "enum"}
		if !contains(validTypes, field.Type) {
			return fmt.Errorf("invalid field type '%s' for field '%s'", field.Type, field.Name)
		}

		// Additional validation per field type
		switch field.Type {
		case "enum":
			if len(field.Options) == 0 {
				return fmt.Errorf("field '%s' of type 'enum' must have options", field.Name)
			}
		case "string":
			if field.Pattern != "" {
				// Validate the regex pattern
				if _, err := regexp.Compile(field.Pattern); err != nil {
					return fmt.Errorf("invalid regex pattern for field '%s': %v", field.Name, err)
				}
			}
			// Add more cases as needed
		}
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
