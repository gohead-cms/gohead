package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
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

func GetItems(CollectionID uint) ([]models.Item, error) {
	var items []models.Item
	err := database.DB.Where("collection_id = ?", CollectionID).Find(&items).Error
	return items, err
}

func UpdateItem(collection *models.Collection, itemID uint, data models.JSONMap) error {
	// Fetch the existing item
	var item models.Item
	if err := database.DB.Where("id = ?", itemID).First(&item).Error; err != nil {
		logger.Log.WithField("item_id", itemID).WithError(err).Error("Failed to find item")
		return fmt.Errorf("collection item not found")
	}

	// Update the item data
	item.Data = data
	if err := database.DB.Save(&item).Error; err != nil {
		logger.Log.WithField("item_id", itemID).WithError(err).Error("Failed to update item data")
		return fmt.Errorf("failed to update item data: %w", err)
	}

	// Delete existing relationships
	if err := database.DB.Where("item_id = ?", itemID).Delete(&models.Relationship{}).Error; err != nil {
		logger.Log.WithField("item_id", itemID).WithError(err).Error("Failed to delete existing relationships")
		return fmt.Errorf("failed to delete existing relationships: %w", err)
	}

	// Save updated relationships
	if err := SaveRelationship(collection, item.ID, data); err != nil {
		logger.Log.WithField("item_id", itemID).WithError(err).Error("Failed to save updated relationships")
		return fmt.Errorf("failed to save updated relationships: %w", err)
	}

	logger.Log.WithField("item_id", itemID).Info("Item updated successfully")
	return nil
}

func DeleteItem(id uint) error {
	if err := database.DB.Where("id = ?", id).Delete(&models.Item{}).Error; err != nil {
		return err
	}

	// Delete relationships
	if err := database.DB.Where("collection = ? AND source_item_id = ?", id).Delete(&models.Relationship{}).Error; err != nil {
		return err
	}

	return nil
}
