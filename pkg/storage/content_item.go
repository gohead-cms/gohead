// pkg/storage/content_type.go
package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
)

func SaveContentItem(item *models.ContentItem) error {
	return database.DB.Create(item).Error
}

func GetContentItemByID(contentTypeName string, id uint) (*models.ContentItem, error) {
	var item models.ContentItem
	err := database.DB.Where("content_type = ? AND id = ?", contentTypeName, id).First(&item).Error
	if err != nil {
		return nil, fmt.Errorf("content item not found")
	}
	return &item, nil
}

func GetContentItems(contentTypeName string) ([]models.ContentItem, error) {
	var items []models.ContentItem
	err := database.DB.Where("content_type = ?", contentTypeName).Find(&items).Error
	return items, err
}

func UpdateContentItem(ct models.ContentType, id uint, data models.JSONMap) error {
	var item models.ContentItem
	err := database.DB.Where("content_type = ? AND id = ?", ct.Name, id).First(&item).Error
	if err != nil {
		return fmt.Errorf("content item not found")
	}

	item.Data = data
	if err := database.DB.Save(&item).Error; err != nil {
		return err
	}

	// Update relationships
	if err := database.DB.Where("content_type = ? AND content_item_id = ?", ct.Name, id).Delete(&models.ContentRelation{}).Error; err != nil {
		return err
	}

	if err := SaveContentRelations(&ct, id, data); err != nil {
		return err
	}

	return nil
}

func DeleteContentItem(ct models.ContentType, id uint) error {
	if err := database.DB.Where("content_type = ? AND id = ?", ct.Name, id).Delete(&models.ContentItem{}).Error; err != nil {
		return err
	}

	// Delete relationships
	if err := database.DB.Where("content_type = ? AND content_item_id = ?", ct.Name, id).Delete(&models.ContentRelation{}).Error; err != nil {
		return err
	}

	return nil
}
