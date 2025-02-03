package storage

import (
	"errors"
	"fmt"

	"gohead/internal/models"
	"gohead/pkg/database"
	"gohead/pkg/logger"

	"gorm.io/gorm"
)

// SaveCollection persists a Collection to the database, handling both new and soft-deleted records.
func SaveCollection(ct *models.Collection) error {
	var existing models.Collection

	logger.Log.WithField("collection", ct.Name).Info("Attempting to save collection")

	// Check if a record with the same name already exists, including soft-deleted records
	err := database.DB.Unscoped().Where("name = ?", ct.Name).First(&existing).Error
	if err == nil {
		// Collection with the same name exists
		if !existing.DeletedAt.Valid {
			// The collection is not soft-deleted, so it's a conflict
			logger.Log.WithField("collection", ct.Name).Warn("Collection with the same name already exists")
			return fmt.Errorf("a collection with the name '%s' already exists", ct.Name)
		}

		// The collection is soft-deleted; restore it
		logger.Log.WithField("collection", ct.Name).Info("Found soft-deleted collection, restoring")

		// Restore the collection
		existing.DeletedAt.Valid = false // Clear the deleted_at flag
		if err := database.DB.Unscoped().Save(&existing).Error; err != nil {
			logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to restore collection")
			return fmt.Errorf("failed to restore collection: %w", err)
		}

		// Restore associated fields
		if err := restoreAssociatedRecords(&models.Attribute{}, existing.ID); err != nil {
			logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to restore associated attributes")
			return fmt.Errorf("failed to restore associated attributes: %w", err)
		}

		// Restore associated relationships
		// TO CHECK

		logger.Log.WithField("collection", ct.Name).Info("Collection restored successfully")
		return nil
	} else if err != gorm.ErrRecordNotFound {
		// An error occurred while checking for existing collections
		logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to check for existing collection")
		return fmt.Errorf("failed to check for existing collection: %w", err)
	}

	// No conflict, create a new collection
	logger.Log.WithField("collection", ct.Name).Info("Creating new collection")
	if err := database.DB.Create(ct).Error; err != nil {
		logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to create collection")
		return fmt.Errorf("failed to save collection: %w", err)
	}

	logger.Log.WithField("collection", ct.Name).Info("Collection created successfully")
	return nil
}

// restoreAssociatedRecords restores soft-deleted associated records (attributes, relationships, etc.).
func restoreAssociatedRecords(model interface{}, collectionID uint) error {
	return database.DB.Unscoped().
		Model(model).
		Where("collection_id = ?", collectionID).
		Update("deleted_at", nil).Error
}

// GetCollectionByName retrieves a collection by its name.
func GetCollectionByName(name string) (*models.Collection, error) {
	var ct models.Collection

	// Preload the 'Attributes' relationship
	if err := database.DB.
		Preload("Attributes").
		Where("name = ?", name).
		First(&ct).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("name", name).
				Warn("collection not found")
			return nil, fmt.Errorf("collection '%s' not found", name)
		}

		logger.Log.WithField("name", name).
			Error("Failed to fetch collection", err)
		return nil, fmt.Errorf("failed to fetch collection '%s': %w", name, err)
	}

	logger.Log.WithField("collection", ct.Name).
		Info("collection fetch successfully")
	return &ct, nil
}

// GetAllCollections retrieves all Collections.
func GetAllCollections() ([]models.Collection, error) {
	var cts []models.Collection
	if err := database.DB.Preload("attributes").Preload("Relationships").Find(&cts).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch collections: %w", err)
	}
	return cts, nil
}

// UpdateCollection updates an existing Collection in the database.
func UpdateCollection(name string, updated *models.Collection) error {
	var existing models.Collection

	logger.Log.WithField("collection input", updated).Info("Attempting to update collection in database")

	// Find the existing collection by name
	if err := database.DB.Preload("Attributes").
		Where("name = ?", name).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Log.WithField("collection", name).Warn("Collection not found for update")
			return fmt.Errorf("collection '%s' not found", name)
		}
		logger.Log.WithError(err).WithField("collection", name).Error("Failed to retrieve collection for update")
		return fmt.Errorf("failed to retrieve collection: %w", err)
	}

	// Update the basic attributes of the collection
	existing.Name = updated.Name

	// Start a transaction for updating associated attributes and relationships
	tx := database.DB.Begin()
	logger.Log.WithField("collection", name).Info("Begin transaction to update collection in database")
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Update fields
	if err := updateAssociatedFields(tx, &existing.ID, updated.Attributes); err != nil {
		logger.Log.WithField("collection", name).Error("Rollback failed to updated fields")
		tx.Rollback()
		return fmt.Errorf("failed to update fields: %w", err)
	}

	// Save the updated collection
	logger.Log.WithField("collection input", updated).Debug("Existing ")
	if err := tx.Save(&existing).Error; err != nil {
		logger.Log.WithError(err).WithField("collection", name).Error("Failed to save updated collection")
		tx.Rollback()
		return fmt.Errorf("failed to save updated collection: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).WithField("collection", name).Error("Failed to commit transaction for collection update")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func updateAssociatedFields(tx *gorm.DB, collectionID *uint, updatedAttributes []models.Attribute) error {
	// Fetch existing attributes
	var existingAttributes []models.Attribute
	if err := tx.Where("collection_id = ?", collectionID).Find(&existingAttributes).Error; err != nil {
		return fmt.Errorf("failed to fetch existing attributes: %w", err)
	}

	// Map existing attributes by name for comparison
	existingMap := make(map[string]models.Attribute)
	for _, attr := range existingAttributes {
		existingMap[attr.Name] = attr
	}

	// Process updates and inserts
	for _, updatedAttr := range updatedAttributes {
		if existingAttr, exists := existingMap[updatedAttr.Name]; exists {
			// Update existing attribute
			updatedAttr.ID = existingAttr.ID
			updatedAttr.CollectionID = collectionID
			if err := tx.Save(&updatedAttr).Error; err != nil {
				return fmt.Errorf("failed to update attribute '%s': %w", updatedAttr.Name, err)
			}
			// Remove from the map after processing
			delete(existingMap, updatedAttr.Name)
		} else {
			// Insert new attribute
			updatedAttr.CollectionID = collectionID
			if err := tx.Create(&updatedAttr).Error; err != nil {
				return fmt.Errorf("failed to insert attribute '%s': %w", updatedAttr.Name, err)
			}
		}
	}

	// Delete remaining attributes in the map
	for _, attr := range existingMap {
		if err := tx.Delete(&attr).Error; err != nil {
			return fmt.Errorf("failed to delete attribute '%s': %w", attr.Name, err)
		}
	}

	return nil
}

// DeleteCollection deletes a collection and all associated data by its ID.
func DeleteCollection(collectionID uint) error {
	// Begin a transaction for cascading deletion
	tx := database.DB.Begin()
	logger.Log.WithField("collection_id", collectionID).Info("DeleteCollection: start to delete collection")
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Retrieve the collection
	var Collection models.Collection
	if err := tx.Where("id = ?", collectionID).First(&Collection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Log.WithField("collection_id", collectionID).Error("DeleteCollection: failed to retreive collection")
			tx.Rollback()
			return fmt.Errorf("DeleteCollection: failed to retreive collection '%d' not found", collectionID)
		}
		tx.Rollback()
		return fmt.Errorf("failed to retrieve collection: %w", err)
	}
	logger.Log.WithField("collection", Collection).Info("DeleteCollection: successfully retrieve collection")

	// Delete associated fields
	if err := tx.Where("collection_id = ?", collectionID).Delete(&models.Attribute{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete fields for collection ID '%d': %w", collectionID, err)
	}
	logger.Log.WithField("collection_id", collectionID).Info("DeleteCollection: delete successfully associated fields")

	// Delete associated relationships
	// TOCHECK

	// Delete associated content items
	if err := tx.Where("collection_id = ?", Collection.ID).Delete(&models.Item{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete content items for collection '%s': %w", Collection.Name, err)
	}
	logger.Log.WithField("collection_id", collectionID).Info("DeleteCollection: delete successfully associated items")

	// Delete the collection itself
	if err := tx.Delete(&Collection).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete collection with ID '%d': %w", collectionID, err)
	}

	// Commit the transaction
	return tx.Commit().Error
}
