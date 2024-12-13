package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateItem(t *testing.T) {
	// Sample collection schema
	collection := Collection{
		Name: "articles",
		Fields: []Field{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "string", Required: true},
			{Name: "published", Type: "bool", Required: false},
		},
	}

	t.Run("Valid Item", func(t *testing.T) {
		item := Item{
			CollectionID: 1,
			Data: JSONMap{
				"title":     "Test Article",
				"content":   "This is a test article.",
				"published": true,
			},
		}

		err := ValidateItem(item, collection)
		assert.NoError(t, err)
	})

	t.Run("Missing CollectionID", func(t *testing.T) {
		item := Item{
			CollectionID: 0,
			Data: JSONMap{
				"title":   "Test Article",
				"content": "This is a test article.",
			},
		}

		err := ValidateItem(item, collection)
		assert.EqualError(t, err, "CollectionID is required")
	})

	t.Run("Empty Data", func(t *testing.T) {
		item := Item{
			CollectionID: 1,
			Data:         JSONMap{},
		}

		err := ValidateItem(item, collection)
		assert.EqualError(t, err, "Data cannot be empty")
	})

	t.Run("Missing Required Field", func(t *testing.T) {
		item := Item{
			CollectionID: 1,
			Data: JSONMap{
				"content": "This is a test article.",
			},
		}

		err := ValidateItem(item, collection)
		assert.EqualError(t, err, "missing required field: 'title'")
	})

	t.Run("Invalid Field Type", func(t *testing.T) {
		item := Item{
			CollectionID: 1,
			Data: JSONMap{
				"title":     "Test Article",
				"content":   "This is a test article.",
				"published": "not-a-boolean",
			},
		}

		err := ValidateItem(item, collection)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid boolean format for value: not-a-boolean")
	})
}
