package models

import (
	"fmt"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/validation"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	ID           uint    `json:"id"`
	CollectionID uint    `json:"collection"`
	Data         JSONMap `json:"data" gorm:"type:json"`
}

// ValidateItemValues validates a single item's data against the Collection's schema (attributes).
// Now rejects any extra fields not defined in the schema.
func ValidateItemValues(ct Collection, itemData map[string]any) error {
	// Build a set of valid attribute names from the collection schema
	validAttributes := make(map[string]bool, len(ct.Attributes))
	for _, attr := range ct.Attributes {
		validAttributes[attr.Name] = true
	}

	// Check for unknown fields in itemData
	for key := range itemData {
		if !validAttributes[key] {
			logger.Log.WithField("attribute", key).Warn("Validation failed: unknown attribute")
			return fmt.Errorf("unknown attribute: '%s'", key)
		}
	}

	// Now perform the usual checks for required, uniqueness, types, etc.
	for _, attribute := range ct.Attributes {
		value, exists := itemData[attribute.Name]
		logger.Log.WithField("item", value).Debug("Validation: item")
		logger.Log.WithField("item", attribute.Type).Debug("Validation: type")
		// Check for required attributes
		if attribute.Required && !exists {
			logger.Log.WithField("attribute", attribute.Name).Warn("Validation failed: missing required attribute")
			return fmt.Errorf("missing required attribute: '%s'", attribute.Name)
		}
		if !exists {
			continue
		}

		// Check uniqueness
		if attribute.Unique {
			logger.Log.WithField("item", value).Debug("Validation: check uniqueness")
			if err := validation.CheckFieldUniqueness(ct.ID, attribute.Name, value); err != nil {
				return err
			}
		}

		// Validate attribute value or relationships
		if attribute.Type != "relation" {
			logger.Log.WithField("item", value).Debug("Validation: check single type")
			// Validate regular attributes
			if err := validateAttributeValue(attribute, value); err != nil {
				logger.Log.WithFields(logrus.Fields{
					"attribute": attribute.Name,
					"type":      attribute.Type,
					"value":     value,
				}).Warn("Validation failed for attribddute")
				return fmt.Errorf("validation failed for attribute '%s': %w", attribute.Name, err)
			}
		} else {
			logger.Log.WithField("item", value).Debug("Validation: check relation")
			// Validate relationships
			if err := validateRelationship(attribute, value); err != nil {
				logger.Log.WithField("attribute", attribute.Name).Warn("Validation failed for relationship")
				return fmt.Errorf("validation failed for relationship '%s': %w", attribute.Name, err)
			}
		}
	}

	logger.Log.WithField("collection", ct.Name).Info("Item data validation passed")
	return nil
}

// validateRelationship checks the validity of a relationship field and verifies if referenced collection and IDs exist.
func validateRelationship(attribute Attribute, value any) error {
	if attribute.Target == "" {
		return fmt.Errorf("missing target collection for relationship '%s'", attribute.Name)
	}

	var relatedCollection Collection
	if err := database.DB.Where("name = ?", attribute.Target).First(&relatedCollection).Error; err != nil {
		if gorm.ErrRecordNotFound == err {
			return fmt.Errorf("target collection '%s' does not exist", attribute.Target)
		}
		return fmt.Errorf("db error validating target collection '%s': %w", attribute.Target, err)
	}

	// This helper function correctly handles both raw IDs (like 3) and nested objects (like {"id": 3})
	checkValue := func(val any) error {
		var itemID uint
		// Case 1: The value is a raw ID (e.g., 3.0 from JSON)
		if id, ok := val.(float64); ok {
			itemID = uint(id)
		} else if obj, ok := val.(map[string]any); ok {
			// Case 2: The value is a nested object, like {"id": 3}
			idVal, exists := obj["id"]
			if !exists {
				return fmt.Errorf("nested object for relationship '%s' is missing an 'id' field", attribute.Name)
			}
			if idFloat, isFloat := idVal.(float64); isFloat {
				itemID = uint(idFloat)
			} else {
				return fmt.Errorf("nested object 'id' for relationship '%s' must be a number", attribute.Name)
			}
		} else {
			// Case 3: The format is invalid
			return fmt.Errorf("invalid format for relationship '%s': expected an ID or an object with an 'id'", attribute.Name)
		}

		// Now, reliably perform the existence check with the extracted ID
		logger.Log.WithField("id_to_check", itemID).Debug("Validation: checking if related item exists")
		if err := checkItemExists(relatedCollection.ID, itemID); err != nil {
			return fmt.Errorf("referenced item with ID '%d' in collection '%s' does not exist", itemID, attribute.Target)
		}
		return nil
	}

	switch attribute.Relation {
	case "oneToOne", "oneToMany":
		return checkValue(value)

	case "manyToMany":
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("invalid format for relationship '%s': expected an array", attribute.Name)
		}
		for _, element := range array {
			if err := checkValue(element); err != nil {
				return err // Return on the first error found in the array
			}
		}

	default:
		return fmt.Errorf("unsupported relationship type '%s'", attribute.Relation)
	}

	return nil
}

// checkItemExists verifies if an item exists in a specific collection by ID.
func checkItemExists(collectionID uint, itemID uint) error {
	var count int64
	err := database.DB.Model(&Item{}).Where("collection_id = ? AND id = ?", collectionID, itemID).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("item with ID '%d' does not exist in the specified collection", itemID)
	}
	return nil
}
