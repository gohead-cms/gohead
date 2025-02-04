// internal/models/component_test.go
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseComponentInput(t *testing.T) {
	t.Run("Valid Input", func(t *testing.T) {
		input := map[string]interface{}{
			"name":        "seo",
			"description": "SEO-related fields",
			"attributes": map[string]interface{}{
				"title": map[string]interface{}{
					"type":     "text",
					"required": true,
				},
				"description": map[string]interface{}{
					"type": "richtext",
				},
			},
		}

		cmp, err := ParseComponentInput(input)
		assert.NoError(t, err, "parsing valid component input should succeed")
		assert.Equal(t, "seo", cmp.Name)
		assert.Equal(t, "SEO-related fields", cmp.Description)
		assert.Len(t, cmp.Attributes, 2)

		// Validate the resulting schema
		err = ValidateComponentSchema(cmp)
		assert.NoError(t, err, "validation should pass for a valid component schema")
	})

	t.Run("Missing Name", func(t *testing.T) {
		input := map[string]interface{}{
			"attributes": map[string]interface{}{
				"title": map[string]interface{}{
					"type": "text",
				},
			},
		}

		cmp, err := ParseComponentInput(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid field 'name'")
		assert.Empty(t, cmp.Name, "component name should be empty on error")
	})

	t.Run("Missing Attributes", func(t *testing.T) {
		input := map[string]interface{}{
			"name": "empty-component",
		}

		cmp, err := ParseComponentInput(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid field 'attributes'")
		assert.Equal(t, "empty-component", cmp.Name)
	})

	t.Run("Invalid Attribute Format", func(t *testing.T) {
		input := map[string]interface{}{
			"name": "bad-attrs",
			"attributes": map[string]interface{}{
				"title": "this should be a map, not a string",
			},
		}

		cmp, err := ParseComponentInput(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid attribute format for 'title'")
		assert.Equal(t, "bad-attrs", cmp.Name)
		assert.Empty(t, cmp.Attributes)
	})
}

func TestValidateComponentSchema(t *testing.T) {
	t.Run("Empty Attributes", func(t *testing.T) {
		cmp := Component{
			Name: "invalid-cmp",
			// No attributes
		}

		err := ValidateComponentSchema(cmp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one attribute")
	})

	t.Run("Duplicate Attributes", func(t *testing.T) {
		cmp := Component{
			Name: "duplicate-test",
			Attributes: []Attribute{
				{Name: "fieldA", Type: "text"},
				{Name: "fieldA", Type: "bool"}, // Duplicate name
			},
		}

		err := ValidateComponentSchema(cmp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate attribute name: 'fieldA'")
	})

	t.Run("Invalid Attribute Type", func(t *testing.T) {
		cmp := Component{
			Name: "invalid-attr-type",
			Attributes: []Attribute{
				{Name: "title", Type: "nonexistentType"},
			},
		}

		err := ValidateComponentSchema(cmp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid attribute 'title' in component 'invalid-attr-type'")
	})

	t.Run("Valid Schema", func(t *testing.T) {
		cmp := Component{
			Name: "valid-cmp",
			Attributes: []Attribute{
				{Name: "title", Type: "text", Required: true},
				{Name: "desc", Type: "richtext"},
			},
		}

		// Suppose validateAttributeType checks "text" and "richtext" as valid
		err := ValidateComponentSchema(cmp)
		assert.NoError(t, err)
	})
}
