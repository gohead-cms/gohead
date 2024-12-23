package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

// SaveRelationship handles saving relationships for an item.
// SaveRelationship handles saving relationships for an item.
func SaveRelationship(ct *models.Collection, sourceItemID uint, itemData models.JSONMap) error {
	// for _, rel := range ct.Relationships {
	// 	value, exists := itemData[rel.Field]
	// 	if !exists {
	// 		logger.Log.WithField("field", rel.Field).Info("Skipping missing field for relationship")
	// 		continue
	// 	}

	// 	switch rel.RelationType {
	// 	case "one-to-one", "one-to-many":
	// 		if err := saveOneToManyRelationship(ct.ID, rel, &sourceItemID, value); err != nil {
	// 			return err
	// 		}
	// 	case "many-to-many":
	// 		if err := saveManyToManyRelationship(ct.ID, rel, &sourceItemID, value); err != nil {
	// 			return err
	// 		}
	// 	default:
	// 		logger.Log.WithField("relation_type", rel.RelationType).Error("Unsupported relationship type")
	// 		return fmt.Errorf("unsupported relationship type: %s", rel.RelationType)
	// 	}
	// }
	return nil
}

// saveOneToManyRelationship handles saving one-to-one or one-to-many relationships.
func saveOneToManyRelationship(collectionID uint, rel models.Relationship, sourceItemID *uint, value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		// Save nested item
		nestedItem := models.Item{
			CollectionID: rel.CollectionID,
			Data:         models.JSONMap(v),
		}
		if err := SaveItem(&nestedItem); err != nil {
			logger.Log.WithField("field", rel.Attribute).WithError(err).Error("Failed to save nested content item")
			return err
		}

		// Save the relationship
		relation := models.Relationship{
			CollectionID: collectionID,
			SourceItemID: sourceItemID,
			RelationType: rel.RelationType,
			Attribute:    rel.Attribute,
		}
		if err := database.DB.Create(&relation).Error; err != nil {
			logger.Log.WithField("field", rel.Attribute).WithError(err).Error("Failed to save content relation")
			return err
		}
	case float64:
		// Save direct ID relationship
		relation := models.Relationship{
			CollectionID: collectionID,
			SourceItemID: sourceItemID,
			RelationType: rel.RelationType,
			Attribute:    rel.Attribute,
		}
		if err := database.DB.Create(&relation).Error; err != nil {
			logger.Log.WithField("field", rel.Attribute).WithError(err).Error("Failed to save relation")
			return err
		}
	default:
		logger.Log.WithField("field", rel.Attribute).Warn("Invalid type for attribute; expected map or float64")
		return fmt.Errorf("invalid type for field '%s': expected map[string]interface{} or float64", rel.Attribute)
	}
	return nil
}

// saveManyToManyRelationship handles saving many-to-many relationships.
func saveManyToManyRelationship(collectionID uint, rel models.Relationship, sourceItemID *uint, value interface{}) error {
	nestedArray, ok := value.([]interface{})
	if !ok {
		logger.Log.WithField("field", rel.Attribute).Warn("Invalid type for field; expected array")
		return fmt.Errorf("invalid type for field '%s': expected array", rel.Attribute)
	}

	for _, element := range nestedArray {
		switch e := element.(type) {
		case map[string]interface{}:
			// Save nested item
			nestedItem := models.Item{
				CollectionID: rel.CollectionID,
				Data:         models.JSONMap(e),
			}
			if err := SaveItem(&nestedItem); err != nil {
				logger.Log.WithField("field", rel.Attribute).WithError(err).Error("Failed to save nested content item")
				return err
			}

			// Save the relationship
			relation := models.Relationship{
				CollectionID: collectionID,
				SourceItemID: sourceItemID,
				RelationType: rel.RelationType,
				Attribute:    rel.Attribute,
			}
			if err := database.DB.Create(&relation).Error; err != nil {
				logger.Log.WithField("field", rel.Attribute).WithError(err).Error("Failed to save relation")
				return err
			}
		case float64:
			// Save direct ID relationship
			relation := models.Relationship{
				CollectionID: collectionID,
				SourceItemID: sourceItemID,
				RelationType: rel.RelationType,
				Attribute:    rel.Attribute,
			}
			if err := database.DB.Create(&relation).Error; err != nil {
				logger.Log.WithField("field", rel.Attribute).WithError(err).Error("Failed to save relation")
				return err
			}
		default:
			logger.Log.WithField("field", rel.Attribute).Warn("Invalid type for array element; expected map or float64")
			return fmt.Errorf("invalid type for array element in field '%s': expected map[string]interface{} or float64", rel.Attribute)
		}
	}
	return nil
}

// GetRelationships retrieves relationships for a specific item within a collection.
func GetRelationships(collectionID uint, sourceItemID uint) ([]models.Relationship, error) {
	var relations []models.Relationship
	err := database.DB.Where("collection_id = ? AND source_item_id = ?", collectionID, sourceItemID).Find(&relations).Error
	if err != nil {
		logger.Log.WithError(err).Error("Failed to fetch relationships")
		return nil, err
	}
	return relations, nil
}
