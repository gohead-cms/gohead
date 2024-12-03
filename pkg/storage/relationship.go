// pkg/storage/content_relation.go
package storage

import (
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

// SaveRelationship handles saving relationships for an item.
func SaveRelationship(ct *models.Collection, itemID uint, itemData map[string]interface{}) error {
	for _, rel := range ct.Relationships {
		value, exists := itemData[rel.FieldName]
		if !exists {
			continue // Skip if the related field is not present in the data
		}

		switch rel.RelationType {
		case "one-to-one", "one-to-many":
			if nestedData, ok := value.(map[string]interface{}); ok {
				// Save nested item
				nestedItem := models.Item{
					CollectionID: rel.RelatedCollection,
					Data:         nestedData,
				}
				if err := SaveItem(&nestedItem); err != nil {
					logger.Log.WithField("field", rel.FieldName).WithError(err).Error("Failed to save nested content item")
					return err
				}
				// Save the relationship
				relation := models.Relationship{
					CollectionID:      ct.ID,
					ItemID:            itemID,
					RelatedCollection: rel.RelatedCollection,
					RelatedItemID:     nestedItem.ID,
					RelationType:      rel.RelationType,
					FieldName:         rel.FieldName,
				}
				if err := database.DB.Create(&relation).Error; err != nil {
					logger.Log.WithField("field", rel.FieldName).WithError(err).Error("Failed to save content relation")
					return err
				}
			} else if relatedID, ok := value.(float64); ok {
				// Save direct ID relationship
				relation := models.Relationship{
					CollectionID:      ct.ID,
					ItemID:            itemID,
					RelatedCollection: rel.RelatedCollection,
					RelatedItemID:     uint(relatedID),
					RelationType:      rel.RelationType,
					FieldName:         rel.FieldName,
				}
				if err := database.DB.Create(&relation).Error; err != nil {
					logger.Log.WithField("field", rel.FieldName).WithError(err).Error("Failed to save relation")
					return err
				}
			} else {
				logger.Log.WithField("field", rel.FieldName).Warn("Invalid type for field; expected map or float64")
				return nil
			}
		case "many-to-many":
			if nestedArray, ok := value.([]interface{}); ok {
				for _, element := range nestedArray {
					if nestedData, isMap := element.(map[string]interface{}); isMap {
						// Save nested item
						nestedItem := models.Item{
							CollectionID: rel.RelatedCollection,
							Data:         nestedData,
						}
						if err := SaveItem(&nestedItem); err != nil {
							logger.Log.WithField("field", rel.FieldName).WithError(err).Error("Failed to save nested content item")
							return err
						}
						element = float64(nestedItem.ID)
					}
					if relatedID, ok := element.(float64); ok {
						// Save many-to-many relationship
						relation := models.Relationship{
							CollectionID:      ct.ID,
							ItemID:            itemID,
							RelatedCollection: rel.RelatedCollection,
							RelatedItemID:     uint(relatedID),
							RelationType:      rel.RelationType,
							FieldName:         rel.FieldName,
						}
						if err := database.DB.Create(&relation).Error; err != nil {
							logger.Log.WithField("field", rel.FieldName).WithError(err).Error("Failed to save relation")
							return err
						}
					} else {
						logger.Log.WithField("field", rel.FieldName).Warn("Invalid type for array element; expected map or float64")
						return nil
					}
				}
			} else {
				logger.Log.WithField("field", rel.FieldName).Warn("Invalid type for field; expected array")
				return nil
			}
		default:
			logger.Log.WithField("relation_type", rel.RelationType).Error("Unsupported relationship type")
			return nil
		}
	}
	return nil
}

func GetRelationships(Collection string, itemID uint) ([]models.Relationship, error) {
	var relations []models.Relationship
	err := database.DB.Where("collection = ? AND item_id = ?", Collection, itemID).Find(&relations).Error
	return relations, err
}
