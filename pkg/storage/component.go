// internal/storage/component.go
package storage

import (
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"

	"gorm.io/gorm"
)

// CreateComponent stores a new Component definition in the DB, including its attributes.
func CreateComponent(cmp *models.Component) error {
	// 1) Validate the component schema
	if err := models.ValidateComponentSchema(*cmp); err != nil {
		return err
	}

	// 2) Check if the name is already taken
	var existing models.Component
	err := database.DB.Where("name = ?", cmp.Name).First(&existing).Error
	if err == nil {
		return fmt.Errorf("component with name '%s' already exists", cmp.Name)
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	// 3) Begin a transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// 4) Create the component record
	if err := tx.Create(cmp).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create component '%s': %w", cmp.Name, err)
	}

	// 5) Create each attribute, ensuring ComponentID is set
	for i := range cmp.Attributes {
		cmp.Attributes[i].ComponentID = &cmp.ID
		if err := tx.Create(&cmp.Attributes[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create attribute '%s' for component '%s': %w",
				cmp.Attributes[i].Name, cmp.Name, err)
		}
	}

	// 6) Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction for component '%s': %w", cmp.Name, err)
	}

	return nil
}

// GetComponentByName retrieves a component definition by its name.
func GetComponentByName(name string) (*models.Component, error) {
	var cmp models.Component
	err := database.DB.Preload("Attributes").
		Where("name = ?", name).
		First(&cmp).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("component '%s' not found", name)
		}
		return nil, err
	}
	return &cmp, nil
}

// UpdateComponent updates an existing component definition by name.
func UpdateComponent(name string, updated *models.Component) error {
	// Validate schema
	if err := models.ValidateComponentSchema(*updated); err != nil {
		return err
	}

	// Fetch the existing record
	var existing models.Component
	err := database.DB.Preload("Attributes").
		Where("name = ?", name).First(&existing).Error
	if err != nil {
		return fmt.Errorf("failed to find component '%s': %w", name, err)
	}

	// If the updated name is different, ensure no conflict
	if updated.Name != "" && updated.Name != existing.Name {
		var conflict models.Component
		err := database.DB.Where("name = ?", updated.Name).First(&conflict).Error
		if err == nil {
			return fmt.Errorf("cannot rename component to '%s': already exists", updated.Name)
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
	}

	// Start transaction for updating attributes
	tx := database.DB.Begin()

	// Update basic fields
	if updated.Name != "" {
		existing.Name = updated.Name
	}

	// Update attributes: similar to your existing “updateAssociatedFields”
	if err := updateComponentAttributes(tx, &existing, updated.Attributes); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update component attributes: %w", err)
	}

	if err := tx.Save(&existing).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save updated component: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// updateComponentAttributes merges the updated attributes
// with existing ones, handling insert/update/delete.
func updateComponentAttributes(tx *gorm.DB, existing *models.Component, updatedAttrs []models.Attribute) error {
	// Fetch existing attributes
	var current []models.Attribute
	if err := tx.Where("component_id = ?", existing.ID).Find(&current).Error; err != nil {
		return err
	}

	// Index existing
	existingMap := make(map[string]models.Attribute)
	for _, a := range current {
		existingMap[a.Name] = a
	}

	for _, newAttr := range updatedAttrs {
		if oldAttr, ok := existingMap[newAttr.Name]; ok {
			// update
			newAttr.ID = oldAttr.ID // preserve ID
			newAttr.ComponentID = &existing.ID
			// save
			if err := tx.Save(&newAttr).Error; err != nil {
				return err
			}
			delete(existingMap, newAttr.Name)
		} else {
			// create
			newAttr.ComponentID = &existing.ID
			if err := tx.Create(&newAttr).Error; err != nil {
				return err
			}
		}
	}

	// delete leftover
	for _, attr := range existingMap {
		if err := tx.Delete(&attr).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteComponent removes a component definition by name, along with its attributes.
func DeleteComponent(name string) error {
	var cmp models.Component
	err := database.DB.Where("name = ?", name).First(&cmp).Error
	if err != nil {
		return fmt.Errorf("failed to find component '%s': %w", name, err)
	}

	tx := database.DB.Begin()
	if err := tx.Where("component_id = ?", cmp.ID).Delete(&models.Attribute{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete attributes for component '%s': %w", name, err)
	}

	if err := tx.Delete(&cmp).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete component '%s': %w", name, err)
	}
	return tx.Commit().Error
}
