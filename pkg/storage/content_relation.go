// pkg/storage/content_relation.go
package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
)

func SaveContentRelations(ct models.ContentType, itemID uint, itemData map[string]interface{}) error {
	for _, rel := range ct.Relationships {
		value := itemData[rel.FieldName]

		switch rel.RelationType {
		case "one-to-one", "one-to-many":
			relatedItemID := uint(value.(float64))
			relation := models.ContentRelation{
				ContentType:   ct.Name,
				ContentItemID: itemID,
				RelatedType:   rel.RelatedType,
				RelatedItemID: relatedItemID,
				RelationType:  rel.RelationType,
				FieldName:     rel.FieldName,
			}
			if err := database.DB.Create(&relation).Error; err != nil {
				return err
			}
		case "many-to-many":
			ids := value.([]interface{})
			for _, id := range ids {
				relatedItemID := uint(id.(float64))
				relation := models.ContentRelation{
					ContentType:   ct.Name,
					ContentItemID: itemID,
					RelatedType:   rel.RelatedType,
					RelatedItemID: relatedItemID,
					RelationType:  rel.RelationType,
					FieldName:     rel.FieldName,
				}
				if err := database.DB.Create(&relation).Error; err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unsupported relationship type: %s", rel.RelationType)
		}
	}
	return nil
}

func GetContentRelations(contentType string, itemID uint) ([]models.ContentRelation, error) {
	var relations []models.ContentRelation
	err := database.DB.Where("content_type = ? AND content_item_id = ?", contentType, itemID).Find(&relations).Error
	return relations, err
}
