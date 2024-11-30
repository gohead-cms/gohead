// internal/models/content_type_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateItemData(t *testing.T) {
	// Define the content type for testing
	ct := ContentType{
		Name: "users",
		Fields: []Field{
			{
				Name:     "username",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "age",
				Type:     "int",
				Required: true,
				Min:      intPtr(18),
				Max:      intPtr(99),
			},
		},
	}

	// Define test cases
	testCases := []struct {
		name     string
		data     map[string]interface{}
		expected string // Expected error message, empty if no error
	}{
		{
			name: "Valid Data",
			data: map[string]interface{}{
				"username": "john_doe",
				"age":      28,
			},
			expected: "",
		},
		{
			name: "Missing Required Field",
			data: map[string]interface{}{
				"age": 28,
			},
			expected: "missing required field: username",
		},
		{
			name: "Invalid Age (Too Young)",
			data: map[string]interface{}{
				"username": "john_doe",
				"age":      16,
			},
			expected: "field 'age' must be at least 18",
		},
		{
			name: "Invalid Age (Too Old)",
			data: map[string]interface{}{
				"username": "john_doe",
				"age":      120,
			},
			expected: "field 'age' must be at most 99",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateItemData(ct, tc.data)
			if tc.expected == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expected)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
