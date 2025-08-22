package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/utils"

	"gorm.io/gorm"
)

// SaveCollection persists a Collection to the database, handling both new and soft-deleted records.
func SaveCollection(ct *models.Collection) error {
	var existing models.Collection

	logger.Log.WithField("collection", ct.Name).Info("Attempting to save collection")

	// Check if a record with the same name already exists, including soft-deleted records
	err := database.DB.Unscoped().Where("name = ?", ct.Name).First(&existing).Error
	if err == nil {
		// Collection with the same name exists
		if !existing.DeletedAt.Valid {
			// The collection is not soft-deleted, so it's a conflict
			logger.Log.WithField("collection", ct.Name).Warn("Collection with the same name already exists")
			return fmt.Errorf("a collection with the name '%s' already exists", ct.Name)
		}

		// The collection is soft-deleted; restore it
		logger.Log.WithField("collection", ct.Name).Info("Found soft-deleted collection, restoring")

		// Restore the collection
		existing.DeletedAt.Valid = false // Clear the deleted_at flag
		if err := database.DB.Unscoped().Save(&existing).Error; err != nil {
			logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to restore collection")
			return fmt.Errorf("failed to restore collection: %w", err)
		}

		// Restore associated fields
		if err := restoreAssociatedRecords(&models.Attribute{}, existing.ID); err != nil {
			logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to restore associated attributes")
			return fmt.Errorf("failed to restore associated attributes: %w", err)
		}

		// Restore associated relationships
		// TO CHECK

		logger.Log.WithField("collection", ct.Name).Info("Collection restored successfully")
		return nil
	} else if err != gorm.ErrRecordNotFound {
		// An error occurred while checking for existing collections
		logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to check for existing collection")
		return fmt.Errorf("failed to check for existing collection: %w", err)
	}

	// No conflict, create a new collection
	logger.Log.WithField("collection", ct.Name).Info("Creating new collection")
	if err := database.DB.Create(ct).Error; err != nil {
		logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to create collection")
		return fmt.Errorf("failed to save collection: %w", err)
	}

	logger.Log.WithField("collection", ct.Name).Info("Collection created successfully")
	return nil
}

// restoreAssociatedRecords restores soft-deleted associated records (attributes, relationships, etc.).
func restoreAssociatedRecords(model interface{}, collectionID uint) error {
	return database.DB.Unscoped().
		Model(model).
		Where("collection_id = ?", collectionID).
		Update("deleted_at", nil).Error
}

// GetCollectionByID retrieves a collection by its ID.
func GetCollectionByID(id uint) (*models.Collection, error) {
	var ct models.Collection

	// Preload the 'Attributes' relationship
	if err := database.DB.
		Preload("Attributes").
		Where("id = ?", id).
		First(&ct).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("id", id).
				Warn("collection not found")
			return nil, fmt.Errorf("collection with ID '%d' not found", id)
		}

		logger.Log.WithField("id", id).
			Error("Failed to fetch collection", err)
		return nil, fmt.Errorf("failed to fetch collection with ID '%d': %w", id, err)
	}

	logger.Log.WithField("collection", ct.Name).
		Info("Collection fetched successfully")
	return &ct, nil
}

// GetCollectionSchema retrieves the schema definition of a collection by name.
func GetCollectionSchema(id int) (map[string]interface{}, error) {
	var collection models.Collection

	if err := database.DB.Preload("Attributes").Where("id = ?", id).First(&collection).Error; err != nil {
		return nil, fmt.Errorf("collection with id '%d' not found", id)
	}

	// Format the collection using the utility function
	return utils.FormatCollectionSchema(&collection), nil
}

// GetCollectionByName retrieves a collection by its name.
func GetCollectionByName(name string) (*models.Collection, error) {
	var ct models.Collection

	// Preload the 'Attributes' relationship
	if err := database.DB.
		Preload("Attributes").
		Where("name = ?", name).
		First(&ct).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("name", name).
				Warn("collection not found")
			return nil, fmt.Errorf("collection '%s' not found", name)
		}

		logger.Log.WithField("name", name).
			Error("Failed to fetch collection", err)
		return nil, fmt.Errorf("failed to fetch collection '%s': %w", name, err)
	}

	logger.Log.WithField("collection", ct.Name).
		Info("collection fetch successfully")
	return &ct, nil
}

// GetAllCollectionsWithFilters retrieves collections with optional filtering, sorting, and pagination.
func GetAllCollections(filters map[string]interface{}, sortValues []string, rangeValues []int) ([]models.Collection, int, error) {
	var collections []models.Collection
	query := database.DB.Model(&models.Collection{}).Preload("Attributes")

	// Apply filtering dynamically
	if len(filters) > 0 {
		for key, value := range filters {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	// Apply sorting (e.g., ["name", "ASC"])
	if len(sortValues) == 2 {
		sortField := sortValues[0]
		sortOrder := strings.ToUpper(sortValues[1])

		if sortOrder != "ASC" && sortOrder != "DESC" {
			sortOrder = "ASC" // Default to ASC if invalid
		}
		query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))
	} else {
		query = query.Order("id ASC") // Default sorting
	}

	// Count total number of collections
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count collections")
		return nil, 0, err
	}

	// Apply pagination if rangeValues are set (e.g., [0,9])
	if len(rangeValues) == 2 {
		offset := rangeValues[0]
		limit := rangeValues[1] - rangeValues[0] + 1
		query = query.Offset(offset).Limit(limit)
	}

	// Execute query
	if err := query.Find(&collections).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Log.Warn("No collections found")
			return nil, int(total), nil
		}
		logger.Log.WithError(err).Error("Failed to fetch collections")
		return nil, 0, err
	}

	logger.Log.WithField("count", len(collections)).Info("Collections retrieved successfully")
	return collections, int(total), nil
}

// UpdateCollectionByID updates an existing Collection in the database by its ID.
func UpdateCollection(id uint, updated *models.Collection) error {
	var existing models.Collection

	logger.Log.WithField("collection input", updated).Info("Attempting to update collection in database")

	// Find the existing collection by ID
	if err := database.DB.Preload("Attributes").
		Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.WithField("collection_id", id).Warn("Collection not found for update")
			return fmt.Errorf("collection with ID '%d' not found", id)
		}
		logger.Log.WithError(err).WithField("collection_id", id).Error("Failed to retrieve collection for update")
		return fmt.Errorf("failed to retrieve collection: %w", err)
	}

	// Update the basic attributes of the collection
	existing.Name = updated.Name

	// Start a transaction for updating associated attributes and relationships
	tx := database.DB.Begin()
	logger.Log.WithField("collection_id", id).Info("Begin transaction to update collection in database")
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Update fields
	if err := updateAssociatedFields(tx, &existing.ID, updated.Attributes); err != nil {
		logger.Log.WithField("collection_id", id).Error("Rollback failed to update fields")
		tx.Rollback()
		return fmt.Errorf("failed to update fields: %w", err)
	}

	// Save the updated collection
	if err := tx.Save(&existing).Error; err != nil {
		logger.Log.WithError(err).WithField("collection_id", id).Error("Failed to save updated collection")
		tx.Rollback()
		return fmt.Errorf("failed to save updated collection: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).WithField("collection_id", id).Error("Failed to commit transaction for collection update")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Log.WithField("collection_id", id).Info("Collection updated successfully")
	return nil
}

func updateAssociatedFields(tx *gorm.DB, collectionID *uint, updatedAttributes []models.Attribute) error {
	// Fetch existing attributes
	var existingAttributes []models.Attribute
	if err := tx.Where("collection_id = ?", collectionID).Find(&existingAttributes).Error; err != nil {
		return fmt.Errorf("failed to fetch existing attributes: %w", err)
	}

	// Map existing attributes by name for comparison
	existingMap := make(map[string]models.Attribute)
	for _, attr := range existingAttributes {
		existingMap[attr.Name] = attr
	}

	// Process updates and inserts
	for _, updatedAttr := range updatedAttributes {
		if existingAttr, exists := existingMap[updatedAttr.Name]; exists {
			// Update existing attribute
			updatedAttr.ID = existingAttr.ID
			updatedAttr.CollectionID = collectionID
			if err := tx.Save(&updatedAttr).Error; err != nil {
				return fmt.Errorf("failed to update attribute '%s': %w", updatedAttr.Name, err)
			}
			// Remove from the map after processing
			delete(existingMap, updatedAttr.Name)
		} else {
			// Insert new attribute
			updatedAttr.CollectionID = collectionID
			if err := tx.Create(&updatedAttr).Error; err != nil {
				return fmt.Errorf("failed to insert attribute '%s': %w", updatedAttr.Name, err)
			}
		}
	}

	// Delete remaining attributes in the map
	for _, attr := range existingMap {
		if err := tx.Delete(&attr).Error; err != nil {
			return fmt.Errorf("failed to delete attribute '%s': %w", attr.Name, err)
		}
	}

	return nil
}

// DeleteCollection deletes a collection and all associated data by its ID.
func DeleteCollection(collectionID uint) error {
	// Begin a transaction for cascading deletion
	tx := database.DB.Begin()
	logger.Log.WithField("collection_id", collectionID).Info("DeleteCollection: start to delete collection")
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Retrieve the collection
	var Collection models.Collection
	if err := tx.Where("id = ?", collectionID).First(&Collection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Log.WithField("collection_id", collectionID).Error("DeleteCollection: failed to retreive collection")
			tx.Rollback()
			return fmt.Errorf("DeleteCollection: failed to retreive collection '%d' not found", collectionID)
		}
		tx.Rollback()
		return fmt.Errorf("failed to retrieve collection: %w", err)
	}
	logger.Log.WithField("collection", Collection).Info("DeleteCollection: successfully retrieve collection")

	// Delete associated fields
	if err := tx.Where("collection_id = ?", collectionID).Delete(&models.Attribute{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete fields for collection ID '%d': %w", collectionID, err)
	}
	logger.Log.WithField("collection_id", collectionID).Info("DeleteCollection: delete successfully associated fields")

	referencingAttrs, err := IsReferencedByOtherCollections(Collection.Name, tx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to check referencing relations: %w", err)
	}
	if len(referencingAttrs) > 0 {
		tx.Rollback()
		// List the referencing collections and fields for UX
		refs := []string{}
		for _, attr := range referencingAttrs {
			refs = append(refs, fmt.Sprintf("'%s.%s'", attr.Target, attr.Name))
		}
		return fmt.Errorf("Cannot delete collection '%s': it is referenced by fields: %s", Collection.Name, strings.Join(refs, ", "))
	}

	// Delete associated content items
	if err := tx.Where("collection_id = ?", Collection.ID).Delete(&models.Item{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete content items for collection '%s': %w", Collection.Name, err)
	}
	logger.Log.WithField("collection_id", collectionID).Info("DeleteCollection: delete successfully associated items")

	// Delete the collection itself
	if err := tx.Delete(&Collection).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete collection with ID '%d': %w", collectionID, err)
	}

	// Commit the transaction
	return tx.Commit().Error
}

func IsReferencedByOtherCollections(collectionName string, tx *gorm.DB) ([]models.Attribute, error) {
	var referencingAttrs []models.Attribute
	err := tx.Where("type = ? AND target = ?", "relation", collectionName).Find(&referencingAttrs).Error
	return referencingAttrs, err
}
