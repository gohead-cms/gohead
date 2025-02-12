package utils

import (
	"gohead/internal/models"
	"strings"
)

// FormatCollectionSchema formats a Collection into a Strapi-like response
func FormatCollectionSchema(collection *models.Collection) map[string]interface{} {
	formatted := map[string]interface{}{
		"id":  collection.ID,
		"uid": "api::" + collection.Name + "." + collection.Name, // Generate UID
		"schema": map[string]interface{}{
			"collectionName": collection.Name,
			"info": map[string]interface{}{
				"singularName": collection.Name, // Singular name assumed same as collection name
				"pluralName":   collection.Name, // Assuming same name, can be modified if needed
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
func FormatCollectionItem(item *models.Item, collection *models.Collection) map[string]interface{} {
	if item == nil || collection == nil {
		return nil
	}

	// Base structure
	formatted := map[string]interface{}{
		"id":         item.ID,
		"attributes": map[string]interface{}{},
	}

	// Format attributes
	attributes := formatted["attributes"].(map[string]interface{})

	for _, attr := range collection.Attributes {
		if value, exists := item.Data[attr.Name]; exists {
			if attr.Type == "relation" {
				// If it's a relation, wrap it in a "data" key
				if attr.Relation == "oneToOne" {
					attributes[attr.Name] = map[string]interface{}{
						"data": map[string]interface{}{
							"id": value,
						},
					}
				} else if attr.Relation == "manyToMany" {
					var relationData []map[string]interface{}
					if ids, ok := value.([]interface{}); ok {
						for _, id := range ids {
							relationData = append(relationData, map[string]interface{}{
								"id": id,
							})
						}
					}
					attributes[attr.Name] = map[string]interface{}{
						"data": relationData,
					}
				}
			} else {
				attributes[attr.Name] = value
			}
		}
	}

	return formatted
}

// FormatCollectionItems formats multiple items and includes pagination metadata.
func FormatCollectionItems(items []models.Item, collection *models.Collection) []map[string]interface{} {
	formattedItems := []map[string]interface{}{}

	for _, item := range items {
		formattedItems = append(formattedItems, FormatCollectionItem(&item, collection))
	}

	return formattedItems
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
