package graphql

import (
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/graphql-go/graphql"
)

// ResolveRelation handles fetching related items in a GraphQL query.
func ResolveRelation(p graphql.ResolveParams, collectionID uint, attr models.Attribute) (any, error) {
	sourceMap, ok := p.Source.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid source type for relation '%s', expected map[string]any", attr.Name)
	}

	relationValue, exists := sourceMap[attr.Name]
	if !exists || relationValue == nil {
		return nil, nil // No relation set, return null.
	}

	targetCollection, err := storage.GetCollectionByName(attr.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch target collection '%s': %w", attr.Target, err)
	}

	// Handle Many-to-Many and One-to-Many Relations
	// Only manyToMany stores a list of IDs (see models.ValidateItemValues);
	// the other kinds store a single ID.
	if attr.Relation == "manyToMany" {
		relatedIDs, ok := relationValue.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid relation format for '%s', expected an array", attr.Name)
		}

		// Field resolvers expect plain map[string]any sources.
		var results []map[string]any
		for _, rawID := range relatedIDs {

			id := models.ToUint(rawID)
			if id == 0 {
				continue
			}

			relatedItem, err := storage.GetItemByID(targetCollection.ID, id)
			if err != nil {
				logger.Log.WithError(err).WithField("id", id).Warn("Failed to fetch related item, skipping.")
				continue
			}

			relatedItem.Data["id"] = relatedItem.ID
			results = append(results, map[string]any(relatedItem.Data))
		}
		return results, nil
	}

	// Handle One-to-One and Many-to-One Relations
	if attr.Relation == "oneToOne" || attr.Relation == "oneToMany" || attr.Relation == "manyToOne" {
		id := models.ToUint(relationValue)
		if id == 0 {
			return nil, fmt.Errorf("invalid ID format for relation '%s'", attr.Name)
		}

		relatedItem, err := storage.GetItemByID(targetCollection.ID, id)
		if err != nil {
			return nil, nil // Return null if the related item isn't found
		}

		relatedItem.Data["id"] = relatedItem.ID
		return map[string]any(relatedItem.Data), nil
	}

	return nil, fmt.Errorf("unsupported relation type '%s'", attr.Type)
}
