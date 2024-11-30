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

func GetContentItems(contentTypeName string) ([]models.ContentItem, error) {
	var items []models.ContentItem
	err := database.DB.Where("content_type = ?", contentTypeName).Find(&items).Error
	return items, err
}

func GetContentItemByID(contentTypeName string, id uint) (*models.ContentItem, error) {
	var item models.ContentItem
	err := database.DB.Where("content_type = ? AND id = ?", contentTypeName, id).First(&item).Error
	if err != nil {
		return nil, fmt.Errorf("content item not found")
	}
	return &item, nil
}

func UpdateContentItem(contentTypeName string, id uint, data map[string]interface{}) error {
	var item models.ContentItem
	err := database.DB.Where("content_type = ? AND id = ?", contentTypeName, id).First(&item).Error
	if err != nil {
		return fmt.Errorf("content item not found")
	}

	item.Data = data
	return database.DB.Save(&item).Error
}

func DeleteContentItem(contentTypeName string, id uint) error {
	return database.DB.Where("content_type = ? AND id = ?", contentTypeName, id).Delete(&models.ContentItem{}).Error
}
