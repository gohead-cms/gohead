package models

import (
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/validation"

	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	ID           uint    `json:"id"`
	CollectionID uint    `json:"collection"`
	Data         JSONMap `json:"data" gorm:"type:json"`
	// PublishedAt is set when the item is published. It is only meaningful
	// for collections with DraftAndPublish; nil means draft.
	PublishedAt *time.Time `json:"publishedAt" gorm:"index"`
}

// Item statuses in draft-and-publish collections.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// Status returns StatusPublished or StatusDraft.
func (i Item) Status() string {
	if i.PublishedAt != nil {
		return StatusPublished
	}
	return StatusDraft
}

// ItemResponse is the JSON shape of an item in every API response: its
// attributes at the top level, plus id, createdAt and updatedAt, and
// publishedAt and status for draft-and-publish collections. data is the
// (possibly relation-populated) item data; nil means item.Data.
func ItemResponse(ct Collection, item Item, data JSONMap) map[string]any {
	if data == nil {
		data = item.Data
	}
	out := make(map[string]any, len(data)+5)
	maps.Copy(out, data)
	out["id"] = item.ID
	out["createdAt"] = APITime(item.CreatedAt)
	out["updatedAt"] = APITime(item.UpdatedAt)
	if ct.DraftAndPublish {
		out["publishedAt"] = nil
		if item.PublishedAt != nil {
			out["publishedAt"] = APITime(*item.PublishedAt)
		}
		out["status"] = item.Status()
	}
	return out
}

// APITime normalizes a timestamp for responses: UTC, microsecond precision
// (what PostgreSQL and MySQL store), so a value reads the same whether it
// comes from memory right after a write or from the database later.
func APITime(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}

// ValidateItemValues validates item data, creates nested relations, and returns the processed data.
// It MUST be called within a database transaction.
func ValidateItemValues(ct Collection, itemData map[string]any, tx *gorm.DB) (JSONMap, error) {
	return validateItem(ct, itemData, tx, 0)
}

// ValidateItemUpdate is ValidateItemValues for an existing item: unique
// attributes may keep the item's own current values.
func ValidateItemUpdate(ct Collection, itemData map[string]any, tx *gorm.DB, itemID uint) (JSONMap, error) {
	return validateItem(ct, itemData, tx, itemID)
}

func validateItem(ct Collection, itemData map[string]any, tx *gorm.DB, excludeID uint) (JSONMap, error) {
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

		_, isText := value.(string)
		uniqueApplies := attribute.Type != "relation" && attribute.Type != "component" &&
			(attribute.Type != "media" || isText) // URL values, as before uploads existed
		if attribute.Unique && uniqueApplies {
			if err := validation.CheckFieldUniqueness(tx, ct.ID, attribute.Name, value, excludeID); err != nil {
				return nil, err
			}
		}

		if attribute.Type == "media" {
			if value == nil {
				if attribute.Required {
					return nil, fmt.Errorf("missing required attribute: '%s'", attribute.Name)
				}
				continue
			}
			processedValue, err := validateMediaValue(attribute, value, tx)
			if err != nil {
				return nil, fmt.Errorf("validation failed for attribute '%s': %w", attribute.Name, err)
			}
			processedData[attribute.Name] = processedValue
		} else if attribute.Type == "component" {
			if value == nil {
				if attribute.Required {
					return nil, fmt.Errorf("missing required attribute: '%s'", attribute.Name)
				}
				continue
			}
			processedValue, err := validateComponentValue(attribute, value, tx, 1)
			if err != nil {
				return nil, fmt.Errorf("validation failed for component '%s': %w", attribute.Name, err)
			}
			processedData[attribute.Name] = processedValue
		} else if attribute.Type == "relation" {
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

// ToUintSlice converts a stored many-to-many value to IDs. It accepts the
// shapes found before and after a JSON round-trip ([]uint right after
// validation, []any of float64 once read back). Invalid entries are dropped.
func ToUintSlice(value any) []uint {
	var out []uint
	add := func(v any) {
		if id := ToUint(v); id != 0 {
			out = append(out, id)
		}
	}
	switch vs := value.(type) {
	case []any:
		for _, v := range vs {
			add(v)
		}
	case []uint:
		for _, v := range vs {
			add(v)
		}
	case []int:
		for _, v := range vs {
			add(v)
		}
	case []float64:
		for _, v := range vs {
			add(v)
		}
	}
	return out
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
		expr, arg := validation.JSONTextExpr(tx.Dialector.Name(), field)
		query = query.Where(expr+" = ?", arg, fmt.Sprintf("%v", data[field]))
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
