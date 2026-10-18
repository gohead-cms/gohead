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

// ItemValidationError wraps a validation failure for an item, as opposed to a
// database error, so callers can answer 400 instead of 500.
type ItemValidationError struct {
	Index int // position in a batch; 0 for single saves
	Err   error
}

func (e *ItemValidationError) Error() string { return e.Err.Error() }
func (e *ItemValidationError) Unwrap() error { return e.Err }

// SaveItem validates and creates a single item, then publishes an
// 'item:created' event once the transaction has committed.
func SaveItem(collection models.Collection, itemData models.JSONMap) (*models.Item, error) {
	items, err := SaveItems(collection, []models.JSONMap{itemData})
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

// SaveItems validates and creates several items atomically: either all are
// stored or none. Nested relations created during validation are part of the
// same transaction, and the stored data is the processed (validated) data.
// Events are published only after a successful commit.
func SaveItems(collection models.Collection, batch []models.JSONMap) ([]*models.Item, error) {
	created := make([]*models.Item, 0, len(batch))

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		for i, data := range batch {
			processedData, err := models.ValidateItemValues(collection, data, tx)
			if err != nil {
				return &ItemValidationError{Index: i, Err: err}
			}
			item := &models.Item{
				CollectionID: collection.ID,
				Data:         processedData,
			}
			if err := tx.Create(item).Error; err != nil {
				return err
			}
			created = append(created, item)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	for _, item := range created {
		publishEvent(database.DB, events.EventTypeItemCreated, item)
	}
	return created, nil
}

func publishEvent(db *gorm.DB, eventType events.EventType, item *models.Item) {
	// 1. Safety Check: If the event system (Asynq) isn't configured, do nothing.
	if asynqClient == nil {
		return
	}

	// 2. Data Enrichment: The event needs the collection's name, not just its ID.
	// We perform a quick lookup to get it.
	var collection models.Collection
	if err := db.First(&collection, item.CollectionID).Error; err != nil {
		logger.Log.WithError(err).
			WithField("collection_id", item.CollectionID).
			Error("Failed to find collection for event publishing")
		return // Do not fail the entire API request if this lookup fails.
	}

	// 3. Payload Creation: Construct the event payload with all necessary information.
	payload := events.CollectionEventPayload{
		EventType:      eventType,
		CollectionName: collection.Name,
		ItemID:         item.ID,
		ItemData:       item.Data,
	}

	// 4. Enqueue Event: Send the event to the dispatcher via the Asynq message queue.
	if err := events.EnqueueCollectionEvent(context.Background(), asynqClient, payload); err != nil {
		logger.Log.WithError(err).Error("Failed to enqueue event")
	} else {
		logger.Log.WithField("event_type", eventType).WithField("item_id", item.ID).Info("Successfully enqueued event")
	}
}

// GetItemByID fetches an item by its ID and collection ID.
func GetItemByID(collectionID uint, itemID uint) (*models.Item, error) {
	var item models.Item
	if err := database.DB.Where("id = ? AND collection_id = ?", itemID, collectionID).First(&item).Error; err != nil {
		return nil, fmt.Errorf("item with ID %d not found: %w", itemID, err)
	}
	return &item, nil
}

// GetItemsByOffset lists items using offset/limit pagination (as GraphQL does).
func GetItemsByOffset(collectionID uint, offset, limit int) ([]models.Item, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}
	var items []models.Item
	var total int64
	q := database.DB.Model(&models.Item{}).Where("collection_id = ?", collectionID)
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count items: %w", err)
	}
	if err := q.Session(&gorm.Session{}).Order("id ASC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch items: %w", err)
	}
	return items, int(total), nil
}

// GetItems lists items using page/pageSize pagination (page starts at 1).
func GetItems(collectionID uint, page, pageSize int) ([]models.Item, int, error) {
	if page < 1 {
		page = 1
	}
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
		Order("id ASC").Offset(offset).Limit(pageSize).
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
