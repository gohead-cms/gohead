package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseComponentInput(t *testing.T) {
	t.Run("Valid Input", func(t *testing.T) {
		input := map[string]any{
			"name":        "seo",
			"description": "SEO-related fields",
			"attributes": map[string]any{
				"title": map[string]any{
					"type":     "text",
					"required": true,
				},
				"description": map[string]any{
					"type": "richtext",
				},
			},
		}

		cmp, err := ParseComponentInput(input)
		assert.NoError(t, err, "parsing valid component input should succeed")
		assert.Equal(t, "seo", cmp.Name)
		assert.Equal(t, "SEO-related fields", cmp.Description)
		assert.Len(t, cmp.Attributes, 2)

		// Also check that a parsed component is valid
		err = ValidateComponentSchema(cmp)
		assert.NoError(t, err, "validation should pass for a valid component schema")
	})

	t.Run("Missing Name", func(t *testing.T) {
		input := map[string]any{
			"attributes": map[string]any{
				"title": map[string]any{
					"type": "text",
				},
			},
		}

		_, err := ParseComponentInput(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid field 'name'")
	})

	t.Run("Missing Attributes in input map", func(t *testing.T) {
		// Note: ParseComponentInput doesn't require attributes, but ValidateComponentSchema does.
		// This test ensures parsing succeeds without an 'attributes' map in the input.
		input := map[string]any{
			"name": "no-attrs-component",
		}

		cmp, err := ParseComponentInput(input)
		assert.NoError(t, err, "Parsing should succeed even without an 'attributes' key")
		assert.Equal(t, "no-attrs-component", cmp.Name)
		assert.Empty(t, cmp.Attributes)
	})

	t.Run("Invalid Attribute Format", func(t *testing.T) {
		input := map[string]any{
			"name": "bad-attrs",
			"attributes": map[string]any{
				"title": "this should be a map, not a string",
			},
		}

		_, err := ParseComponentInput(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid attribute format for 'title'")
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
			Attributes: []ComponentAttribute{
				{BaseAttribute: BaseAttribute{Name: "fieldA", Type: "text"}},
				{BaseAttribute: BaseAttribute{Name: "fieldA", Type: "bool"}}, // Duplicate name
			},
		}

		err := ValidateComponentSchema(cmp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate attribute name: 'fieldA'")
	})

	t.Run("Invalid Attribute Type", func(t *testing.T) {
		cmp := Component{
			Name: "invalid-attr-type",
			Attributes: []ComponentAttribute{
				{BaseAttribute: BaseAttribute{Name: "title", Type: "nonexistentType"}},
			},
		}

		err := ValidateComponentSchema(cmp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid attribute 'title' in component 'invalid-attr-type'")
	})

	t.Run("Valid Schema", func(t *testing.T) {
		// This test is now corrected to use ComponentAttribute and BaseAttribute
		cmp := Component{
			Name: "valid-cmp",
			Attributes: []ComponentAttribute{
				{BaseAttribute: BaseAttribute{Name: "title", Type: "text", Required: true}},
				{BaseAttribute: BaseAttribute{Name: "desc", Type: "richtext"}},
			},
		}

		// validateAttributeType will check "text" and "richtext" against the type registry.
		err := ValidateComponentSchema(cmp)
		assert.NoError(t, err)
	})
}
