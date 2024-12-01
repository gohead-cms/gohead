// internal/models/json_map_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONMap_Value(t *testing.T) {
	tests := []struct {
		name     string
		jsonMap  JSONMap
		expected string
	}{
		{
			name: "Valid JSONMap",
			jsonMap: JSONMap{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
			expected: `{"key1":"value1","key2":123,"key3":true}`,
		},
		{
			name:     "Empty JSONMap",
			jsonMap:  JSONMap{},
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.jsonMap.Value()
			assert.NoError(t, err)

			jsonString, ok := value.([]byte)
			assert.True(t, ok)

			assert.JSONEq(t, tt.expected, string(jsonString))
		})
	}
}

func TestJSONMap_Scan(t *testing.T) {
	tests := []struct {
		name          string
		input         interface{}
		expectedJSON  JSONMap
		expectedError bool
	}{
		{
			name: "Valid JSON String",
			input: []byte(`{
				"key1": "value1",
				"key2": 123,
				"key3": true
			}`),
			expectedJSON: JSONMap{
				"key1": "value1",
				"key2": int(123),
				"key3": true,
			},
			expectedError: false,
		},
		{
			name:          "Nil Input",
			input:         nil,
			expectedJSON:  JSONMap{},
			expectedError: false,
		},
		{
			name:          "Invalid Input Type",
			input:         12345, // Invalid type
			expectedJSON:  nil,
			expectedError: true,
		},
		{
			name:          "Malformed JSON String",
			input:         []byte(`{"key1": "value1", "key2":`), // Malformed JSON
			expectedJSON:  nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonMap JSONMap
			err := jsonMap.Scan(tt.input)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedJSON, jsonMap)
			}
		})
	}
}

func TestJSONMap_ValueAndScanIntegration(t *testing.T) {
	original := JSONMap{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	// Convert JSONMap to driver.Value
	value, err := original.Value()
	assert.NoError(t, err)

	// Convert driver.Value back to JSONMap
	var scanned JSONMap
	err = scanned.Scan(value)
	assert.NoError(t, err)

	// Verify the original and scanned JSONMap are identical
	assert.Equal(t, original, scanned)
}
