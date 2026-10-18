package storage

import (
	"errors"
	"fmt"
	"strings"

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
		refs, err := ComponentReferences(tx, existing.Name)
		if err != nil {
			tx.Rollback()
			return err
		}
		if len(refs) > 0 {
			tx.Rollback()
			return fmt.Errorf("%w: cannot rename '%s', it is used by %s", ErrComponentInUse, existing.Name, strings.Join(refs, ", "))
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

// ComponentReferences lists the attributes (as "collection.attribute",
// "singleton.attribute" or "component.attribute") that use the component.
func ComponentReferences(tx *gorm.DB, name string) ([]string, error) {
	if tx == nil {
		tx = database.DB
	}
	var refs []string
	var attrs []models.Attribute
	if err := tx.Preload("Collection").Preload("Singleton").
		Where("type = ? AND component_ref = ?", "component", name).Find(&attrs).Error; err != nil {
		return nil, err
	}
	for _, a := range attrs {
		switch {
		case a.Collection != nil:
			refs = append(refs, a.Collection.Name+"."+a.Name)
		case a.Singleton != nil:
			refs = append(refs, a.Singleton.Name+"."+a.Name)
		}
	}
	var cattrs []models.ComponentAttribute
	if err := tx.Where("type = ? AND component_ref = ?", "component", name).Find(&cattrs).Error; err != nil {
		return nil, err
	}
	for _, a := range cattrs {
		var owner models.Component
		if err := tx.Select("name").First(&owner, a.ComponentID).Error; err == nil && owner.Name != name {
			refs = append(refs, owner.Name+"."+a.Name)
		}
	}
	return refs, nil
}

// ErrComponentInUse is returned when deleting or renaming a component that
// attributes still reference.
var ErrComponentInUse = errors.New("component is in use")

// DeleteComponent removes a component definition by name. It refuses to
// delete a component that attributes still use.
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
	refs, err := ComponentReferences(nil, name)
	if err != nil {
		return fmt.Errorf("failed to check references to component '%s': %w", name, err)
	}
	if len(refs) > 0 {
		return fmt.Errorf("%w: '%s' is used by %s", ErrComponentInUse, name, strings.Join(refs, ", "))
	}

	// The ON DELETE CASCADE constraint only fires on hard deletes (and only
	// when the database enforces foreign keys), so remove the attributes
	// explicitly. Definitions are hard-deleted so the name can be reused.
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("component_id = ?", cmp.ID).Delete(&models.ComponentAttribute{}).Error; err != nil {
			return fmt.Errorf("failed to delete attributes of component '%s': %w", name, err)
		}
		if err := tx.Unscoped().Delete(&cmp).Error; err != nil {
			return fmt.Errorf("failed to delete component '%s': %w", name, err)
		}
		return nil
	})
}

// GetAllComponents returns every component definition with its attributes,
// ordered by name.
func GetAllComponents() ([]models.Component, error) {
	var cmps []models.Component
	if err := database.DB.Preload("Attributes").Order("name ASC").Find(&cmps).Error; err != nil {
		return nil, fmt.Errorf("failed to list components: %w", err)
	}
	return cmps, nil
}
