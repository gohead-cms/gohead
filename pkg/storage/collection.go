package storage

import (
	"fmt"
	"time"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/gorm"
)

// SaveCollection persists a Collection to the database.
func SaveCollection(ct *models.Collection) error {
	var existing models.Collection

	logger.Log.WithField("content_type", ct.Name).Info("Attempting to save content type")

	// Check if a record with the same name already exists (including soft-deleted records)
	err := database.DB.Unscoped().Where("name = ?", ct.Name).First(&existing).Error
	if err == nil {
		// If an existing record is found
		if !existing.DeletedAt.Valid {
			logger.Log.WithField("content_type", ct.Name).Warn("Content type with the same name already exists")
			return fmt.Errorf("a content type with the name '%s' already exists", ct.Name)
		}

		// If the record is soft-deleted, restore it along with its associations
		logger.Log.WithField("content_type", ct.Name).Info("Found soft-deleted content type, restoring")

		// Restore the content type
		existing.DeletedAt.Time = time.Time{} // Clear the deleted_at timestamp
		existing.DeletedAt.Valid = false

		if saveErr := database.DB.Save(&existing).Error; saveErr != nil {
			logger.Log.WithError(saveErr).WithField("content_type", ct.Name).Error("Failed to restore existing content type")
			return fmt.Errorf("failed to restore existing content type: %w", saveErr)
		}

		// Restore associated fields
		if err := database.DB.Unscoped().Model(&models.Field{}).
			Where("content_type_id = ?", existing.ID).Update("deleted_at", nil).Error; err != nil {
			logger.Log.WithError(err).WithField("content_type", ct.Name).Error("Failed to restore associated fields")
			return fmt.Errorf("failed to restore associated fields: %w", err)
		}

		// Restore associated relationships
		if err := database.DB.Unscoped().Model(&models.Relationship{}).
			Where("content_type_id = ?", existing.ID).Update("deleted_at", nil).Error; err != nil {
			logger.Log.WithError(err).WithField("content_type", ct.Name).Error("Failed to restore associated relationships")
			return fmt.Errorf("failed to restore associated relationships: %w", err)
		}

		logger.Log.WithField("content_type", ct.Name).Info("Content type restored successfully")
		return nil
	} else if err != gorm.ErrRecordNotFound {
		// Log database error
		logger.Log.WithError(err).WithField("content_type", ct.Name).Error("Failed to check for existing content type")
		return fmt.Errorf("failed to check for existing content type: %w", err)
	}

	// No conflict, create a new record
	logger.Log.WithField("content_type", ct.Name).Info("Creating new content type")
	if createErr := database.DB.Create(ct).Error; createErr != nil {
		logger.Log.WithError(createErr).WithField("content_type", ct.Name).Error("Failed to create content type")
		return fmt.Errorf("failed to save content type: %w", createErr)
	}

	logger.Log.WithField("content_type", ct.Name).Info("Content type created successfully")
	return nil
}

// GetCollectionByName retrieves a content type by its name.
func GetCollectionByName(name string) (*models.Collection, error) {
	var ct models.Collection

	// Load the Collection, including its fields and relationships
	if err := database.DB.Preload("CollectionField").Preload("Relationships").
		Where("name = ?", name).First(&ct).Error; err != nil {
		if err.Error() == "record not found" {
			logger.Log.WithField("name", name).Warn("Content type not found")
			return nil, fmt.Errorf("content type '%s' not found", name)
		}
		logger.Log.WithField("name", name).Error("Failed to fetch content type", err)
		return nil, fmt.Errorf("failed to fetch content type '%s': %w", name, err)
	}
	logger.Log.WithField("content_type", ct.Name).Info("Content type fetch successfully")
	return &ct, nil
}

// GetAllCollections retrieves all Collections.
func GetAllCollections() ([]models.Collection, error) {
	var cts []models.Collection
	if err := database.DB.Preload("CollectionFields").Preload("Relationships").Find(&cts).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch content types: %w", err)
	}
	return cts, nil
}

// UpdateCollection updates an existing Collection in the database.
func UpdateCollection(name string, updated *models.Collection) error {
	var existing models.Collection

	logger.Log.WithField("content_type", name).Info("Attempting to update content type")

	// Find the existing content type by name
	if err := database.DB.Preload("CollectionFields").Preload("Relationships").
		Where("name = ?", name).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Log.WithField("content_type", name).Warn("Content type not found for update")
			return fmt.Errorf("content type '%s' not found", name)
		}
		logger.Log.WithError(err).WithField("content_type", name).Error("Failed to retrieve content type for update")
		return fmt.Errorf("failed to retrieve content type: %w", err)
	}

	// Update the basic fields of the content type
	existing.Name = updated.Name

	// Start a transaction for updating associated fields and relationships
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Update fields
	if err := updateAssociatedFields(tx, existing.ID, updated.Fields); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update fields: %w", err)
	}

	// Update relationships
	if err := updateAssociatedRelationships(tx, existing.ID, updated.Relationships); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update relationships: %w", err)
	}

	// Save the updated content type
	if err := tx.Save(&existing).Error; err != nil {
		tx.Rollback()
		logger.Log.WithError(err).WithField("content_type", name).Error("Failed to save updated content type")
		return fmt.Errorf("failed to save updated content type: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).WithField("content_type", name).Error("Failed to commit transaction for content type update")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Log.WithField("content_type", name).Info("Content type updated successfully")
	return nil
}

// updateAssociatedFields updates or creates fields for a content type.
func updateAssociatedFields(tx *gorm.DB, CollectionID uint, fields []models.Field) error {
	// Soft-delete existing fields
	if err := tx.Where("content_type_id = ?", CollectionID).Delete(&models.Field{}).Error; err != nil {
		logger.Log.WithError(err).WithField("content_type_id", CollectionID).Error("Failed to soft-delete fields")
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

	logger.Log.WithField("content_type_id", CollectionID).Info("Fields updated successfully")
	return nil
}

// updateAssociatedRelationships updates or creates relationships for a content type.
func updateAssociatedRelationships(tx *gorm.DB, CollectionID uint, relationships []models.Relationship) error {
	// Soft-delete existing relationships
	if err := tx.Where("content_type_id = ?", CollectionID).Delete(&models.Relationship{}).Error; err != nil {
		logger.Log.WithError(err).WithField("content_type_id", CollectionID).Error("Failed to soft-delete relationships")
		return fmt.Errorf("failed to soft-delete relationships: %w", err)
	}

	// Save the new relationships
	for _, relationship := range relationships {
		relationship.ID = CollectionID
		if err := tx.Save(&relationship).Error; err != nil {
			logger.Log.WithError(err).WithField("field_name", relationship.FieldName).Error("Failed to save relationship")
			return fmt.Errorf("failed to save relationship '%s': %w", relationship.FieldName, err)
		}
	}

	logger.Log.WithField("content_type_id", CollectionID).Info("Relationships updated successfully")
	return nil
}

// DeleteCollection deletes a content type and all associated data by its ID.
func DeleteCollection(CollectionID uint) error {
	// Begin a transaction for cascading deletion
	tx := database.DB.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Retrieve the content type
	var Collection models.Collection
	if err := tx.Where("id = ?", CollectionID).First(&Collection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("content type with ID '%d' not found", CollectionID)
		}
		tx.Rollback()
		return fmt.Errorf("failed to retrieve content type: %w", err)
	}

	// Delete associated fields
	if err := tx.Where("content_type_id = ?", CollectionID).Delete(&models.Field{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete fields for content type ID '%d': %w", CollectionID, err)
	}

	// Delete associated relationships
	if err := tx.Where("content_type_id = ?", CollectionID).Delete(&models.Relationship{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete relationships for content type ID '%d': %w", CollectionID, err)
	}

	// Delete associated content items
	if err := tx.Where("content_type = ?", Collection.Name).Delete(&models.Item{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete content items for content type '%s': %w", Collection.Name, err)
	}

	// Delete the content type itself
	if err := tx.Delete(&Collection).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete content type with ID '%d': %w", CollectionID, err)
	}

	// Commit the transaction
	return tx.Commit().Error
}
