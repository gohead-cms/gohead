package storage

import (
	"errors"
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// CreateSingleItem creates a new SingleItem for a given Singleton.
// Returns an error if an item already exists for that Singleton (one-to-one).
func CreateSingleItem(st *models.Singleton, itemData map[string]any) (*models.SingleItem, error) {
	// Check if a single item already exists for this single type
	var existing models.SingleItem
	err := database.DB.Where("single_type_id = ?", st.ID).First(&existing).Error
	if err == nil {
		// Found an existing record => conflict
		errMsg := fmt.Sprintf("a single item already exists for single type '%s'", st.Name)
		logger.Log.WithField("Singleton", st.Name).Warn(errMsg)
		return nil, errors.New(errMsg)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Some unexpected DB error
		logger.Log.WithError(err).WithField("Singleton", st.Name).
			Error("Failed to check for existing single item")
		return nil, err
	}

	// Validate the new item data
	if valErr := models.ValidateSingleItemValues(*st, itemData); valErr != nil {
		logger.Log.WithError(valErr).WithField("Singleton", st.Name).
			Warn("Validation failed for single item creation")
		return nil, valErr
	}

	// If validation passes, create the item
	item := &models.SingleItem{
		SingleTypeID: st.ID,
		Data:         itemData,
	}
	if createErr := database.DB.Create(item).Error; createErr != nil {
		logger.Log.WithError(createErr).WithField("Singleton", st.Name).
			Error("Failed to create single item in DB")
		return nil, fmt.Errorf("failed to create single item: %w", createErr)
	}

	logger.Log.WithField("Singleton", st.Name).Info("Single item created successfully")
	return item, nil
}

// GetSingleItemByType retrieves the SingleItem for a given single type name.
func GetSingleItemByType(SingletonName string) (*models.SingleItem, error) {
	// First, find the single type
	var st models.Singleton
	if err := database.DB.Where("name = ?", SingletonName).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("single type '%s' not found", SingletonName)
		}
		return nil, fmt.Errorf("failed to retrieve single type '%s': %w", SingletonName, err)
	}

	// Then, find the associated single item
	var item models.SingleItem
	if err := database.DB.Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No item yet => not found
			return nil, fmt.Errorf("no single item found for single type '%s'", SingletonName)
		}
		return nil, fmt.Errorf("failed to retrieve single item: %w", err)
	}

	return &item, nil
}

// UpdateSingleItem updates the SingleItem for a given single type with new data.
// Returns the updated SingleItem or an error if not found or validation fails.
func UpdateSingleItem(SingletonName string, newData map[string]any) (*models.SingleItem, error) {
	// Retrieve the single type
	var st models.Singleton
	if err := database.DB.Where("name = ?", SingletonName).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("single type '%s' not found", SingletonName)
		}
		return nil, fmt.Errorf("failed to retrieve single type: %w", err)
	}

	// Retrieve the existing single item
	var item models.SingleItem
	if err := database.DB.Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no existing single item for single type '%s'", SingletonName)
		}
		return nil, fmt.Errorf("failed to retrieve single item: %w", err)
	}

	// Validate new data
	if valErr := models.ValidateSingleItemValues(st, newData); valErr != nil {
		logger.Log.WithError(valErr).WithField("Singleton", SingletonName).
			Warn("Validation failed for single item update")
		return nil, valErr
	}

	// Update
	item.Data = newData
	if saveErr := database.DB.Save(&item).Error; saveErr != nil {
		logger.Log.WithError(saveErr).WithField("Singleton", SingletonName).
			Error("Failed to save updated single item in DB")
		return nil, fmt.Errorf("failed to update single item: %w", saveErr)
	}

	logger.Log.WithField("Singleton", SingletonName).Info("Single item updated successfully")
	return &item, nil
}

// DeleteSingleItem removes the SingleItem for a given single type name (if it exists).
func DeleteSingleItem(SingletonName string) error {
	// Find the single type
	var st models.Singleton
	if err := database.DB.Where("name = ?", SingletonName).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("single type '%s' not found", SingletonName)
		}
		return fmt.Errorf("failed to retrieve single type: %w", err)
	}

	// Find the single item
	var item models.SingleItem
	if err := database.DB.Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("no single item found for single type '%s'", SingletonName)
		}
		return fmt.Errorf("failed to retrieve single item: %w", err)
	}

	// Delete the item
	if err := database.DB.Delete(&item).Error; err != nil {
		logger.Log.WithError(err).WithField("Singleton", SingletonName).
			Error("Failed to delete single item from DB")
		return fmt.Errorf("failed to delete single item: %w", err)
	}

	logger.Log.WithField("Singleton", SingletonName).Info("Single item deleted successfully")
	return nil
}
