package models

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/validation"
	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	CollectionID uint    `json:"collection"`
	Data         JSONMap `json:"data" gorm:"type:json"`
}

// ValidateItem validates the Item model.
func ValidateItemModel(item Item, collection Collection) error {
	// Check if CollectionID is set
	if item.CollectionID == 0 {
		logger.Log.WithField("item", item).Warn("Validation failed: missing CollectionID")
		return fmt.Errorf("CollectionID is required")
	}

	// Check if Data is empty
	if len(item.Data) == 0 {
		logger.Log.WithField("item", item).Warn("Validation failed: empty Data field")
		return fmt.Errorf("Data cannot be empty")
	}

	// Validate fields in Data against the Collection schema
	for _, attribute := range collection.Attributes {
		value, exists := item.Data[attribute.Name]
		if attribute.Required && !exists {
			logger.Log.WithField("field", attribute.Name).Warn("Validation failed: missing required field")
			return fmt.Errorf("missing required field: '%s'", attribute.Name)
		}

		// Skip validation for non-required fields that are not present
		if !exists {
			continue
		}

		// Validate the field value
		if err := validateFieldValue(attribute, value); err != nil {
			logger.Log.WithField("field", attribute.Name).WithError(err).Warn("Validation failed for field value")
			return err
		}
	}

	logger.Log.WithField("item", item).Info("Item validation passed")
	return nil
}

// ValidateItemData validates a single item’s data *against* the Collection’s schema (fields).
func ValidateItemData(ct Collection, data map[string]interface{}) error {
	// Validate required fields & field data
	for _, attribute := range ct.Attributes {
		value, exists := data[attribute.Name]

		if attribute.Required && !exists {
			logger.Log.WithField("field", attribute.Name).Warn("Validation failed: missing required field")
			return fmt.Errorf("missing required field: '%s'", attribute.Name)
		}
		if !exists {
			continue
		}

		logger.Log.WithField("item", value).Info("Validation: check uniqueness")
		if attribute.Unique {
			logger.Log.WithField("item", value).Info("Validation: check uniqueness")
			if err := validation.CheckFieldUniqueness(ct.ID, attribute.Name, value); err != nil {
				return err
			}
		}

		// Validate field's value
		if err := validateFieldValue(attribute, value); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"field": attribute.Name,
				"type":  attribute.Type,
				"value": value,
			}).Warn("Validation failed for field")
			return fmt.Errorf("validation failed for field '%s': %w", attribute.Name, err)
		}
	}

	// Check for unknown fields
	//for key := range data {
	//	isValidField := false
	//	for _, field := range ct.Attributes {
	//		if key == field.Name {
	//			isValidField = true
	//			break
	//		}
	//	}
	// Also allow relationships as valid top-level keys
	//if !isValidField {
	//	for _, rel := range ct.Relationships {
	//		if key == rel.Field {
	//			isValidField = true
	//			break
	//		}
	//	}
	//}
	//if !isValidField {
	//	logger.Log.WithField("field", key).Warn("Validation failed: unknown field")
	//	return fmt.Errorf("unknown field: '%s'", key)
	//}
	//}

	logger.Log.WithField("collection", ct.Name).Info("Item data validation passed")
	return nil
}
