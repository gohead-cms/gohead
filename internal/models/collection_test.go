package models

import (
	"bytes"
	"testing"

	"gohead/pkg/logger"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestValidateCollectionSchema(t *testing.T) {
	t.Run("Valid Collection", func(t *testing.T) {
		collection := Collection{
			Name: "articles",
			Attributes: []Attribute{
				{Name: "title", Type: "text", Required: true},
				{Name: "content", Type: "richtext", Required: true},
			},
		}
		assert.NoError(t, ValidateCollectionSchema(collection))
	})

	t.Run("Missing Name", func(t *testing.T) {
		collection := Collection{
			Attributes: []Attribute{
				{Name: "title", Type: "text", Required: true},
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
				{Name: "title", Type: "text", Required: true},
				{Name: "title", Type: "richtext", Required: true},
			},
		}
		err := ValidateCollectionSchema(collection)
		assert.Error(t, err)
		assert.Equal(t, "duplicate attribute name: 'title'", err.Error())
	})

	t.Run("Invalid Attribute Type", func(t *testing.T) {
		collection := Collection{
			Name: "articles",
			Attributes: []Attribute{
				{Name: "invalid", Type: "unknownType"},
			},
		}
		err := ValidateCollectionSchema(collection)
		assert.Error(t, err)
		assert.Equal(t, "invalid type 'unknownType' for attribute 'invalid'", err.Error())
	})
}

func TestValidateCollection(t *testing.T) {
	t.Run("Valid Collection", func(t *testing.T) {
		collection := Collection{
			Name: "articles",
			Attributes: []Attribute{
				{Name: "title", Type: "text", Required: true},
				{Name: "content", Type: "richtext", Required: true},
			},
		}
		assert.NoError(t, ValidateCollectionSchema(collection))
	})

	t.Run("Missing Name", func(t *testing.T) {
		collection := Collection{
			Attributes: []Attribute{
				{Name: "title", Type: "text", Required: true},
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
				{Name: "title", Type: "text", Required: true},
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
		{Name: "title", Type: "text", Required: true},
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
		assert.NoError(t, ValidateItemValues(collection, data))
	})

	t.Run("Missing Required Field", func(t *testing.T) {
		data := map[string]interface{}{
			"published_date": "2024-12-10",
		}
		err := ValidateItemValues(collection, data)
		assert.Error(t, err)
		assert.Equal(t, "missing required attribute: 'title'", err.Error())
	})

	t.Run("Invalid Date Format", func(t *testing.T) {
		data := map[string]interface{}{
			"title":          "An Article",
			"published_date": "12-10-2024",
		}
		err := ValidateItemValues(collection, data)
		assert.Error(t, err)
		assert.Equal(t, "validation failed for attribute 'published_date': invalid date format for value: 12-10-2024", err.Error())
	})

	t.Run("Out of Range Integer", func(t *testing.T) {
		data := map[string]interface{}{
			"title":  "An Article",
			"rating": 30,
		}
		err := ValidateItemValues(collection, data)
		assert.Error(t, err)
		assert.Equal(t, "validation failed for attribute 'rating': attribute 'rating' must be at most 20", err.Error())
	})
}
