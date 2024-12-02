package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
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

// DeleteContentType deletes a ContentType by name.
func DeleteContentType(name string) error {
	if err := database.DB.Where("name = ?", name).Delete(&models.ContentType{}).Error; err != nil {
		return fmt.Errorf("failed to delete content type: %w", err)
	}
	return nil
}

// DeleteContentItemsByType deletes all content items for a given content type.
func DeleteContentItemsByType(contentTypeName string) error {
	return database.DB.Where("content_type = ?", contentTypeName).Delete(&models.ContentItem{}).Error
}

// DeleteFieldsByContentType deletes all fields associated with a given content type.
func DeleteFieldsByContentType(contentTypeName string) error {
	return database.DB.Where("content_type_name = ?", contentTypeName).Delete(&models.Field{}).Error
}
