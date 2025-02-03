package storage

import (
	"errors"
	"fmt"

	"gohead/internal/models"
	"gohead/pkg/database"
	"gohead/pkg/logger"

	"gorm.io/gorm"
)

// CreateSingleItem creates a new SingleItem for a given SingleType.
// Returns an error if an item already exists for that SingleType (one-to-one).
func CreateSingleItem(st *models.SingleType, itemData map[string]interface{}) (*models.SingleItem, error) {
	// Check if a single item already exists for this single type
	var existing models.SingleItem
	err := database.DB.Where("single_type_id = ?", st.ID).First(&existing).Error
	if err == nil {
		// Found an existing record => conflict
		errMsg := fmt.Sprintf("a single item already exists for single type '%s'", st.Name)
		logger.Log.WithField("singleType", st.Name).Warn(errMsg)
		return nil, errors.New(errMsg)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Some unexpected DB error
		logger.Log.WithError(err).WithField("singleType", st.Name).
			Error("Failed to check for existing single item")
		return nil, err
	}

	// Validate the new item data
	if valErr := models.ValidateSingleItemValues(*st, itemData); valErr != nil {
		logger.Log.WithError(valErr).WithField("singleType", st.Name).
			Warn("Validation failed for single item creation")
		return nil, valErr
	}

	// If validation passes, create the item
	item := &models.SingleItem{
		SingleTypeID: st.ID,
		Data:         itemData,
	}
	if createErr := database.DB.Create(item).Error; createErr != nil {
		logger.Log.WithError(createErr).WithField("singleType", st.Name).
			Error("Failed to create single item in DB")
		return nil, fmt.Errorf("failed to create single item: %w", createErr)
	}

	logger.Log.WithField("singleType", st.Name).Info("Single item created successfully")
	return item, nil
}

// GetSingleItemByType retrieves the SingleItem for a given single type name.
func GetSingleItemByType(singleTypeName string) (*models.SingleItem, error) {
	// First, find the single type
	var st models.SingleType
	if err := database.DB.Where("name = ?", singleTypeName).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("single type '%s' not found", singleTypeName)
		}
		return nil, fmt.Errorf("failed to retrieve single type '%s': %w", singleTypeName, err)
	}

	// Then, find the associated single item
	var item models.SingleItem
	if err := database.DB.Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No item yet => not found
			return nil, fmt.Errorf("no single item found for single type '%s'", singleTypeName)
		}
		return nil, fmt.Errorf("failed to retrieve single item: %w", err)
	}

	return &item, nil
}

// UpdateSingleItem updates the SingleItem for a given single type with new data.
// Returns the updated SingleItem or an error if not found or validation fails.
func UpdateSingleItem(singleTypeName string, newData map[string]interface{}) (*models.SingleItem, error) {
	// Retrieve the single type
	var st models.SingleType
	if err := database.DB.Where("name = ?", singleTypeName).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("single type '%s' not found", singleTypeName)
		}
		return nil, fmt.Errorf("failed to retrieve single type: %w", err)
	}

	// Retrieve the existing single item
	var item models.SingleItem
	if err := database.DB.Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no existing single item for single type '%s'", singleTypeName)
		}
		return nil, fmt.Errorf("failed to retrieve single item: %w", err)
	}

	// Validate new data
	if valErr := models.ValidateSingleItemValues(st, newData); valErr != nil {
		logger.Log.WithError(valErr).WithField("singleType", singleTypeName).
			Warn("Validation failed for single item update")
		return nil, valErr
	}

	// Update
	item.Data = newData
	if saveErr := database.DB.Save(&item).Error; saveErr != nil {
		logger.Log.WithError(saveErr).WithField("singleType", singleTypeName).
			Error("Failed to save updated single item in DB")
		return nil, fmt.Errorf("failed to update single item: %w", saveErr)
	}

	logger.Log.WithField("singleType", singleTypeName).Info("Single item updated successfully")
	return &item, nil
}

// DeleteSingleItem removes the SingleItem for a given single type name (if it exists).
func DeleteSingleItem(singleTypeName string) error {
	// Find the single type
	var st models.SingleType
	if err := database.DB.Where("name = ?", singleTypeName).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("single type '%s' not found", singleTypeName)
		}
		return fmt.Errorf("failed to retrieve single type: %w", err)
	}

	// Find the single item
	var item models.SingleItem
	if err := database.DB.Where("single_type_id = ?", st.ID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("no single item found for single type '%s'", singleTypeName)
		}
		return fmt.Errorf("failed to retrieve single item: %w", err)
	}

	// Delete the item
	if err := database.DB.Delete(&item).Error; err != nil {
		logger.Log.WithError(err).WithField("singleType", singleTypeName).
			Error("Failed to delete single item from DB")
		return fmt.Errorf("failed to delete single item: %w", err)
	}

	logger.Log.WithField("singleType", singleTypeName).Info("Single item deleted successfully")
	return nil
}
