package storage

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

func SaveItem(item *models.Item) error {
	return database.DB.Create(item).Error
}

// GetItemByID fetches an item by its ID and collection ID.
func GetItemByID(collectionID uint, itemID uint) (*models.Item, error) {
	var item models.Item
	if err := database.DB.Where("id = ? AND collection_id = ?", itemID, collectionID).First(&item).Error; err != nil {
		return nil, fmt.Errorf("item with ID %d not found: %w", itemID, err)
	}
	return &item, nil
}

func GetItems(CollectionID uint) ([]models.Item, error) {
	var items []models.Item
	err := database.DB.Where("collection_id = ?", CollectionID).Find(&items).Error
	return items, err
}

func UpdateItem(itemID uint, data models.JSONMap) error {
	// Fetch the existing item
	var item models.Item
	if err := database.DB.Where("id = ?", itemID).First(&item).Error; err != nil {
		logger.Log.WithField("item_id", itemID).WithError(err).Error("Failed to find item")
		return fmt.Errorf("collection item not found")
	}

	// Update the item data
	item.Data = data
	if err := database.DB.Save(&item).Error; err != nil {
		logger.Log.WithField("item_id", itemID).WithError(err).Error("Failed to update item data")
		return fmt.Errorf("failed to update item data: %w", err)
	}

	// Save updated relationships
	//TO CHECK

	logger.Log.WithField("item_id", itemID).Info("Item updated successfully")
	return nil
}

func DeleteItem(id uint) error {
	if err := database.DB.Where("id = ?", id).Delete(&models.Item{}).Error; err != nil {
		return err
	}

	return nil
}

// FetchNestedRelations retrieves nested relationships up to the specified level for an item's data.
func FetchNestedRelations(ct models.Collection, data models.JSONMap, level uint) (models.JSONMap, error) {
	logger.Log.WithField("data'", data).Debug("FetchNestedRelations:data")
	if level <= 0 {
		return data, nil
	}
	logger.Log.WithField("collection", ct).Debug("FetchNestedRelations:collection")
	// Iterate through the attributes to find relationships
	for _, attribute := range ct.Attributes {
		if attribute.Type != "relation" {
			continue
		}
		// Get the relationship data from the current item's data
		relationshipValue, exists := data[attribute.Name]
		logger.Log.WithFields(logrus.Fields{
			"attribute": attribute,
			"exists":    exists,
			"(exists)":  exists,
		}).Debug("FetchNestedRelations:attribute")
		if !exists {
			continue
		}

		// Fetch the target collection details
		var targetCollection models.Collection
		err := database.DB.Where("name = ?", attribute.Target).First(&targetCollection).Error
		if err != nil {
			logger.Log.WithError(err).WithField("target", attribute.Target).Warn("Failed to fetch target collection")
			return nil, fmt.Errorf("failed to fetch target collection '%s': %w", attribute.Target, err)
		}

		switch attribute.Relation {
		case "oneToOne", "oneToMany":
			if idFloat, ok := relationshipValue.(int); ok {
				id := uint(idFloat)
				nestedItem, err := fetchItemWithRelations(ct, uint(id), data, level-1)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch nested item for '%s': %w", attribute.Name, err)
				}
				data[attribute.Name] = nestedItem
			} else {
				logger.Log.WithField("relationshipValue", relationshipValue).Warn("Failed to fetch relationshipValue")
			}

		case "manyToMany":
			// For many-to-many relationships, resolve each related item
			if ids, ok := relationshipValue.([]interface{}); ok {
				var nestedItems []models.JSONMap
				for _, rawID := range ids {
					if id, ok := rawID.(float64); ok {
						nestedItem, err := fetchItemWithRelations(targetCollection, uint(id), data, level-1)
						if err != nil {
							return nil, fmt.Errorf("failed to fetch nested item for '%s': %w", attribute.Name, err)
						}
						nestedItems = append(nestedItems, nestedItem)
					}
				}
				data[attribute.Name] = nestedItems
			}
		}
	}

	return data, nil
}

// fetchItemWithRelations retrieves an item and recursively fetches its nested relations.
func fetchItemWithRelations(ct models.Collection, id uint, data models.JSONMap, level uint) (models.JSONMap, error) {
	var item models.Item
	err := database.DB.Where("id = ?", id).First(&item).Error
	if err != nil {
		logger.Log.WithError(err).WithFields(map[string]interface{}{
			"item_id": id,
		}).Warn("Failed to fetch item with relations")
		return nil, fmt.Errorf("failed to fetch item with ID '%d' in collection '%s': %w", id, err)
	}

	// Fetch nested relations for the item
	data, err = FetchNestedRelations(ct, item.Data, level)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nested relations for item '%d': %w", id, err)
	}

	return data, nil
}
