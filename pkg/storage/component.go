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

	// 4) Create the component record. GORM will automatically handle the nested attributes
	// because of the foreignKey and constraint tags in the model struct.
	if err := tx.Create(cmp).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create component '%s': %w", cmp.Name, err)
	}

	// 5) Commit the transaction
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
	// Validate schema of the incoming data
	if err := models.ValidateComponentSchema(*updated); err != nil {
		return err
	}

	// Begin a transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction for update: %w", tx.Error)
	}

	// Fetch the existing record within the transaction
	var existing models.Component
	err := tx.Preload("Attributes").
		Where("name = ?", name).First(&existing).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find component '%s': %w", name, err)
	}

	// If the name is being changed, ensure no conflict
	if updated.Name != "" && updated.Name != existing.Name {
		var conflict models.Component
		if err := tx.Where("name = ?", updated.Name).First(&conflict).Error; err == nil {
			tx.Rollback()
			return fmt.Errorf("cannot rename component to '%s': name already exists", updated.Name)
		} else if err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return err
		}
		existing.Name = updated.Name // Update the name
	}

	// Update description if provided
	existing.Description = updated.Description

	// Handle attribute updates: create, update, delete
	if err := updateComponentAttributes(tx, &existing, updated.Attributes); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update component attributes: %w", err)
	}

	// Save the parent component changes
	if err := tx.Save(&existing).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save updated component: %w", err)
	}

	return tx.Commit().Error
}

// updateComponentAttributes merges the updated attributes with existing ones.
func updateComponentAttributes(tx *gorm.DB, existing *models.Component, updatedAttrs []models.ComponentAttribute) error {
	// Index existing attributes by name for easy lookup
	existingMap := make(map[string]models.ComponentAttribute)
	for _, attr := range existing.Attributes {
		existingMap[attr.Name] = attr
	}

	// Process incoming attributes
	for _, newAttr := range updatedAttrs {
		if oldAttr, ok := existingMap[newAttr.Name]; ok {
			// Attribute exists, so update it
			// GORM's Save will update if the primary key is set
			newAttr.ID = oldAttr.ID
			newAttr.ComponentID = existing.ID
			if err := tx.Save(&newAttr).Error; err != nil {
				return err
			}
			// Remove from map to track which ones are left to be deleted
			delete(existingMap, newAttr.Name)
		} else {
			// Attribute is new, so create it
			newAttr.ComponentID = existing.ID
			if err := tx.Create(&newAttr).Error; err != nil {
				return err
			}
		}
	}

	// Any attributes left in existingMap were not in the update, so delete them
	for _, attrToDelete := range existingMap {
		if err := tx.Delete(&attrToDelete).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteComponent removes a component definition by name.
// GORM's cascading delete will handle the attributes.
func DeleteComponent(name string) error {
	var cmp models.Component
	// Using .Select("ID") is efficient as we only need the ID for the delete operation.
	err := database.DB.Where("name = ?", name).Select("ID").First(&cmp).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("component '%s' not found", name)
		}
		return fmt.Errorf("failed to find component '%s': %w", name, err)
	}

	// GORM will automatically delete associated ComponentAttributes due to the
	// `constraint:OnDelete:CASCADE` tag in the Component model.
	if err := database.DB.Delete(&cmp).Error; err != nil {
		return fmt.Errorf("failed to delete component '%s': %w", name, err)
	}

	return nil
}
