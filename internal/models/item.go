package models

import (
	"fmt"
	"maps"
	"strconv"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/validation"

	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	ID           uint    `json:"id"`
	CollectionID uint    `json:"collection"`
	Data         JSONMap `json:"data" gorm:"type:json"`
}

// ValidateItemValues validates item data, creates nested relations, and returns the processed data.
// It MUST be called within a database transaction.
func ValidateItemValues(ct Collection, itemData map[string]any, tx *gorm.DB) (JSONMap, error) {
	processedData := make(JSONMap)
	maps.Copy(processedData, itemData) // Work on a copy

	validAttributes := make(map[string]bool)
	for _, attr := range ct.Attributes {
		validAttributes[attr.Name] = true
	}

	for key := range itemData {
		if !validAttributes[key] {
			return nil, fmt.Errorf("unknown attribute: '%s'", key)
		}
	}

	for _, attribute := range ct.Attributes {
		value, exists := itemData[attribute.Name]

		if attribute.Required && !exists {
			return nil, fmt.Errorf("missing required attribute: '%s'", attribute.Name)
		}
		if !exists {
			continue
		}

		if attribute.Unique {
			if err := validation.CheckFieldUniqueness(ct.ID, attribute.Name, value); err != nil {
				return nil, err
			}
		}

		if attribute.Type == "relation" {
			// This now returns the processed value (with new IDs) and an error.
			processedValue, err := validateRelationship(attribute, value, tx)
			if err != nil {
				return nil, fmt.Errorf("validation failed for relationship '%s': %w", attribute.Name, err)
			}
			// Replace the original nested object with the final ID(s).
			processedData[attribute.Name] = processedValue
		} else {
			if err := validateAttributeValue(attribute, value); err != nil {
				return nil, fmt.Errorf("validation failed for attribute '%s': %w", attribute.Name, err)
			}
		}
	}

	logger.Log.WithField("collection", ct.Name).Info("Item data validation passed")
	return processedData, nil
}

// validateRelationship validates and processes a relationship, creating new items if necessary.
// It returns the final value for the relation (a single ID or an array of IDs).
// validateRelationship validates and processes a relationship with smart lookup support.
func validateRelationship(attribute Attribute, value any, tx *gorm.DB) (any, error) {
	if attribute.Target == "" {
		return nil, fmt.Errorf("missing target collection for relationship '%s'", attribute.Name)
	}

	var relatedCollection Collection
	if err := tx.Where("name = ?", attribute.Target).Preload("Attributes").First(&relatedCollection).Error; err != nil {
		if gorm.ErrRecordNotFound == err {
			return nil, fmt.Errorf("target collection '%s' does not exist", attribute.Target)
		}
		return nil, fmt.Errorf("db error validating target collection: %w", err)
	}

	// Helper function to resolve a single relation value with smart lookup
	resolveRelationValue := func(val any) (uint, error) {
		obj, isMap := val.(map[string]any)

		// Case 1: Raw ID provided
		if !isMap {
			id := ToUint(val)
			if id == 0 {
				return 0, fmt.Errorf("invalid ID format for relation '%s'", attribute.Name)
			}
			if err := checkItemExists(relatedCollection.ID, id, tx); err != nil {
				return 0, err
			}
			return id, nil
		}

		// Sub-case 2a: Object has an ID → Connect (or update if _upsert: true)
		if idVal, exists := obj["id"]; exists {
			id := ToUint(idVal)
			if err := checkItemExists(relatedCollection.ID, id, tx); err != nil {
				return 0, err
			}

			return id, nil
		}

		// Sub-case 2b: No ID provided → Smart Lookup
		existingID, found, err := lookup(relatedCollection, obj, tx)
		if err != nil {
			return 0, fmt.Errorf("smart lookup failed for '%s': %w", attribute.Name, err)
		}

		if found {
			logger.Log.WithField("collection", relatedCollection.Name).
				WithField("id", existingID).
				Info("Found existing item via smart lookup")

			return existingID, nil
		}

		// Not found → Create new item
		logger.Log.WithField("collection", relatedCollection.Name).Info("Creating new nested item")
		processedNestedData, err := ValidateItemValues(relatedCollection, obj, tx)
		if err != nil {
			return 0, fmt.Errorf("invalid data for new item in '%s': %w", attribute.Target, err)
		}

		newItem := &Item{
			CollectionID: relatedCollection.ID,
			Data:         processedNestedData,
		}
		if err := tx.Create(newItem).Error; err != nil {
			return 0, fmt.Errorf("failed to create nested item in '%s': %w", attribute.Target, err)
		}

		return newItem.ID, nil
	}

	// Process single or multiple relations
	if attribute.Relation == "manyToMany" {
		array, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("expected an array for many-to-many relation '%s'", attribute.Name)
		}

		var resolvedIDs []uint
		for _, element := range array {
			id, err := resolveRelationValue(element)
			if err != nil {
				return nil, err
			}
			resolvedIDs = append(resolvedIDs, id)
		}
		return resolvedIDs, nil
	} else {
		id, err := resolveRelationValue(value)
		if err != nil {
			return nil, err
		}
		return id, nil
	}
}

// checkItemExists now accepts a transaction object.
func checkItemExists(collectionID uint, itemID uint, tx *gorm.DB) error {
	var count int64
	err := tx.Model(&Item{}).Where("collection_id = ? AND id = ?", collectionID, itemID).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("referenced item with ID '%d' in collection ID '%d' does not exist", itemID, collectionID)
	}
	return nil
}

func ToUint(value any) uint {
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case float64:
		return uint(v)
	case int:
		return uint(v)
	case int64:
		return uint(v)
	case uint:
		return v
	case uint64:
		return uint(v)
	case string:
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0
		}
		return uint(i)
	case map[string]any:
		if idVal, exists := v["id"]; exists {
			// Recursively call toUint on the inner value to handle any type.
			return ToUint(idVal)
		}
	}
	return 0
}

// smartLookup attempts to find an existing item based on unique/lookup fields
func lookup(collection Collection, data map[string]any, tx *gorm.DB) (uint, bool, error) {
	// Build a query based on unique fields and lookup keys
	var lookupFields []string

	// First, try to find by unique fields
	for _, attr := range collection.Attributes {
		if attr.Unique && data[attr.Name] != nil {
			lookupFields = append(lookupFields, attr.Name)
		}
	}

	// If still no lookup fields, return not found
	if len(lookupFields) == 0 {
		return 0, false, nil
	}

	// Build the query
	query := tx.Model(&Item{}).Where("collection_id = ?", collection.ID)

	for _, field := range lookupFields {
		query = query.Where("data ->> ? = ?", field, fmt.Sprintf("%v", data[field]))
	}

	var items []Item
	if err := query.Limit(2).Find(&items).Error; err != nil {
		return 0, false, err
	}

	// If exactly one match found, return it
	if len(items) == 1 {
		return items[0].ID, true, nil
	}

	// If multiple matches, that's ambiguous
	if len(items) > 1 {
		return 0, false, fmt.Errorf("ambiguous lookup: found %d matching items", len(items))
	}

	// No matches
	return 0, false, nil
}
