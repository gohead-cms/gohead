package graphql

import (
	"fmt"

	"gohead/internal/models"
	"gohead/pkg/database"

	"github.com/graphql-go/graphql"
)

// Cache of dynamically generated GraphQL types
var typeRegistry = map[string]*graphql.Object{}

// ConvertCollectionToGraphQLType dynamically creates a GraphQL type for a collection.
func ConvertCollectionToGraphQLType(collection models.Collection) (*graphql.Object, error) {
	// If already created, return cached type
	if gqlType, exists := typeRegistry[collection.Name]; exists {
		return gqlType, nil
	}

	// Define GraphQL Fields based on collection attributes
	fields := graphql.Fields{
		"id": &graphql.Field{Type: graphql.ID}, // Default ID field
	}

	for _, attr := range collection.Attributes {
		// Map attribute types to GraphQL types
		var gqlFieldType graphql.Output
		switch attr.Type {
		case "string":
			gqlFieldType = graphql.String
		case "int":
			gqlFieldType = graphql.Int
		case "bool":
			gqlFieldType = graphql.Boolean
		case "float":
			gqlFieldType = graphql.Float
		case "date":
			gqlFieldType = graphql.String // Can be formatted later
		case "relation":
			// Recursive call to resolve relationship
			relatedType, err := GetOrCreateGraphQLType(attr.Target)
			if err != nil {
				return nil, err
			}
			if attr.Relation == "oneToMany" || attr.Relation == "manyToMany" {
				gqlFieldType = graphql.NewList(relatedType)
			} else {
				gqlFieldType = relatedType
			}
		default:
			return nil, fmt.Errorf("unsupported attribute type: %s", attr.Type)
		}

		// Add field to schema
		fields[attr.Name] = &graphql.Field{
			Type: gqlFieldType,
		}
	}

	// Create new GraphQL object type
	gqlType := graphql.NewObject(graphql.ObjectConfig{
		Name:   collection.Name,
		Fields: fields,
	})

	// Store in registry
	typeRegistry[collection.Name] = gqlType

	return gqlType, nil
}

// GetOrCreateGraphQLType retrieves or creates a GraphQL type from the database.
func GetOrCreateGraphQLType(collectionName string) (*graphql.Object, error) {
	// Check if type is already created
	if gqlType, exists := typeRegistry[collectionName]; exists {
		return gqlType, nil
	}

	// Fetch collection schema from DB
	var collection models.Collection
	if err := database.DB.Where("name = ?", collectionName).Preload("Attributes").First(&collection).Error; err != nil {
		return nil, fmt.Errorf("collection '%s' not found", collectionName)
	}

	// Convert collection to GraphQL type
	return ConvertCollectionToGraphQLType(collection)
}
