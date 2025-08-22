package storage

import (
	"errors"
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// SaveOrUpdateSingleType either creates a new single type or updates/restores
// a previously soft-deleted record with the same name.
//
// This mimics the "single type" behavior in Strapi (one record per name).
func SaveOrUpdateSingleType(st *models.SingleType) error {
	var existing models.SingleType

	logger.Log.WithField("singleType", st.Name).Info("Attempting to save or update single type")

	// Check if a record with the same name exists (including soft-deleted)
	err := database.DB.Unscoped().Where("name = ?", st.Name).First(&existing).Error
	if err == nil {
		// A single type with this name exists
		if !existing.DeletedAt.Valid {
			// Not soft-deleted => we update the single type
			logger.Log.WithField("singleType", st.Name).Info("Single type exists, updating it")

			// Copy updatable fields; preserve ID & Timestamps
			existing.Description = st.Description

			// Start transaction to update attributes, etc.
			tx := database.DB.Begin()
			if err := updateSingleTypeAttributes(tx, &existing, st.Attributes); err != nil {
				logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to update attributes")
				tx.Rollback()
				return fmt.Errorf("failed to update single type attributes: %w", err)
			}

			if err := tx.Save(&existing).Error; err != nil {
				logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to save existing single type")
				tx.Rollback()
				return fmt.Errorf("failed to save existing single type: %w", err)
			}

			if err := tx.Commit().Error; err != nil {
				logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to commit transaction")
				return fmt.Errorf("failed to commit single type update transaction: %w", err)
			}

			logger.Log.WithField("singleType", st.Name).Info("Single type updated successfully")
			return nil
		}

		// The single type is soft-deleted; restore it
		logger.Log.WithField("singleType", st.Name).Info("Found soft-deleted single type, restoring")

		existing.DeletedAt.Valid = false // Clear the deleted_at flag
		if err := database.DB.Unscoped().Save(&existing).Error; err != nil {
			logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to restore single type")
			return fmt.Errorf("failed to restore single type: %w", err)
		}

		// Restore associated attributes
		if err := restoreAssociatedRecords(&models.Attribute{}, existing.ID); err != nil {
			logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to restore associated attributes")
			return fmt.Errorf("failed to restore associated attributes: %w", err)
		}

		logger.Log.WithField("singleType", st.Name).Info("Single type restored successfully")
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// An unexpected error occurred
		logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to check for existing single type")
		return fmt.Errorf("failed to check for existing single type: %w", err)
	}

	// If we reach here, record not found => create a new single type
	logger.Log.WithField("singleType", st.Name).Info("Creating new single type")

	tx := database.DB.Begin()
	if err := tx.Create(st).Error; err != nil {
		tx.Rollback()
		logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to create single type")
		return fmt.Errorf("failed to create single type: %w", err)
	}

	// Commit creation
	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to commit transaction for new single type")
		return fmt.Errorf("failed to commit transaction for new single type: %w", err)
	}

	logger.Log.WithField("singleType", st.Name).Info("Single type created successfully")
	return nil
}

// GetSingleTypeByName retrieves a single type by its name (including associated attributes).
func GetSingleTypeByName(name string) (*models.SingleType, error) {
	var st models.SingleType

	// Preload the 'Attributes' relationship
	if err := database.DB.
		Preload("Attributes").
		Where("name = ?", name).
		First(&st).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("name", name).
				Warn("single type not found")
			return nil, fmt.Errorf("single type '%s' not found", name)
		}

		logger.Log.WithField("name", name).
			Error("Failed to fetch single type", err)
		return nil, fmt.Errorf("failed to fetch single type '%s': %w", name, err)
	}

	logger.Log.WithField("singleType", st.Name).
		Info("Single type fetch successfully")
	return &st, nil
}

// DeleteSingleType permanently deletes a single type by its ID (including attributes).
// Note: Strapi typically doesn't expose DELETE for single types, but you can implement it if needed.
func DeleteSingleType(singleTypeID uint) error {
	tx := database.DB.Begin()
	logger.Log.WithField("single_type_id", singleTypeID).Info("DeleteSingleType: start deletion")

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Retrieve the single type
	var st models.SingleType
	if err := tx.Where("id = ?", singleTypeID).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			logger.Log.WithField("single_type_id", singleTypeID).Warn("DeleteSingleType: single type not found")
			return fmt.Errorf("single type with ID '%d' not found", singleTypeID)
		}
		tx.Rollback()
		return fmt.Errorf("failed to retrieve single type: %w", err)
	}

	logger.Log.WithField("singleType", st.Name).Info("DeleteSingleType: successfully retrieved single type")

	// Delete associated attributes
	if err := tx.Where("single_type_id = ?", singleTypeID).Delete(&models.Attribute{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete attributes for single type ID '%d': %w", singleTypeID, err)
	}
	logger.Log.WithField("single_type_id", singleTypeID).Info("DeleteSingleType: deleted associated attributes")

	// Delete the single type
	if err := tx.Delete(&st).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete single type with ID '%d': %w", singleTypeID, err)
	}

	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).WithField("singleType", st.Name).Error("Failed to commit transaction for single type deletion")
		return fmt.Errorf("failed to commit transaction for single type deletion: %w", err)
	}

	logger.Log.WithField("singleType", st.Name).Info("Single type deleted successfully")
	return nil
}

// restoreAssociatedRecords is reused from your existing code to restore soft-deleted attributes, etc.
// func restoreAssociatedRecords(model interface{}, parentID uint) error {
// 	return database.DB.Unscoped().
// 		Model(model).
// 		Where("single_type_id = ?", parentID).
// 		Update("deleted_at", nil).Error
// }

// updateSingleTypeAttributes handles creating/updating attributes for a single type within a transaction.
// Similar to `updateAssociatedFields` for collections, but keyed on `single_type_id`.
func updateSingleTypeAttributes(tx *gorm.DB, existing *models.SingleType, updatedAttributes []models.Attribute) error {
	// Fetch existing attributes
	var existingAttrs []models.Attribute
	if err := tx.Where("single_type_id = ?", existing.ID).Find(&existingAttrs).Error; err != nil {
		return fmt.Errorf("failed to fetch existing single type attributes: %w", err)
	}

	// Build a map of existing attributes by name
	existingMap := make(map[string]models.Attribute)
	for _, attr := range existingAttrs {
		existingMap[attr.Name] = attr
	}

	// Process updates & inserts
	for _, updatedAttr := range updatedAttributes {
		if existingAttr, ok := existingMap[updatedAttr.Name]; ok {
			// Update existing
			updatedAttr.ID = existingAttr.ID
			updatedAttr.SingleTypeID = &existing.ID
			updatedAttr.CollectionID = nil // ensure we don't mix collection vs single type
			if err := tx.Save(&updatedAttr).Error; err != nil {
				return fmt.Errorf("failed to update single type attribute '%s': %w", updatedAttr.Name, err)
			}
			delete(existingMap, updatedAttr.Name)
		} else {
			// Insert new
			updatedAttr.SingleTypeID = &existing.ID
			updatedAttr.CollectionID = nil
			if err := tx.Create(&updatedAttr).Error; err != nil {
				return fmt.Errorf("failed to create single type attribute '%s': %w", updatedAttr.Name, err)
			}
		}
	}

	// Delete remaining (old) attributes
	for _, oldAttr := range existingMap {
		if err := tx.Delete(&oldAttr).Error; err != nil {
			return fmt.Errorf("failed to delete old single type attribute '%s': %w", oldAttr.Name, err)
		}
	}

	return nil
}
