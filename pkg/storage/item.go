package storage

import (
	"context"
	"fmt"
	"maps"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"gorm.io/gorm"
)

// SaveItem creates a new item in the database and then publishes a generic
// 'item:created' event to the dispatcher queue.
func SaveItem(item *models.Item) error {
	// Step 1: Create the item in the database.
	// After this call succeeds, GORM will automatically populate the 'ID'
	// field on the 'item' struct that was passed in.
	if err := database.DB.Create(item).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to save new item")
		return err
	}

	// Step 2: Publish the creation event.
	// We check if the asynqClient has been initialized.
	if asynqClient != nil {
		// We need the collection name. We can get it efficiently from the item's CollectionID.
		var collection models.Collection
		if err := database.DB.First(&collection, item.CollectionID).Error; err != nil {
			// The item was saved, but we couldn't find its collection to publish the event.
			// This should be logged, but we don't return an error because the main
			// operation (saving the item) was successful.
			logger.Log.WithError(err).
				WithField("collection_id", item.CollectionID).
				Error("Failed to find collection for event publishing after item creation")
			return nil
		}

		// Now, item.ID has the correct, non-zero ID from the database.
		payload := events.CollectionEventPayload{
			EventType:      events.EventTypeItemCreated,
			CollectionName: collection.Name,
			ItemID:         item.ID,
			ItemData:       item.Data,
		}
		if err := events.EnqueueCollectionEvent(context.Background(), asynqClient, payload); err != nil {
			// Log the enqueuing error but don't fail the overall SaveItem operation.
			logger.Log.WithError(err).Error("Failed to enqueue item:created event")
		}
	}

	return nil
}

// SaveItem creates a new item in the database and then publishes a generic
// 'item:created' event to the dispatcher queue.
func SaveItemWithTransaction(tx *gorm.DB, item *models.Item) error {
	// Step 1: Create the item in the database.
	// After this call succeeds, GORM will automatically populate the 'ID'
	// field on the 'item' struct that was passed in.
	if err := tx.Create(item).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to save new item")
		return err
	}

	// Step 2: Publish the creation event.
	// We check if the asynqClient has been initialized.
	if asynqClient != nil {
		// We need the collection name. We can get it efficiently from the item's CollectionID.
		var collection models.Collection
		if err := tx.First(&collection, item.CollectionID).Error; err != nil {
			// The item was saved, but we couldn't find its collection to publish the event.
			// This should be logged, but we don't return an error because the main
			// operation (saving the item) was successful.
			logger.Log.WithError(err).
				WithField("collection_id", item.CollectionID).
				Error("Failed to find collection for event publishing after item creation")
			return nil
		}

		// Now, item.ID has the correct, non-zero ID from the database.
		payload := events.CollectionEventPayload{
			EventType:      events.EventTypeItemCreated,
			CollectionName: collection.Name,
			ItemID:         item.ID,
			ItemData:       item.Data,
		}
		if err := events.EnqueueCollectionEvent(context.Background(), asynqClient, payload); err != nil {
			// Log the enqueuing error but don't fail the overall SaveItem operation.
			logger.Log.WithError(err).Error("Failed to enqueue item:created event")
		}
	}

	return nil
}

// GetItemByID fetches an item by its ID and collection ID.
func GetItemByID(collectionID uint, itemID uint) (*models.Item, error) {
	var item models.Item
	if err := database.DB.Where("id = ? AND collection_id = ?", itemID, collectionID).First(&item).Error; err != nil {
		return nil, fmt.Errorf("item with ID %d not found: %w", itemID, err)
	}
	return &item, nil
}

func GetItems(collectionID uint, page, pageSize int) ([]models.Item, int, error) {
	var items []models.Item
	var totalItems int64

	// Calculate offset
	offset := (page - 1) * pageSize

	// Fetch total count
	err := database.DB.Model(&models.Item{}).
		Where("collection_id = ?", collectionID).
		Count(&totalItems).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count items: %w", err)
	}

	// Fetch paginated data
	err = database.DB.Where("collection_id = ?", collectionID).
		Offset(offset).Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch items: %w", err)
	}

	return items, int(totalItems), nil
}

func UpdateItem(itemID uint, data models.JSONMap) error {
	var item models.Item
	if err := database.DB.Where("id = ?", itemID).First(&item).Error; err != nil {
		logger.Log.WithField("item_id", itemID).WithError(err).Error("Failed to find item")
		return fmt.Errorf("collection item not found")
	}

	item.Data = data
	if err := database.DB.Save(&item).Error; err != nil {
		logger.Log.WithField("item_id", itemID).WithError(err).Error("Failed to update item data")
		return fmt.Errorf("failed to update item data: %w", err)
	}

	// Publish the 'updated' event.
	if asynqClient != nil {
		var collection models.Collection
		if err := database.DB.First(&collection, item.CollectionID).Error; err != nil {
			logger.Log.WithError(err).WithField("collection_id", item.CollectionID).Error("Failed to find collection for event publishing after item update")
			return nil
		}

		payload := events.CollectionEventPayload{
			EventType:      events.EventTypeItemUpdated,
			CollectionName: collection.Name,
			ItemID:         item.ID,
			ItemData:       item.Data,
		}
		if err := events.EnqueueCollectionEvent(context.Background(), asynqClient, payload); err != nil {
			logger.Log.WithError(err).Error("Failed to enqueue item:updated event")
		}
	}

	logger.Log.WithField("item_id", itemID).Info("Item updated successfully")
	return nil
}

func DeleteItem(id uint) error {
	// --- NEW: Publish Event ---
	// Before deleting, we must fetch the item to get its data for the event payload.
	var item models.Item
	// Fetch the item before deleting it so we can include its data in the event.
	err := database.DB.First(&item, id).Error
	if err == nil && asynqClient != nil {
		// If the item was found, try to find its collection to get the name.
		var collection models.Collection
		if err := database.DB.First(&collection, item.CollectionID).Error; err == nil {
			// If both are found, publish the event.
			payload := events.CollectionEventPayload{
				EventType:      events.EventTypeItemDeleted,
				CollectionName: collection.Name,
				ItemID:         item.ID,
				ItemData:       item.Data,
			}
			if err := events.EnqueueCollectionEvent(context.Background(), asynqClient, payload); err != nil {
				logger.Log.WithError(err).Error("Failed to enqueue item:deleted event")
			}
		}
	}
	// --- END ---

	if err := database.DB.Where("id = ?", id).Delete(&models.Item{}).Error; err != nil {
		return err
	}

	return nil
}

func FetchNestedRelations(collection models.Collection, data models.JSONMap, level uint) (models.JSONMap, error) {
	if level == 0 {
		return data, nil
	}
	// We'll mutate a copy
	result := make(models.JSONMap, len(data))
	maps.Copy(result, data)

	for _, attr := range collection.Attributes {
		if attr.Type != "relation" {
			continue
		}

		raw, exists := data[attr.Name]
		if !exists || raw == nil {
			// Always output as empty array/object if missing
			switch attr.Relation {
			case "manyToMany":
				result[attr.Name] = []any{}
			case "oneToOne", "oneToMany":
				result[attr.Name] = nil
			}
			continue
		}

		// Fetch target collection
		var targetCollection models.Collection
		err := database.DB.Where("name = ?", attr.Target).First(&targetCollection).Error
		if err != nil {
			return nil, fmt.Errorf("failed to fetch target collection '%s': %w", attr.Target, err)
		}

		switch attr.Relation {
		case "oneToOne", "oneToMany":
			id := models.ToUint(raw)
			if id == 0 {
				result[attr.Name] = nil
				break
			}
			nested, err := fetchItemWithRelations(targetCollection, id, level-1)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch nested item for '%s': %w", attr.Name, err)
			}
			result[attr.Name] = nested
		case "manyToMany":
			var nestedItems []models.JSONMap
			logger.Log.WithField("netste", nestedItems)
			switch ids := raw.(type) {
			case []any:
				for _, elem := range ids {
					id := models.ToUint(elem)
					if id == 0 {
						continue
					}
					nested, err := fetchItemWithRelations(targetCollection, id, level-1)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch nested item for '%s': %w", attr.Name, err)
					}
					nestedItems = append(nestedItems, nested)
				}
			case []float64:
				for _, elem := range ids {
					id := uint(elem)
					nested, err := fetchItemWithRelations(targetCollection, id, level-1)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch nested item for '%s': %w", attr.Name, err)
					}
					nestedItems = append(nestedItems, nested)
				}
			default:
				// If not an array, just output empty array
			}
			result[attr.Name] = nestedItems
		}
	}
	return result, nil
}

func fetchItemWithRelations(collection models.Collection, itemID uint, level uint) (models.JSONMap, error) {
	var item models.Item

	err := database.DB.Where("id = ? AND collection_id = ?", itemID, collection.ID).First(&item).Error
	if err != nil {
		return nil, err
	}
	item.Data["id"] = item.ID
	return FetchNestedRelations(collection, item.Data, level)
}
