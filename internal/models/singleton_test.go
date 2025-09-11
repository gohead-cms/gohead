package models

import (
	"bytes"
	"testing"

	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Initialize logger for testing
func init() {
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestParseSingletonInput(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected Singleton
		hasError bool
	}{
		{
			name: "Valid input with attributes",
			input: map[string]any{
				"name":        "homepage",
				"description": "A single type for the homepage content",
				"attributes": map[string]any{
					"title": map[string]any{
						"type":     "text",
						"required": true,
					},
					"description": map[string]any{
						"type": "text",
					},
				},
			},
			expected: Singleton{
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
			expected: Singleton{},
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
			expected: Singleton{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := ParseSingletonInput(tt.input)
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

func TestValidateSingletonSchema(t *testing.T) {
	tests := []struct {
		name     string
		schema   Singleton
		hasError bool
	}{
		{
			name: "Valid schema",
			schema: Singleton{
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
			schema: Singleton{
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
				},
			},
			hasError: true,
		},
		{
			name: "Duplicate attributes",
			schema: Singleton{
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
			err := ValidateSingletonSchema(tt.schema)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSingletonValues(t *testing.T) {
	tests := []struct {
		name     string
		schema   Singleton
		data     map[string]any
		hasError bool
	}{
		{
			name: "Valid data",
			schema: Singleton{
				Name: "homepage",
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
					{Name: "description", Type: "text", Required: false},
				},
			},
			data: map[string]any{
				"title":       "Welcome to the homepage",
				"description": "This is a test description",
			},
			hasError: false,
		},
		{
			name: "Missing required attribute",
			schema: Singleton{
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
			schema: Singleton{
				Name: "homepage",
				Attributes: []Attribute{
					{Name: "title", Type: "text", Required: true},
				},
			},
			data: map[string]any{
				"title": "Welcome to the homepage",
				"extra": "Unexpected value",
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSingletonValues(tt.schema, tt.data)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
