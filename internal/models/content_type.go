// internal/models/content_type.go
package models

import "fmt"

type ContentType struct {
	Name   string            `json:"name"`
	Fields map[string]string `json:"fields"` // FieldName: FieldType
}

// ValidateItemData validates the item data against the content type's schema
func ValidateItemData(ct ContentType, data map[string]interface{}) error {
	for fieldName, fieldType := range ct.Fields {
		value, exists := data[fieldName]
		if !exists {
			return fmt.Errorf("missing field: %s", fieldName)
		}

		switch fieldType {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("field '%s' must be a string", fieldName)
			}
		case "bool":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("field '%s' must be a boolean", fieldName)
			}
		case "int":
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("field '%s' must be a number", fieldName)
			}
		// Add more types as needed
		default:
			return fmt.Errorf("unsupported field type: %s", fieldType)
		}
	}
	return nil
}
