package graphql

import (
	"fmt"
	"gohead/internal/models"
	"gohead/pkg/logger"
	"gohead/pkg/storage"

	"github.com/graphql-go/graphql"
)

// ResolveRelation handles fetching related items in a GraphQL query.
func ResolveRelation(p graphql.ResolveParams, collectionID uint, attr models.Attribute) (interface{}, error) {
	// Get the item source
	item, ok := p.Source.(models.Item)
	if !ok {
		return nil, fmt.Errorf("invalid source type for relation '%s'", attr.Name)
	}

	// Extract relation ID(s) from item data
	relationValue, exists := item.Data[attr.Name]
	if !exists {
		logger.Log.WithField("relation", attr.Name).Debug("No relation found in item data")
		return nil, nil // No relation set
	}

	logger.Log.WithFields(map[string]interface{}{
		"collection_id": collectionID,
		"relation":      attr.Name,
		"value":         relationValue,
	}).Debug("Resolving relation")

	// **Handle Many-to-Many and One-to-Many Relations**
	if attr.Relation == "oneToMany" || attr.Relation == "manyToMany" {
		relatedIDs, ok := relationValue.([]interface{})
		if !ok {
			logger.Log.WithField("relation", attr.Name).Warn("Invalid format for many-to-many relation")
			return nil, fmt.Errorf("invalid relation format for '%s'", attr.Name)
		}

		// Convert IDs to uint slice
		var relatedItems []models.Item
		for _, rawID := range relatedIDs {
			if idFloat, ok := rawID.(float64); ok {
				id := uint(idFloat)

				// Fetch target collection
				targetCollection, err := storage.GetCollectionByName(attr.Target)
				if err != nil {
					logger.Log.WithError(err).WithField("target", attr.Target).Warn("Failed to fetch target collection")
					return nil, fmt.Errorf("failed to fetch target collection '%s': %w", attr.Target, err)
				}

				// Fetch related item by ID
				relatedItem, err := storage.GetItemByID(targetCollection.ID, id)
				if err != nil {
					logger.Log.WithError(err).WithField("raw_id", rawID).Warn("Failed to fetch related item")
					continue // Skip this ID but continue processing others
				}
				relatedItems = append(relatedItems, *relatedItem)
			} else {
				logger.Log.WithField("raw_id", rawID).Warn("Invalid relation ID type")
			}
		}

		return relatedItems, nil
	}

	// **Handle One-to-One and Many-to-One Relations**
	if attr.Relation == "oneToOne" || attr.Relation == "manyToOne" {
		relatedID, ok := relationValue.(float64)
		if !ok {
			logger.Log.WithField("relation", attr.Name).Warn("Invalid format for one-to-one relation")
			return nil, fmt.Errorf("invalid relation format for '%s'", attr.Name)
		}

		// Fetch target collection
		targetCollection, err := storage.GetCollectionByName(attr.Target)
		if err != nil {
			logger.Log.WithError(err).WithField("target", attr.Target).Warn("Failed to fetch target collection")
			return nil, fmt.Errorf("failed to fetch target collection '%s': %w", attr.Target, err)
		}

		// Fetch related item by ID
		relatedItem, err := storage.GetItemByID(targetCollection.ID, uint(relatedID))
		if err != nil {
			logger.Log.WithError(err).WithField("relation", attr.Name).Error("Failed to resolve related item")
			return nil, err
		}

		return relatedItem, nil
	}

	logger.Log.WithField("relation", attr.Name).Error("Unsupported relation type")
	return nil, fmt.Errorf("unsupported relation type for '%s'", attr.Name)
}
