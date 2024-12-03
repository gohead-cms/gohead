package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
)

func SaveItem(item *models.Item) error {
	return database.DB.Create(item).Error
}

func GetItemByID(id uint) (*models.Item, error) {
	var item models.Item
	err := database.DB.Where("id = ?", id).First(&item).Error
	if err != nil {
		return nil, fmt.Errorf("content item not found")
	}
	return &item, nil
}

func GetItems(CollectionName string) ([]models.Item, error) {
	var items []models.Item
	err := database.DB.Where("collection = ?", CollectionName).Find(&items).Error
	return items, err
}

func UpdateItem(ct models.Collection, id uint, data models.JSONMap) error {
	var item models.Item
	err := database.DB.Where("collection = ? AND id = ?", ct.Name, id).First(&item).Error
	if err != nil {
		return fmt.Errorf("content item not found")
	}

	item.Data = data
	if err := database.DB.Save(&item).Error; err != nil {
		return err
	}

	// Update relationships
	if err := database.DB.Where("collection = ? AND item_id = ?", ct.Name, id).Delete(&models.Relationship{}).Error; err != nil {
		return err
	}

	if err := SaveRelationship(&ct, id, data); err != nil {
		return err
	}

	return nil
}

func DeleteItem(ct models.Collection, id uint) error {
	if err := database.DB.Where("collection = ? AND id = ?", ct.Name, id).Delete(&models.Item{}).Error; err != nil {
		return err
	}

	// Delete relationships
	if err := database.DB.Where("collection = ? AND item_id = ?", ct.Name, id).Delete(&models.Relationship{}).Error; err != nil {
		return err
	}

	return nil
}
