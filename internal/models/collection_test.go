package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCollection(t *testing.T) {
	t.Run("Valid Collection", func(t *testing.T) {
		collection := Collection{
			Name: "articles",
			Attributes: []Attribute{
				{Name: "title", Type: "string", Required: true},
				{Name: "content", Type: "richtext", Required: true},
			},
		}
		assert.NoError(t, ValidateCollectionSchema(collection))
	})

	t.Run("Missing Name", func(t *testing.T) {
		collection := Collection{
			Attributes: []Attribute{
				{Name: "title", Type: "string", Required: true},
			},
		}
		err := ValidateCollectionSchema(collection)
		assert.Error(t, err)
		assert.Equal(t, "missing required attribute: 'name'", err.Error())
	})

	t.Run("Duplicate Field Names", func(t *testing.T) {
		collection := Collection{
			Name: "articles",
			Attributes: []Attribute{
				{Name: "title", Type: "string", Required: true},
				{Name: "title", Type: "richtext", Required: true},
			},
		}
		err := ValidateCollectionSchema(collection)
		assert.Error(t, err)
		assert.Equal(t, "duplicate attribute name: 'title'", err.Error())
	})
}
func intPtr(i int) *int {
	return &i
}

func TestValidateItemData(t *testing.T) {
	min := 1
	max := 20

	attributes := []Attribute{
		{Name: "title", Type: "string", Required: true},
		{Name: "published_date", Type: "date"},
		{Name: "rating", Type: "int", Min: &min, Max: &max},
	}
	rating := 18

	collection := Collection{
		Name:       "articles",
		Attributes: attributes,
	}

	t.Run("Valid Data", func(t *testing.T) {
		data := map[string]interface{}{
			"title":          "An Article",
			"published_date": "2024-12-10",
			"rating":         rating,
		}
		assert.NoError(t, ValidateItemData(collection, data))
	})

	t.Run("Missing Required Field", func(t *testing.T) {
		data := map[string]interface{}{
			"published_date": "2024-12-10",
		}
		err := ValidateItemData(collection, data)
		assert.Error(t, err)
		assert.Equal(t, "missing required attribute: 'title'", err.Error())
	})

	t.Run("Invalid Date Format", func(t *testing.T) {
		data := map[string]interface{}{
			"title":          "An Article",
			"published_date": "12-10-2024",
		}
		err := ValidateItemData(collection, data)
		assert.Error(t, err)
		assert.Equal(t, "validation failed for attribute 'published_date': invalid date format for value: 12-10-2024", err.Error())
	})

	t.Run("Out of Range Integer", func(t *testing.T) {
		data := map[string]interface{}{
			"title":  "An Article",
			"rating": 30,
		}
		err := ValidateItemData(collection, data)
		assert.Error(t, err)
		assert.Equal(t, "validation failed for attribute 'rating': attribute 'rating' must be at most 20", err.Error())
	})
}

func ptrInt(v int) *int {
	return &v
}
