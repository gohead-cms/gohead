package models

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/validation"
	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	CollectionID uint    `json:"collection"`
	Data         JSONMap `json:"data" gorm:"type:json"`
}

// ValidateItemValues validates a single item's data against the Collection's schema (attributes).
func ValidateItemValues(ct Collection, itemData map[string]interface{}) error {
	for _, attribute := range ct.Attributes {
		value, exists := itemData[attribute.Name]

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
			logger.Log.WithField("item", value).Info("Validation: check uniqueness")
			if err := validation.CheckFieldUniqueness(ct.ID, attribute.Name, value); err != nil {
				return err
			}
		}

		// Validate attribute value or relationships
		if attribute.Type != "relation" { // Validate regular attributes
			if err := validateAttributeValue(attribute, value); err != nil {
				logger.Log.WithFields(logrus.Fields{
					"attribute": attribute.Name,
					"type":      attribute.Type,
					"value":     value,
				}).Warn("Validation failed for attribute")
				return fmt.Errorf("validation failed for attribute '%s': %w", attribute.Name, err)
			}
		} else { // Validate relationships
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
func validateRelationship(attribute Attribute, value interface{}) error {
	// Ensure the target collection exists and fetch its `collection_id`
	if attribute.Target == "" {
		logger.Log.WithField("attribute", attribute.Name).Warn("Missing target collection for relationship")
		return fmt.Errorf("missing target collection for relationship '%s'", attribute.Name)
	}

	// Query the database directly to fetch the target collection
	var relatedCollection Collection
	err := database.DB.Where("name = ?", attribute.Target).First(&relatedCollection).Error
	if err != nil {
		if err.Error() == "record not found" {
			logger.Log.WithField("target", attribute.Target).Warn("Target collection does not exist")
			return fmt.Errorf("target collection '%s' for relationship '%s' does not exist", attribute.Target, attribute.Name)
		}
		logger.Log.WithError(err).WithField("target", attribute.Target).Error("Database error while checking target collection")
		return fmt.Errorf("error validating target collection '%s' for relationship '%s': %w", attribute.Target, attribute.Name, err)
	}

	// Validate relationship values based on type
	switch attribute.Relation {
	case "oneToOne", "oneToMany":
		// For single relationships, validate ID or nested object
		if id, ok := value.(float64); ok {
			// Use the related collection's ID to validate the item existence
			if err := checkItemExists(relatedCollection.ID, uint(id)); err != nil {
				return fmt.Errorf("referenced item with ID '%d' in collection '%s' does not exist", uint(id), attribute.Target)
			}
		} else if _, isObject := value.(map[string]interface{}); !isObject {
			logger.Log.WithField("attribute", attribute.Name).Warn("Invalid relationship format: expected ID or object")
			return fmt.Errorf("invalid relationship format for '%s': expected ID or object", attribute.Name)
		}

	case "manyToMany":
		// For many-to-many relationships, validate array of IDs or objects
		array, ok := value.([]interface{})
		if !ok {
			logger.Log.WithField("attribute", attribute.Name).Warn("Invalid relationship format: expected array")
			return fmt.Errorf("invalid relationship format for '%s': expected array", attribute.Name)
		}
		for _, element := range array {
			if id, isID := element.(float64); isID {
				// Use the related collection's ID to validate the item existence
				if err := checkItemExists(relatedCollection.ID, uint(id)); err != nil {
					return fmt.Errorf("referenced item with ID '%d' in collection '%s' does not exist", uint(id), attribute.Target)
				}
			} else if _, isObject := element.(map[string]interface{}); !isObject {
				logger.Log.WithField("attribute", attribute.Name).Warn("Invalid element in relationship array")
				return fmt.Errorf("invalid element in relationship array for '%s'", attribute.Name)
			}
		}

	default:
		logger.Log.WithField("relation", attribute.Relation).Warn("Unsupported relationship type")
		return fmt.Errorf("unsupported relationship type '%s' for '%s'", attribute.Relation, attribute.Name)
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
