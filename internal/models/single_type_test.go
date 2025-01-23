package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/internal/models"
)

func TestParseSingleTypeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected models.SingleType
		hasError bool
	}{
		{
			name: "Valid input with attributes",
			input: map[string]interface{}{
				"name":        "homepage",
				"description": "A single type for the homepage content",
				"attributes": map[string]interface{}{
					"title": map[string]interface{}{
						"type":     "text",
						"required": true,
					},
					"description": map[string]interface{}{
						"type": "text",
					},
				},
			},
			expected: models.SingleType{
				Name:        "homepage",
				Description: "A single type for the homepage content",
				Attributes: []models.Attribute{
					{Name: "title", Type: "text", Required: true},
					{Name: "description", Type: "text"},
				},
			},
			hasError: false,
		},
		{
			name: "Invalid input without attributes",
			input: map[string]interface{}{
				"name":        "homepage",
				"description": "A single type for the homepage content",
			},
			expected: models.SingleType{},
			hasError: true,
		},
		{
			name: "Invalid attribute format",
			input: map[string]interface{}{
				"name":        "homepage",
				"description": "A single type for the homepage content",
				"attributes": map[string]interface{}{
					"title": "invalid_format",
				},
			},
			expected: models.SingleType{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := models.ParseSingleTypeInput(tt.input)
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
		schema   models.SingleType
		hasError bool
	}{
		{
			name: "Valid schema",
			schema: models.SingleType{
				Name: "homepage",
				Attributes: []models.Attribute{
					{Name: "title", Type: "text", Required: true},
					{Name: "description", Type: "text"},
				},
			},
			hasError: false,
		},
		{
			name: "Missing name",
			schema: models.SingleType{
				Attributes: []models.Attribute{
					{Name: "title", Type: "text", Required: true},
				},
			},
			hasError: true,
		},
		{
			name: "Duplicate attributes",
			schema: models.SingleType{
				Name: "homepage",
				Attributes: []models.Attribute{
					{Name: "title", Type: "text", Required: true},
					{Name: "title", Type: ""},
				},
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateSingleTypeSchema(tt.schema)
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
		schema   models.SingleType
		data     map[string]interface{}
		hasError bool
	}{
		{
			name: "Valid data",
			schema: models.SingleType{
				Name: "homepage",
				Attributes: []models.Attribute{
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
			schema: models.SingleType{
				Name: "homepage",
				Attributes: []models.Attribute{
					{Name: "title", Type: "text", Required: true},
				},
			},
			data:     map[string]interface{}{},
			hasError: true,
		},
		{
			name: "Extra undefined attribute",
			schema: models.SingleType{
				Name: "homepage",
				Attributes: []models.Attribute{
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
			err := models.ValidateSingleTypeValues(tt.schema, tt.data)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
