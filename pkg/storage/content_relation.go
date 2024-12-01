// pkg/storage/content_relation.go
package storage

import (
	"fmt"

	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
)

func SaveContentRelations(ct *models.ContentType, itemID uint, itemData map[string]interface{}) error {
	for _, rel := range ct.Relationships {
		value, exists := itemData[rel.FieldName]
		if !exists {
			continue // Skip if the related field is not present in the data
		}

		switch rel.RelationType {
		case "one-to-one", "one-to-many":
			// Handle nested objects or numeric IDs for one-to-one/one-to-many
			if nestedData, ok := value.(map[string]interface{}); ok {
				// Handle a single nested object
				nestedItem := models.ContentItem{
					ContentType: rel.RelatedType,
					Data:        nestedData,
				}
				if err := SaveContentItem(&nestedItem); err != nil {
					return fmt.Errorf("failed to save nested content item for field '%s': %w", rel.FieldName, err)
				}
				// Save the relation
				relation := models.ContentRelation{
					ContentType:   ct.Name,
					ContentItemID: itemID,
					RelatedType:   rel.RelatedType,
					RelatedItemID: nestedItem.ID,
					RelationType:  rel.RelationType,
					FieldName:     rel.FieldName,
				}
				if err := database.DB.Create(&relation).Error; err != nil {
					return fmt.Errorf("failed to save content relation for field '%s': %w", rel.FieldName, err)
				}
			} else if nestedArray, ok := value.([]interface{}); ok {
				// Handle an array of nested objects
				for _, element := range nestedArray {
					if nestedData, ok := element.(map[string]interface{}); ok {
						nestedItem := models.ContentItem{
							ContentType: rel.RelatedType,
							Data:        nestedData,
						}
						if err := SaveContentItem(&nestedItem); err != nil {
							return fmt.Errorf("failed to save nested content item for field '%s': %w", rel.FieldName, err)
						}
						// Save the relation
						relation := models.ContentRelation{
							ContentType:   ct.Name,
							ContentItemID: itemID,
							RelatedType:   rel.RelatedType,
							RelatedItemID: nestedItem.ID,
							RelationType:  rel.RelationType,
							FieldName:     rel.FieldName,
						}
						if err := database.DB.Create(&relation).Error; err != nil {
							return fmt.Errorf("failed to save content relation for field '%s': %w", rel.FieldName, err)
						}
					} else {
						return fmt.Errorf("invalid type for array element in field '%s': expected map[string]interface{}", rel.FieldName)
					}
				}
			} else {
				return fmt.Errorf("invalid type for field '%s': expected map or array", rel.FieldName)
			}
		case "many-to-many":
			// Handle an array of numeric IDs or nested objects for many-to-many
			if nestedArray, ok := value.([]interface{}); ok {
				for _, element := range nestedArray {
					if nestedData, isMap := element.(map[string]interface{}); isMap {
						// Handle nested objects
						nestedItem := models.ContentItem{
							ContentType: rel.RelatedType,
							Data:        nestedData,
						}
						if err := SaveContentItem(&nestedItem); err != nil {
							return fmt.Errorf("failed to save nested content item for field '%s': %w", rel.FieldName, err)
						}
						element = float64(nestedItem.ID) // Use the newly created item's ID
					}
					if relatedID, ok := element.(float64); ok {
						relation := models.ContentRelation{
							ContentType:   ct.Name,
							ContentItemID: itemID,
							RelatedType:   rel.RelatedType,
							RelatedItemID: uint(relatedID),
							RelationType:  rel.RelationType,
							FieldName:     rel.FieldName,
						}
						if err := database.DB.Create(&relation).Error; err != nil {
							return fmt.Errorf("failed to save content relation for field '%s': %w", rel.FieldName, err)
						}
					} else {
						return fmt.Errorf("invalid type for array element in field '%s': expected map or float64", rel.FieldName)
					}
				}
			} else {
				return fmt.Errorf("invalid type for field '%s': expected array", rel.FieldName)
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
