package utils

import (
	"gohead/internal/models"
	"strings"

	"github.com/gertd/go-pluralize"
)

var pluralizeClient = pluralize.NewClient()

func FormatCollectionSchema(collection *models.Collection) map[string]any {
	singular := pluralizeClient.Singular(collection.Name)
	plural := pluralizeClient.Plural(collection.Name)

	attrSchema := map[string]any{}
	for _, attr := range collection.Attributes {
		attrDef := map[string]any{
			"type":     attr.Type,
			"required": attr.Required,
		}
		if attr.Type == "relation" {
			attrDef["target"] = "api::" + attr.Target + "." + attr.Target
			attrDef["relation"] = attr.Relation
		}
		attrSchema[attr.Name] = attrDef
	}

	return map[string]any{
		"id":  collection.ID,
		"uid": "api::" + collection.Name + "." + collection.Name,
		"schema": map[string]any{
			"collectionName": collection.Name,
			"info": map[string]any{
				"singularName": singular,
				"pluralName":   plural,
				"displayName":  capitalize(collection.Name),
			},
			"attributes": attrSchema,
		},
	}
}

// FormatCollectionsSchema: returns an array of collection schemas for admin endpoints.
func FormatCollectionsSchema(collections []models.Collection) []map[string]any {
	formatted := make([]map[string]any, 0, len(collections))
	for _, col := range collections {
		formatted = append(formatted, FormatCollectionSchema(&col))
	}
	return formatted
}

// FormatCollectionData returns an array of collections as DATA (not schema), Strapi-style for /collections API.
func FormatCollectionsData(collections []models.Collection) []map[string]any {
	formatted := make([]map[string]any, 0, len(collections))
	for _, col := range collections {
		formatted = append(formatted, map[string]any{
			"id": col.ID,
			"attributes": map[string]any{
				"name":        col.Name,
				"kind":        col.Kind,
				"description": col.Description,
				// You may add more, but do NOT nest attributes definition here!
			},
		})
	}
	return formatted
}

// FormatCollectionItem formats a single item for a Strapi-like /api/[collection] response.
func FormatCollectionItem(item *models.Item, collection *models.Collection) map[string]any {
	if item == nil || collection == nil {
		return nil
	}

	attributes := map[string]any{}
	for _, attr := range collection.Attributes {
		val, exists := item.Data[attr.Name]
		switch attr.Type {
		case "relation":
			switch attr.Relation {
			case "oneToOne":
				var relData any
				if exists && val != nil {
					relData = map[string]any{"id": toInt(val)}
				}
				attributes[attr.Name] = map[string]any{"data": relData}
			case "manyToMany":
				relArr := []map[string]any{}
				if exists && val != nil {
					if ids, ok := val.([]any); ok {
						for _, id := range ids {
							relArr = append(relArr, map[string]any{"id": toInt(id)})
						}
					}
				}
				attributes[attr.Name] = map[string]any{"data": relArr}
			}
		default:
			if exists {
				attributes[attr.Name] = val
			}
		}
	}

	return map[string]any{
		"id":         item.ID,
		"attributes": attributes,
	}
}

// FormatCollectionItems returns multiple items formatted for Strapi
func FormatCollectionItems(items []models.Item, collection *models.Collection) []map[string]any {
	formatted := make([]map[string]any, 0, len(items))
	for i := range items {
		formatted = append(formatted, FormatCollectionItem(&items[i], collection))
	}
	return formatted
}

// FormatNestedItemToStrapi wraps your nested JSONMap
func FormatNestedItems(itemID uint, data models.JSONMap, collection *models.Collection) map[string]any {
	attributes := map[string]any{}

	for _, attr := range collection.Attributes {
		val, exists := data[attr.Name]
		switch attr.Type {
		case "relation":
			switch attr.Relation {
			case "oneToOne":
				if !exists || val == nil {
					attributes[attr.Name] = nil
				} else if nested, ok := val.(map[string]any); ok {
					// Remove "id" if present from attributes
					attrCopy := map[string]any{}
					for k, v := range nested {
						if k != "id" {
							attrCopy[k] = v
						}
					}
					attributes[attr.Name] = attrCopy
				} else {
					attributes[attr.Name] = val // fallback: put value directly
				}
			case "manyToMany":
				arr := []any{}
				if exists && val != nil {
					switch list := val.(type) {
					case []any:
						for _, elem := range list {
							arr = append(arr, elem)
						}
					case []models.JSONMap:
						for _, nested := range list {
							arr = append(arr, nested)
						}
					}
				}
				attributes[attr.Name] = arr

			}
		default:
			if exists {
				attributes[attr.Name] = val
			}
		}
	}

	return map[string]any{
		"id":         itemID,
		"attributes": attributes,
	}
}

func toUintIfPresent(val any) (uint, bool) {
	switch v := val.(type) {
	case int:
		return uint(v), true
	case int64:
		return uint(v), true
	case float64:
		return uint(v), true
	case uint:
		return v, true
	case uint64:
		return uint(v), true
	}
	return 0, false
}

// --- Helpers ---

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func toInt(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}
