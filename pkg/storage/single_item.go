package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// CreateSingleItem creates a new SingleItem for a given Singleton and
// records its first revision. Returns an error if an item already exists
// for that Singleton (one-to-one).
func CreateSingleItem(actor Actor, st *models.Singleton, itemData map[string]any) (*models.SingleItem, error) {
	// Validate the new item data
	if valErr := models.ValidateSingleItemValues(*st, itemData); valErr != nil {
		logger.Log.WithError(valErr).WithField("Singleton", st.Name).
			Warn("Validation failed for single item creation")
		return nil, valErr
	}

	item := &models.SingleItem{
		SingleTypeID: st.ID,
		Data:         itemData,
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.SingleItem
		err := tx.Where("single_type_id = ?", st.ID).First(&existing).Error
		if err == nil {
			return fmt.Errorf("a single item already exists for single type '%s'", st.Name)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(item).Error; err != nil {
			return fmt.Errorf("failed to create single item: %w", err)
		}
		_, err = recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntitySingleton, entityID: st.ID, typeName: st.Name,
			action: models.RevisionCreate, data: item.Data, translations: item.Translations,
		})
		return err
	})
	if err != nil {
		logger.Log.WithError(err).WithField("Singleton", st.Name).Warn("Failed to create single item")
		return nil, err
	}

	logger.Log.WithField("Singleton", st.Name).Info("Single item created successfully")
	publishSingletonEvent(actor, events.EventTypeSingletonUpdated, st.Name, item)
	return item, nil
}

// GetSingleItemByType retrieves the SingleItem for a given single type name.
func GetSingleItemByType(SingletonName string) (*models.SingleItem, error) {
	// First, find the single type
	var st models.Singleton
	if err := database.DB.Preload("Attributes").Where("name = ?", SingletonName).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("single type '%s' not found", SingletonName)
		}
		return nil, fmt.Errorf("failed to retrieve single type '%s': %w", SingletonName, err)
	}

	// Then, find the associated single item
	var item models.SingleItem
	if err := database.DB.Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No item yet => not found
			return nil, fmt.Errorf("no single item found for single type '%s'", SingletonName)
		}
		return nil, fmt.Errorf("failed to retrieve single item: %w", err)
	}

	return &item, nil
}

// UpdateSingleItem updates the SingleItem for a given single type with new
// data and records a revision (unless nothing changed). Returns the updated
// SingleItem or an error if not found or validation fails.
func UpdateSingleItem(actor Actor, SingletonName string, newData map[string]any) (*models.SingleItem, error) {
	st, err := singletonWithAttributes(SingletonName)
	if err != nil {
		return nil, err
	}
	if valErr := models.ValidateSingleItemValues(*st, newData); valErr != nil {
		logger.Log.WithError(valErr).WithField("Singleton", SingletonName).
			Warn("Validation failed for single item update")
		return nil, valErr
	}

	var item models.SingleItem
	changed := false
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockRow(tx).Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("no existing single item for single type '%s'", SingletonName)
			}
			return fmt.Errorf("failed to retrieve single item: %w", err)
		}
		previous := item.Data
		before := &entryState{data: previous, translations: item.Translations}
		item.Data = newData
		if err := tx.Save(&item).Error; err != nil {
			return fmt.Errorf("failed to update single item: %w", err)
		}
		if sameData(previous, item.Data) {
			return nil
		}
		changed = true
		_, err := recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntitySingleton, entityID: st.ID, typeName: st.Name,
			action: models.RevisionUpdate, data: item.Data, translations: item.Translations, before: before,
		})
		return err
	})
	if err != nil {
		logger.Log.WithError(err).WithField("Singleton", SingletonName).Warn("Failed to update single item")
		return nil, err
	}

	logger.Log.WithField("Singleton", SingletonName).Info("Single item updated successfully")
	if changed {
		publishSingletonEvent(actor, events.EventTypeSingletonUpdated, st.Name, &item)
	}
	return &item, nil
}

// DeleteSingleItem removes the SingleItem for a given single type name and
// records a "delete" revision holding its data, so it can be restored.
func DeleteSingleItem(actor Actor, SingletonName string) error {
	st, err := singletonWithAttributes(SingletonName)
	if err != nil {
		return err
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var item models.SingleItem
		if err := lockRow(tx).Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("no single item found for single type '%s'", SingletonName)
			}
			return fmt.Errorf("failed to retrieve single item: %w", err)
		}
		if _, err := recordRevision(tx, actor, revisionInput{
			entityType: models.RevisionEntitySingleton, entityID: st.ID, typeName: st.Name,
			action: models.RevisionDelete, data: item.Data, translations: item.Translations, before: &entryState{data: item.Data, translations: item.Translations},
		}); err != nil {
			return err
		}
		// A hard delete: the revision keeps the content, and the unique
		// index on single_type_id would block new content after a soft one.
		if err := tx.Unscoped().Delete(&item).Error; err != nil {
			return fmt.Errorf("failed to delete single item: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	logger.Log.WithField("Singleton", SingletonName).Info("Single item deleted successfully")
	publishSingletonEvent(actor, events.EventTypeSingletonDeleted, st.Name, nil)
	return nil
}

func singletonWithAttributes(name string) (*models.Singleton, error) {
	var st models.Singleton
	if err := database.DB.Preload("Attributes").Where("name = ?", name).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("single type '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to retrieve single type: %w", err)
	}
	return &st, nil
}

// publishSingletonEvent sends singleton:updated or singleton:deleted
// (webhooks), with the content (nil when deleted). The event's collection
// is "singleton:<name>".
func publishSingletonEvent(actor Actor, eventType events.EventType, name string, item *models.SingleItem) {
	if asynqClient == nil {
		return
	}
	payload := events.CollectionEventPayload{
		EventType: eventType, CollectionName: "singleton:" + name,
		ActorKind: actor.Kind, ActorName: actor.Name,
	}
	if item != nil {
		payload.ItemData = item.Data
		payload.ItemTranslations = copyTranslations(item.Translations)
	}
	if err := events.EnqueueCollectionEvent(context.Background(), asynqClient, payload); err != nil {
		logger.Log.WithError(err).WithField("singleton", name).Error("Failed to enqueue a singleton event")
	}
}
