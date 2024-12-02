package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gorm.io/gorm"
)

// SaveContentType persists a ContentType to the database.
func SaveContentType(ct *models.ContentType) error {
	if err := database.DB.Create(ct).Error; err != nil {
		return fmt.Errorf("failed to save content type: %w", err)
	}
	return nil
}

// GetContentType retrieves a ContentType by name.
func GetContentType(name string) (*models.ContentType, error) {
	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("Relationships").
		Where("name = ?", name).First(&ct).Error; err != nil {
		return nil, fmt.Errorf("content type not found: %w", err)
	}
	return &ct, nil
}

// GetContentTypeByName retrieves a content type by its name.
func GetContentTypeByName(name string) (*models.ContentType, error) {
	var contentType models.ContentType
	// Query the database for the content type by name
	if err := database.DB.Where("name = ?", name).First(&contentType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Handle case where the content type does not exist
			return nil, fmt.Errorf("content type '%s' not found", name)
		}
		// Handle general database errors
		return nil, fmt.Errorf("failed to retrieve content type: %w", err)
	}
	return &contentType, nil
}

// GetAllContentTypes retrieves all ContentTypes.
func GetAllContentTypes() ([]models.ContentType, error) {
	var cts []models.ContentType
	if err := database.DB.Preload("Fields").Preload("Relationships").Find(&cts).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch content types: %w", err)
	}
	return cts, nil
}

// UpdateContentType updates an existing ContentType.
func UpdateContentType(name string, updated *models.ContentType) error {
	var ct models.ContentType
	if err := database.DB.Where("name = ?", name).First(&ct).Error; err != nil {
		return fmt.Errorf("content type not found: %w", err)
	}

	// Update fields
	ct.Name = updated.Name
	ct.Fields = updated.Fields
	ct.Relationships = updated.Relationships

	if err := database.DB.Save(&ct).Error; err != nil {
		return fmt.Errorf("failed to update content type: %w", err)
	}
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
