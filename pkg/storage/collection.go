package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
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
		if err := restoreAssociatedRecords(&models.Relationship{}, existing.ID); err != nil {
			logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to restore associated relationships")
			return fmt.Errorf("failed to restore associated relationships: %w", err)
		}

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

	// Load the Collection, including its fields and relationships
	if err := database.DB.Preload("attributes").Preload("relationships").
		Where("name = ?", name).First(&ct).Error; err != nil {
		if err.Error() == "record not found" {
			logger.Log.WithField("name", name).Warn("collection not found")
			return nil, fmt.Errorf("collection '%s' not found", name)
		}
		logger.Log.WithField("name", name).Error("Failed to fetch collection", err)
		return nil, fmt.Errorf("failed to fetch collection '%s': %w", name, err)
	}
	logger.Log.WithField("collection", ct.Name).Info("collection fetch successfully")
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

	logger.Log.WithField("collection", name).Info("Attempting to update collection in database")

	// Find the existing collection by name
	if err := database.DB.Preload("Fields").Preload("Relationships").
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
	if err := updateAssociatedFields(tx, existing.ID, updated.Attributes); err != nil {
		logger.Log.WithField("collection", name).Error("Rollback failed to updated fields")
		tx.Rollback()
		return fmt.Errorf("failed to update fields: %w", err)
	}

	// Update relationships
	// if err := updateAssociatedRelationships(tx, existing.ID, updated.Relationships); err != nil {
	// 	logger.Log.WithField("collection", name).Error("Rollback failed to updated relationships")
	// 	tx.Rollback()
	// 	return fmt.Errorf("failed to update relationships: %w", err)
	// }

	// Save the updated collection
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

// updateAssociatedFields updates or creates fields for a collection.
func updateAssociatedFields(tx *gorm.DB, CollectionID uint, fields []models.Attribute) error {
	// Soft-delete existing fields
	if err := tx.Where("collection_id = ?", CollectionID).Delete(&models.Attribute{}).Error; err != nil {
		logger.Log.WithError(err).WithField("collection_id", CollectionID).Error("Failed to soft-delete fields")
		return fmt.Errorf("failed to soft-delete fields: %w", err)
	}

	// Save the new fields
	for _, field := range fields {
		field.CollectionID = CollectionID
		if err := tx.Save(&field).Error; err != nil {
			logger.Log.WithError(err).WithField("field_name", field.Name).Error("Failed to save field")
			return fmt.Errorf("failed to save field '%s': %w", field.Name, err)
		}
	}

	logger.Log.WithField("collection_id", CollectionID).Info("Fields updated successfully")
	return nil
}

// updateAssociatedRelationships updates or creates relationships for a collection.
// func updateAssociatedRelationships(tx *gorm.DB, CollectionID uint, relationships []models.Relationship) error {
// 	// Soft-delete existing relationships
// 	if err := tx.Where("collection_id = ?", CollectionID).Delete(&models.Relationship{}).Error; err != nil {
// 		logger.Log.WithError(err).WithField("collection_id", CollectionID).Error("Failed to soft-delete relationships")
// 		return fmt.Errorf("failed to soft-delete relationships: %w", err)
// 	} else {
// 		logger.Log.WithField("collection", CollectionID).Error("Soft-delete relationships successfully")
// 	}

// 	// Save the new relationships
// 	for _, relationship := range relationships {
// 		relationship.CollectionID = CollectionID
// 		if err := tx.Save(&relationship).Error; err != nil {
// 			logger.Log.WithError(err).WithField("field_name", relationship.Field).Error("Failed to save relationship")
// 			return fmt.Errorf("failed to save relationship '%s': %w", relationship.Field, err)
// 		}
// 	}

// 	logger.Log.WithField("collection_id", CollectionID).Info("Relationships updated successfully")
// 	return nil
// }

// DeleteCollection deletes a collection and all associated data by its ID.
func DeleteCollection(CollectionID uint) error {
	// Begin a transaction for cascading deletion
	tx := database.DB.Begin()
	logger.Log.WithField("collection_id", CollectionID).Info("DeleteCollection: start to delete collection")
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Retrieve the collection
	var Collection models.Collection
	if err := tx.Where("id = ?", CollectionID).First(&Collection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Log.WithField("collection_id", CollectionID).Error("DeleteCollection: failed to retreive collection")
			tx.Rollback()
			return fmt.Errorf("DeleteCollection: failed to retreive collection '%d' not found", CollectionID)
		}
		tx.Rollback()
		return fmt.Errorf("failed to retrieve collection: %w", err)
	}
	logger.Log.WithField("collection", Collection).Info("DeleteCollection: successfully retrieve collection")

	// Delete associated fields
	if err := tx.Where("collection_id = ?", CollectionID).Delete(&models.Attribute{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete fields for collection ID '%d': %w", CollectionID, err)
	}
	logger.Log.WithField("collection_id", CollectionID).Info("DeleteCollection: delete successfully associated fields")

	// Delete associated relationships
	if err := tx.Where("collection_id = ?", CollectionID).Delete(&models.Relationship{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete relationships for collection ID '%d': %w", CollectionID, err)
	}
	logger.Log.WithField("collection_id", CollectionID).Info("DeleteCollection: delete successfully associated relationships")

	// Delete associated content items
	if err := tx.Where("collection_id = ?", Collection.ID).Delete(&models.Item{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete content items for collection '%s': %w", Collection.Name, err)
	}
	logger.Log.WithField("collection_id", CollectionID).Info("DeleteCollection: delete successfully associated items")

	// Delete the collection itself
	if err := tx.Delete(&Collection).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete collection with ID '%d': %w", CollectionID, err)
	}

	// Commit the transaction
	return tx.Commit().Error
}
