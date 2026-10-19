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
func SaveItem(actor Actor, collection models.Collection, itemData models.JSONMap) (*models.Item, error) {
	items, err := SaveItems(actor, collection, []models.JSONMap{itemData})
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

// SaveItems validates and creates several items atomically, as drafts in
// draft-and-publish collections. See CreateItems.
func SaveItems(actor Actor, collection models.Collection, batch []models.JSONMap) ([]*models.Item, error) {
	inputs := make([]ItemInput, len(batch))
	for i, data := range batch {
		inputs[i] = ItemInput{Data: data}
	}
	return CreateItems(actor, collection, inputs)
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
// Each item gets its first revision. Events are published only after a
// successful commit.
func CreateItems(actor Actor, collection models.Collection, inputs []ItemInput) ([]*models.Item, error) {
	created := make([]*models.Item, 0, len(inputs))
	now := models.APITime(time.Now())
	nested := &models.NestedCreated{}

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		for i, in := range inputs {
			processedData, err := models.ValidateItemValues(collection, in.Data, models.WithNestedCollector(tx, nested))
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
			if _, err := recordRevision(tx, actor, revisionInput{
				entityType: models.RevisionEntityItem, entityID: item.ID, typeName: collection.Name,
				action: models.RevisionCreate, status: itemStatus(collection, item), data: item.Data, translations: item.Translations,
			}); err != nil {
				return err
			}
			created = append(created, item)
		}
		// Items created for relations given as objects get their history too.
		return recordNested(tx, actor, nested)
	})
	if txErr != nil {
		return nil, txErr
	}

	publishNested(nested)
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
// A change is recorded as a revision; a save that changes nothing is not.
func UpdateItemContent(actor Actor, collection models.Collection, itemID uint, data models.JSONMap, publish *bool) (*models.Item, error) {
	if publish != nil && !collection.DraftAndPublish {
		return nil, ErrDraftAndPublishDisabled
	}
	var item models.Item
	var transition events.EventType
	nested := &models.NestedCreated{}
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx).Where("id = ? AND collection_id = ?", itemID, collection.ID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		before := &entryState{data: item.Data, translations: item.Translations, status: itemStatus(collection, &item)}
		previous := item.Data
		processed, err := models.ValidateItemUpdate(collection, data, models.WithNestedCollector(tx, nested), item.ID)
		if err != nil {
			return &ItemValidationError{Err: err}
		}
		if err := recordNested(tx, actor, nested); err != nil {
			return err
		}
		// Write only what this request changes, so a concurrent publish or
		// unpublish of the same item is not overwritten.
		if err := tx.Model(&item).Updates(map[string]any{"data": processed}).Error; err != nil {
			return err
		}
		if publish != nil {
			if transition, err = setPublishedTx(tx, &item, *publish); err != nil {
				return err
			}
		}
		if err := reloadItem(tx, &item); err != nil {
			return err
		}
		action := models.RevisionUpdate
		if sameData(previous, item.Data) {
			switch transition {
			case "":
				return nil // nothing changed
			case events.EventTypeItemPublished:
				action = models.RevisionPublish
			default:
				action = models.RevisionUnpublish
			}
		}
		_, err = recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntityItem, entityID: item.ID, typeName: collection.Name,
			action: action, status: itemStatus(collection, &item), data: item.Data, translations: item.Translations, before: before,
		})
		return err
	})
	if txErr != nil {
		return nil, txErr
	}
	publishNested(nested)
	publishEvent(database.DB, events.EventTypeItemUpdated, &item)
	if transition != "" {
		publishEvent(database.DB, transition, &item)
	}
	return &item, nil
}

// SetItemPublished publishes or unpublishes an item of a draft-and-publish
// collection. Publishing an already published item keeps its publishedAt.
func SetItemPublished(actor Actor, collection models.Collection, itemID uint, publish bool) (*models.Item, error) {
	if !collection.DraftAndPublish {
		return nil, ErrDraftAndPublishDisabled
	}
	var item models.Item
	var transition events.EventType
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx).Where("id = ? AND collection_id = ?", itemID, collection.ID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		before := &entryState{data: item.Data, translations: item.Translations, status: itemStatus(collection, &item)}
		var err error
		if transition, err = setPublishedTx(tx, &item, publish); err != nil {
			return err
		}
		if err := reloadItem(tx, &item); err != nil {
			return err
		}
		if transition == "" {
			return nil
		}
		action := models.RevisionUnpublish
		if publish {
			action = models.RevisionPublish
		}
		_, err = recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntityItem, entityID: item.ID, typeName: collection.Name,
			action: action, status: itemStatus(collection, &item), data: item.Data, translations: item.Translations, before: before,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	if transition != "" {
		publishEvent(database.DB, transition, &item)
	}
	return &item, nil
}

// setPublishedTx changes the publication state with a conditional update,
// so that of two concurrent requests only the one that actually changes
// the state reports a transition (and emits the event).
func setPublishedTx(tx *gorm.DB, item *models.Item, publish bool) (events.EventType, error) {
	q := tx.Model(&models.Item{}).Where("id = ?", item.ID)
	var res *gorm.DB
	var event events.EventType
	if publish {
		res = q.Where("published_at IS NULL").Update("published_at", models.APITime(time.Now()))
		event = events.EventTypeItemPublished
	} else {
		res = q.Where("published_at IS NOT NULL").Update("published_at", nil)
		event = events.EventTypeItemUnpublished
	}
	if res.Error != nil {
		return "", fmt.Errorf("failed to update item: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return "", nil
	}
	return event, nil
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
		EventType:        eventType,
		CollectionName:   collection.Name,
		ItemID:           item.ID,
		ItemData:         item.Data,
		ItemTranslations: copyTranslations(item.Translations),
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

// DeleteItem deletes an item and records a "delete" revision holding the
// data it had, so the item can be restored. It returns ErrItemNotFound for
// an unknown item and publishes item:deleted after the commit.
func DeleteItem(actor Actor, id uint) error {
	var item models.Item
	var collection models.Collection
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx).First(&item, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrItemNotFound
			}
			return err
		}
		if err := tx.First(&collection, item.CollectionID).Error; err != nil {
			return fmt.Errorf("failed to find the item's collection: %w", err)
		}
		if _, err := recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntityItem, entityID: item.ID, typeName: collection.Name,
			action: models.RevisionDelete, status: itemStatus(collection, &item), data: item.Data, translations: item.Translations,
			before: &entryState{data: item.Data, translations: item.Translations, status: itemStatus(collection, &item)},
		}); err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
	if txErr != nil {
		return txErr
	}
	publishEvent(database.DB, events.EventTypeItemDeleted, &item)
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
	return PopulateRelationsIn(collection, data, level, publishedOnly, ReadLocale{})
}

// ReadLocale is the locale related entries are read in. The zero value
// reads the default locale.
type ReadLocale struct {
	Locale, Default string
	Fallback        bool
}

// resolved fills in the default locale when it is not known yet.
func (l ReadLocale) resolved() ReadLocale {
	if l.Default == "" {
		l.Default, _ = DefaultLocale()
		l.Fallback = true
	}
	if l.Locale == "" {
		l.Locale = l.Default
	}
	return l
}

// PopulateRelationsIn is PopulateRelations with related entries read in a
// locale (their localized attributes, and their locale fields).
func PopulateRelationsIn(collection models.Collection, data models.JSONMap, level uint, publishedOnly bool, loc ReadLocale) (models.JSONMap, error) {
	if level == 0 {
		if publishedOnly {
			return hideDraftIDs(collection, data)
		}
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
			nested, err := fetchItemWithRelations(targetCollection, id, level-1, publishedOnly, loc)
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
				nested, err := fetchItemWithRelations(targetCollection, id, level-1, publishedOnly, loc)
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
	return PopulateMedia(context.Background(), collection.Attributes, result), nil
}

// fetchItemWithRelations returns the related item in the flat API shape, or
// nil when it does not exist (or is a hidden draft).
func fetchItemWithRelations(collection models.Collection, itemID uint, level uint, publishedOnly bool, loc ReadLocale) (models.JSONMap, error) {
	item, err := GetVisibleItem(collection, itemID, publishedOnly)
	if errors.Is(err, ErrItemNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	data := item.Data
	localized := models.HasLocalized(collection.Attributes)
	if localized {
		loc = loc.resolved()
		data = models.LocalizeData(collection.Attributes, item.Data, item.Translations, loc.Locale, loc.Default, loc.Fallback)
	}
	data, err = PopulateRelationsIn(collection, data, level, publishedOnly, loc)
	if err != nil {
		return nil, err
	}
	if level == 0 {
		// A related item is shown as an object, so its files are too.
		data = PopulateMedia(context.Background(), collection.Attributes, data)
	}
	out := models.ItemResponse(collection, *item, data)
	if localized {
		out = models.LocalizedResponse(out, collection.Attributes, item.Translations, loc.Locale, loc.Default)
	}
	return out, nil
}

// hideDraftIDs removes, from unpopulated relation values, the IDs of drafts
// (and of items that no longer exist) in draft-and-publish targets, so even
// IDs of hidden items do not reach readers.
func hideDraftIDs(collection models.Collection, data models.JSONMap) (models.JSONMap, error) {
	result := make(models.JSONMap, len(data))
	maps.Copy(result, data)
	for _, attr := range collection.Attributes {
		raw, exists := data[attr.Name]
		if attr.Type != "relation" || !exists || raw == nil {
			continue
		}
		var target models.Collection
		if err := database.DB.Where("name = ?", attr.Target).First(&target).Error; err != nil {
			return nil, fmt.Errorf("failed to fetch target collection '%s': %w", attr.Target, err)
		}
		if !target.DraftAndPublish {
			continue
		}
		ids := models.ToUintSlice(raw)
		if attr.Relation != "manyToMany" {
			ids = nil
			if id := models.ToUint(raw); id != 0 {
				ids = []uint{id}
			}
		}
		var visible []uint
		if len(ids) > 0 {
			if err := database.DB.Model(&models.Item{}).
				Where("collection_id = ? AND id IN ? AND published_at IS NOT NULL", target.ID, ids).
				Pluck("id", &visible).Error; err != nil {
				return nil, err
			}
		}
		keep := make(map[uint]bool, len(visible))
		for _, id := range visible {
			keep[id] = true
		}
		if attr.Relation == "manyToMany" {
			out := []uint{}
			for _, id := range ids {
				if keep[id] {
					out = append(out, id)
				}
			}
			result[attr.Name] = out
		} else if len(ids) == 0 || !keep[ids[0]] {
			result[attr.Name] = nil
		}
	}
	return result, nil
}

// reloadItem re-reads an item into a fresh struct, so columns set to NULL
// (such as published_at after unpublishing) are not left at old values.
func reloadItem(tx *gorm.DB, item *models.Item) error {
	var fresh models.Item
	if err := tx.First(&fresh, item.ID).Error; err != nil {
		return err
	}
	*item = fresh
	return nil
}

// ItemCollectionName returns the name of the collection an item belongs to.
func ItemCollectionName(itemID uint) (string, error) {
	var names []string
	err := database.DB.Table("items").
		Joins("JOIN collections ON collections.id = items.collection_id").
		Where("items.id = ? AND items.deleted_at IS NULL AND collections.deleted_at IS NULL", itemID).
		Pluck("collections.name", &names).Error
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", ErrItemNotFound
	}
	return names[0], nil
}
