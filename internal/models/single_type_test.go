package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSingleTypeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected SingleType
		hasError bool
	}{
		{
			name: "Valid input with attributes",
			input: map[string]interface{}{
				"name":        "homepage",
				"description": "A single type for the homepage content",
				"attributes": map[string]any{
					"title": map[string]any{
						"type":     "text",
						"required": true,
					},
					"description": map[string]interface{}{
						"type": "text",
					},
				},
			},
			expected: SingleType{
				Name:        "homepage",
				Description: "A single type for the homepage content",
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
					{Name: "description", Type: "text"},
				},
			},
			hasError: false,
		},
		{
			name: "Invalid input without attributes",
			input: map[string]any{
				"name":        "homepage",
				"description": "A single type for the homepage content",
			},
			expected: SingleType{},
			hasError: true,
		},
		{
			name: "Invalid attribute format",
			input: map[string]any{
				"name":        "homepage",
				"description": "A single type for the homepage content",
				"attributes": map[string]any{
					"title": "invalid_format",
				},
			},
			expected: SingleType{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := ParseSingleTypeInput(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Name, st.Name)
				assert.Equal(t, tt.expected.Description, st.Description)
				assert.Equal(t, len(tt.expected.Attributes), len(st.Attributes))
			}
		})
	}
}

func TestValidateSingleTypeSchema(t *testing.T) {
	tests := []struct {
		name     string
		schema   SingleType
		hasError bool
	}{
		{
			name: "Valid schema",
			schema: SingleType{
				Name: "homepage",
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
					{Name: "description", Type: "text"},
				},
			},
			hasError: false,
		},
		{
			name: "Missing name",
			schema: SingleType{
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
				},
			},
			hasError: true,
		},
		{
			name: "Duplicate attributes",
			schema: SingleType{
				Name: "homepage",
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
					{Name: "title", Type: ""},
				},
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSingleTypeSchema(tt.schema)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSingleTypeValues(t *testing.T) {
	tests := []struct {
		name     string
		schema   SingleType
		data     map[string]interface{}
		hasError bool
	}{
		{
			name: "Valid data",
			schema: SingleType{
				Name: "homepage",
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
					{Name: "description", Type: "text", Required: false},
				},
			},
			data: map[string]interface{}{
				"title":       "Welcome to the homepage",
				"description": "This is a test description",
			},
			hasError: false,
		},
		{
			name: "Missing required attribute",
			schema: SingleType{
				Name: "homepage",
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
				},
			},
			data:     map[string]any{},
			hasError: true,
		},
		{
			name: "Extra undefined attribute",
			schema: SingleType{
				Name: "homepage",
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
				},
			},
			data: map[string]interface{}{
				"title": "Welcome to the homepage",
				"extra": "Unexpected value",
			},
			hasError: true, // Schema does not restrict extra fields unless explicitly required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSingleTypeValues(tt.schema, tt.data)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
