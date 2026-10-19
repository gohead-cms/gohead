package graphql

import (
	"errors"
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
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

	// Drafts stay hidden when the parent was read as published content, and
	// always for roles that cannot edit.
	publishedOnly, set := sourceMap[publishedOnlyKey].(bool)
	if !set {
		publishedOnly = true
	}
	if !auth.HasPermission(auth.RoleFromContext(p.Context), auth.ActionUpdate) {
		publishedOnly = true
	}
	fetch := func(id uint) map[string]any {
		item, err := storage.GetVisibleItem(*targetCollection, id, publishedOnly)
		if err != nil {
			if !errors.Is(err, storage.ErrItemNotFound) {
				logger.Log.WithError(err).WithField("id", id).Warn("Failed to fetch related item, skipping.")
			}
			return nil
		}
		loc, _ := sourceMap[localeKey].(gqlLocale)
		return mapItemToGraphQLResult(*item, *targetCollection, publishedOnly, loc)
	}

	// Only manyToMany stores a list of IDs (see models.ValidateItemValues);
	// the other kinds store a single ID.
	if attr.Relation == "manyToMany" {
		results := []map[string]any{}
		for _, id := range models.ToUintSlice(relationValue) {
			if related := fetch(id); related != nil {
				results = append(results, related)
			}
		}
		return results, nil
	}

	if attr.Relation == "oneToOne" || attr.Relation == "oneToMany" || attr.Relation == "manyToOne" {
		id := models.ToUint(relationValue)
		if id == 0 {
			return nil, fmt.Errorf("invalid ID format for relation '%s'", attr.Name)
		}
		if related := fetch(id); related != nil {
			return related, nil
		}
		return nil, nil
	}

	return nil, fmt.Errorf("unsupported relation type '%s'", attr.Type)
}
