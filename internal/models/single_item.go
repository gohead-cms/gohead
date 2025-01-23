package models

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/gorm"
)

// SingleItem holds the actual content for a SingleType.
type SingleItem struct {
	gorm.Model
	SingleTypeID uint       `json:"single_type_id" gorm:"uniqueIndex"`
	SingleType   SingleType `json:"-" gorm:"constraint:OnDelete:CASCADE;"`

	Data JSONMap `json:"data" gorm:"type:json"`
}

// ValidateSingleItemValues validates a single item's data against the SingleType's schema (attributes).
func ValidateSingleItemValues(st SingleType, itemData map[string]interface{}) error {
	// 1. Build a set of valid attribute names
	validAttributes := make(map[string]Attribute, len(st.Attributes))
	for _, attr := range st.Attributes {
		validAttributes[attr.Name] = attr
	}

	// 2. Check for unknown fields in 'data'
	for key := range itemData {
		if _, ok := validAttributes[key]; !ok {
			logger.Log.WithField("attribute", key).Warn("Validation failed: unknown attribute")
			return fmt.Errorf("unknown attribute: '%s'", key)
		}
	}
	for _, attribute := range st.Attributes {
		value, exists := itemData[attribute.Name]

		// Check for required attributes
		if attribute.Required && !exists {
			logger.Log.WithField("attribute", attribute.Name).
				Warn("Validation failed: missing required attribute")
			return fmt.Errorf("missing required attribute: '%s'", attribute.Name)
		}
		if !exists {
			continue // if not required and missing, skip
		}

		// Validate regular attributes or relationships
		if attribute.Type == "relation" {
			if err := validateSingleItemRelationship(attribute, value); err != nil {
				logger.Log.WithField("attribute", attribute.Name).
					Warn("Validation failed for relationship")
				return fmt.Errorf("validation failed for relationship '%s': %w", attribute.Name, err)
			}
		} else {
			if err := validateAttributeValue(attribute, value); err != nil {
				logger.Log.WithFields(logrus.Fields{
					"attribute": attribute.Name,
					"type":      attribute.Type,
					"value":     value,
				}).Warn("Validation failed for attribute")
				return fmt.Errorf("validation failed for attribute '%s': %w", attribute.Name, err)
			}
		}
	}

	logger.Log.WithField("single_type", st.Name).
		Info("Single item data validation passed")
	return nil
}

// validateSingleItemRelationship checks the validity of a relationship field within a SingleItem's Data.
// This might reference a normal collection or other single-type content, depending on your design.
func validateSingleItemRelationship(attribute Attribute, value interface{}) error {
	// 1. Check if attribute.Target is specified
	if attribute.Target == "" {
		logger.Log.WithField("attribute", attribute.Name).
			Warn("Missing target for single item relationship")
		return fmt.Errorf("missing target collection or single type for relationship '%s'", attribute.Name)
	}

	// 2. Retrieve the "target" — typically a Collection by name, or possibly another single type
	var relatedCollection Collection
	err := database.DB.Where("name = ?", attribute.Target).First(&relatedCollection).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("target", attribute.Target).
				Warn("Target collection does not exist for single item relationship")
			return fmt.Errorf("target collection '%s' for relationship '%s' does not exist",
				attribute.Target, attribute.Name)
		}
		logger.Log.WithError(err).WithField("target", attribute.Target).
			Error("Database error while checking target collection for single item relationship")
		return fmt.Errorf("error validating target collection '%s' for relationship '%s': %w",
			attribute.Target, attribute.Name, err)
	}

	// 3. Depending on relation type, validate the data format
	switch attribute.Relation {
	case "oneToOne", "oneToMany":
		// Expect a single ID (float64 if coming from JSON) or an object
		if id, ok := value.(float64); ok {
			// Validate that an item with ID `id` exists in that collection
			if err := checkItemExists(relatedCollection.ID, uint(id)); err != nil {
				return fmt.Errorf("referenced item with ID '%d' in collection '%s' does not exist",
					uint(id), attribute.Target)
			}
		} else if _, isObject := value.(map[string]interface{}); !isObject {
			return fmt.Errorf("invalid relationship format for '%s': expected ID or object",
				attribute.Name)
		}

	case "manyToMany":
		// Expect an array of IDs or objects
		array, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("invalid relationship format for '%s': expected array", attribute.Name)
		}
		for _, element := range array {
			if id, isID := element.(float64); isID {
				if err := checkItemExists(relatedCollection.ID, uint(id)); err != nil {
					return fmt.Errorf("referenced item with ID '%d' in collection '%s' does not exist",
						uint(id), attribute.Target)
				}
			} else if _, isObject := element.(map[string]interface{}); !isObject {
				return fmt.Errorf("invalid element in relationship array for '%s'", attribute.Name)
			}
		}

	default:
		logger.Log.WithField("relation", attribute.Relation).Warn("Unsupported single item relationship type")
		return fmt.Errorf("unsupported relationship type '%s' for attribute '%s'",
			attribute.Relation, attribute.Name)
	}

	return nil
}
