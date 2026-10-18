package storage

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

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

// SaveItems validates and creates several items atomically, as drafts in
// draft-and-publish collections. See CreateItems.
func SaveItems(collection models.Collection, batch []models.JSONMap) ([]*models.Item, error) {
	inputs := make([]ItemInput, len(batch))
	for i, data := range batch {
		inputs[i] = ItemInput{Data: data}
	}
	return CreateItems(collection, inputs)
}

// ItemInput is one item to create.
type ItemInput struct {
	Data models.JSONMap
	// Publish creates the item as published. It is ignored unless the
	// collection uses draft and publish (other items are always visible).
	Publish bool
}

// ErrItemNotFound is returned when an item does not exist in a collection.
var ErrItemNotFound = errors.New("item not found")

// ErrDraftAndPublishDisabled is returned when publishing an item of a
// collection that does not use draft and publish.
var ErrDraftAndPublishDisabled = errors.New("this collection does not use draft and publish")

// CreateItems validates and creates several items atomically: either all are
// stored or none. Nested relations created during validation are part of the
// same transaction, and the stored data is the processed (validated) data.
// Events are published only after a successful commit.
func CreateItems(collection models.Collection, inputs []ItemInput) ([]*models.Item, error) {
	created := make([]*models.Item, 0, len(inputs))
	now := models.APITime(time.Now())

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		for i, in := range inputs {
			processedData, err := models.ValidateItemValues(collection, in.Data, tx)
			if err != nil {
				return &ItemValidationError{Index: i, Err: err}
			}
			item := &models.Item{
				CollectionID: collection.ID,
				Data:         processedData,
			}
			if collection.DraftAndPublish && in.Publish {
				item.PublishedAt = &now
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
		if item.PublishedAt != nil {
			publishEvent(database.DB, events.EventTypeItemPublished, item)
		}
	}
	return created, nil
}

// UpdateItemContent validates data (the item's complete new data) and saves
// it. publish changes the publication state of a draft-and-publish item
// (nil leaves it unchanged). It returns ErrItemNotFound when the item is not
// in the collection, an *ItemValidationError for invalid data, and publishes
// item:updated (plus item:published / item:unpublished on a state change).
func UpdateItemContent(collection models.Collection, itemID uint, data models.JSONMap, publish *bool) (*models.Item, error) {
	if publish != nil && !collection.DraftAndPublish {
		return nil, ErrDraftAndPublishDisabled
	}
	var item models.Item
	var transition events.EventType
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND collection_id = ?", itemID, collection.ID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		processed, err := models.ValidateItemUpdate(collection, data, tx, item.ID)
		if err != nil {
			return &ItemValidationError{Err: err}
		}
		item.Data = processed
		transition = applyPublish(&item, publish)
		return tx.Save(&item).Error
	})
	if txErr != nil {
		return nil, txErr
	}
	publishEvent(database.DB, events.EventTypeItemUpdated, &item)
	if transition != "" {
		publishEvent(database.DB, transition, &item)
	}
	return &item, nil
}

// SetItemPublished publishes or unpublishes an item of a draft-and-publish
// collection. Publishing an already published item keeps its publishedAt.
func SetItemPublished(collection models.Collection, itemID uint, publish bool) (*models.Item, error) {
	if !collection.DraftAndPublish {
		return nil, ErrDraftAndPublishDisabled
	}
	var item models.Item
	if err := database.DB.Where("id = ? AND collection_id = ?", itemID, collection.ID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}
	transition := applyPublish(&item, &publish)
	if transition == "" {
		return &item, nil
	}
	if err := database.DB.Model(&item).Update("published_at", item.PublishedAt).Error; err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}
	publishEvent(database.DB, transition, &item)
	return &item, nil
}

// applyPublish sets item.PublishedAt according to publish and returns the
// event to emit, if the state changed.
func applyPublish(item *models.Item, publish *bool) events.EventType {
	switch {
	case publish == nil:
		return ""
	case *publish && item.PublishedAt == nil:
		now := models.APITime(time.Now())
		item.PublishedAt = &now
		return events.EventTypeItemPublished
	case !*publish && item.PublishedAt != nil:
		item.PublishedAt = nil
		return events.EventTypeItemUnpublished
	}
	return ""
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

// GetVisibleItem fetches an item of the collection. With publishedOnly, a
// draft of a draft-and-publish collection is reported as not found.
func GetVisibleItem(ct models.Collection, itemID uint, publishedOnly bool) (*models.Item, error) {
	q := database.DB.Where("id = ? AND collection_id = ?", itemID, ct.ID)
	if publishedOnly && ct.DraftAndPublish {
		q = q.Where("published_at IS NOT NULL")
	}
	var item models.Item
	if err := q.First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}
	return &item, nil
}

// GetItemsByOffset lists items using offset/limit pagination (as GraphQL
// does). status filters draft-and-publish collections: "" (all),
// models.StatusPublished or models.StatusDraft.
func GetItemsByOffset(ct models.Collection, offset, limit int, status string) ([]models.Item, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}
	var items []models.Item
	var total int64
	q := withStatus(database.DB.Model(&models.Item{}).Where("collection_id = ?", ct.ID), ct, status)
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

// FetchNestedRelations replaces relation IDs in data with the related items,
// down to level levels, including drafts. See PopulateRelations.
func FetchNestedRelations(collection models.Collection, data models.JSONMap, level uint) (models.JSONMap, error) {
	return PopulateRelations(collection, data, level, false)
}

// PopulateRelations replaces relation IDs in data with the related items
// (in the API's flat item shape), down to level levels. With publishedOnly,
// drafts of draft-and-publish target collections are left out, as are
// related items that no longer exist.
func PopulateRelations(collection models.Collection, data models.JSONMap, level uint, publishedOnly bool) (models.JSONMap, error) {
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

		// Fetch target collection (with attributes, for deeper levels)
		var targetCollection models.Collection
		err := database.DB.Preload("Attributes").Where("name = ?", attr.Target).First(&targetCollection).Error
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
			nested, err := fetchItemWithRelations(targetCollection, id, level-1, publishedOnly)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch nested item for '%s': %w", attr.Name, err)
			}
			if nested == nil {
				result[attr.Name] = nil
			} else {
				result[attr.Name] = nested
			}
		case "manyToMany":
			nestedItems := []models.JSONMap{}
			for _, id := range models.ToUintSlice(raw) {
				nested, err := fetchItemWithRelations(targetCollection, id, level-1, publishedOnly)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch nested item for '%s': %w", attr.Name, err)
				}
				if nested != nil {
					nestedItems = append(nestedItems, nested)
				}
			}
			result[attr.Name] = nestedItems
		}
	}
	return result, nil
}

// fetchItemWithRelations returns the related item in the flat API shape, or
// nil when it does not exist (or is a hidden draft).
func fetchItemWithRelations(collection models.Collection, itemID uint, level uint, publishedOnly bool) (models.JSONMap, error) {
	item, err := GetVisibleItem(collection, itemID, publishedOnly)
	if errors.Is(err, ErrItemNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	data, err := PopulateRelations(collection, item.Data, level, publishedOnly)
	if err != nil {
		return nil, err
	}
	return models.ItemResponse(collection, *item, data), nil
}
