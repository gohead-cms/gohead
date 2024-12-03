package storage

import (
	"fmt"
	"time"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/gorm"
)

// SaveContentType persists a ContentType to the database.
func SaveContentType(ct *models.ContentType) error {
	var existing models.ContentType

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

// GetContentTypeByName retrieves a content type by its name.
func GetContentTypeByName(name string) (*models.ContentType, error) {
	var ct models.ContentType

	// Load the ContentType, including its fields and relationships
	if err := database.DB.Preload("Fields").Preload("Relationships").
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

// GetAllContentTypes retrieves all ContentTypes.
func GetAllContentTypes() ([]models.ContentType, error) {
	var cts []models.ContentType
	if err := database.DB.Preload("Fields").Preload("Relationships").Find(&cts).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch content types: %w", err)
	}
	return cts, nil
}

func UpdateContentType(name string, updated *models.ContentType) error {
	var existing models.ContentType

	logger.Log.WithField("content_type", name).Info("Attempting to update content type")

	// Check if the content type exists
	err := database.DB.Where("name = ?", name).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Log.WithField("content_type", name).Warn("Content type not found for update")
			return fmt.Errorf("content type '%s' not found", name)
		}
		logger.Log.WithError(err).WithField("content_type", name).Error("Failed to fetch content type for update")
		return fmt.Errorf("failed to fetch content type: %w", err)
	}

	// Perform the update
	logger.Log.WithField("content_type", name).Info("Updating content type details")
	existing.Name = updated.Name
	existing.Fields = updated.Fields
	existing.Relationships = updated.Relationships

	if saveErr := database.DB.Save(&existing).Error; saveErr != nil {
		logger.Log.WithError(saveErr).WithField("content_type", name).Error("Failed to update content type")
		return fmt.Errorf("failed to update content type: %w", saveErr)
	}

	logger.Log.WithField("content_type", name).Info("Content type updated successfully")
	return nil
}

// DeleteContentType deletes a content type and all associated data by its ID.
func DeleteContentType(contentTypeID uint) error {
	// Begin a transaction for cascading deletion
	tx := database.DB.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Retrieve the content type
	var contentType models.ContentType
	if err := tx.Where("id = ?", contentTypeID).First(&contentType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("content type with ID '%d' not found", contentTypeID)
		}
		tx.Rollback()
		return fmt.Errorf("failed to retrieve content type: %w", err)
	}

	// Delete associated fields
	if err := tx.Where("content_type_id = ?", contentTypeID).Delete(&models.Field{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete fields for content type ID '%d': %w", contentTypeID, err)
	}

	// Delete associated relationships
	if err := tx.Where("content_type_id = ?", contentTypeID).Delete(&models.Relationship{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete relationships for content type ID '%d': %w", contentTypeID, err)
	}

	// Delete associated content items
	if err := tx.Where("content_type = ?", contentType.Name).Delete(&models.ContentItem{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete content items for content type '%s': %w", contentType.Name, err)
	}

	// Delete the content type itself
	if err := tx.Delete(&contentType).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete content type with ID '%d': %w", contentTypeID, err)
	}

	// Commit the transaction
	return tx.Commit().Error
}
