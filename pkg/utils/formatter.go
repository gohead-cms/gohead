package utils

import (
	"gohead/internal/models"
	"strings"
)

// FormatCollectionSchema formats a Collection into a Strapi-like response
func FormatCollectionSchema(collection *models.Collection) map[string]interface{} {
	formatted := map[string]interface{}{
		"id":  collection.ID,
		"uid": "api::" + collection.Name + "." + collection.Name,
		"schema": map[string]interface{}{
			"collectionName": collection.Name,
			"info": map[string]interface{}{
				"singularName": collection.Name,
				"pluralName":   collection.Name,
				"displayName":  capitalize(collection.Name),
			},
			"attributes": map[string]interface{}{},
		},
	}

	schema := formatted["schema"].(map[string]interface{})
	attributes := schema["attributes"].(map[string]interface{})

	for _, attr := range collection.Attributes {
		attrMap := map[string]interface{}{
			"type":     attr.Type,
			"required": attr.Required,
		}
		if attr.Type == "relation" {
			attrMap["target"] = "api::" + attr.Target + "." + attr.Target
			attrMap["relation"] = attr.Relation
		}
		attributes[attr.Name] = attrMap
	}

	return formatted
}

// FormatCollectionItem formats a single item to match Strapi's response format.
func FormatCollectionItem(item *models.Item, collection *models.Collection) map[string]any {
	if item == nil || collection == nil {
		return nil
	}

	formatted := map[string]interface{}{
		"id":         item.ID,
		"attributes": map[string]interface{}{},
	}

	attributes := formatted["attributes"].(map[string]any)

	for _, attr := range collection.Attributes {
		value, exists := item.Data[attr.Name]
		if attr.Type == "relation" {
			switch attr.Relation {
			case "oneToOne":
				var relData any
				if exists && value != nil {
					relData = map[string]any{"id": toInt(value)}
				} else {
					relData = nil
				}
				attributes[attr.Name] = map[string]any{"data": relData}
			case "manyToMany":
				relationData := []map[string]interface{}{}
				if exists && value != nil {
					if ids, ok := value.([]interface{}); ok {
						for _, id := range ids {
							relationData = append(relationData, map[string]interface{}{"id": toInt(id)})
						}
					}
				}
				attributes[attr.Name] = map[string]interface{}{"data": relationData}
			}
		} else if exists {
			attributes[attr.Name] = value
		}
	}

	return formatted
}

// FormatCollectionItems formats multiple items and includes pagination metadata.
func FormatCollectionItems(items []models.Item, collection *models.Collection) []map[string]interface{} {
	formattedItems := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		formattedItems = append(formattedItems, FormatCollectionItem(&items[i], collection))
	}
	return formattedItems
}

// Helper: capitalize first letter
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Helper: safely convert numeric interface{} to int for IDs
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
		return 0 // or panic/error if desired
	}
}
