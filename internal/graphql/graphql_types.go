package graphql

import (
	"fmt"
	"gohead/internal/models"
	"gohead/pkg/database"
	"gohead/pkg/logger"

	"github.com/graphql-go/graphql"
)

// Cache of dynamically generated GraphQL types
var typeRegistry = map[string]*graphql.Object{}

// ConvertCollectionToGraphQLType dynamically creates a GraphQL type for a collection.
func ConvertCollectionToGraphQLType(collection models.Collection) (*graphql.Object, error) {
	logger.Log.WithField("collection_name", collection.Name).Debug("Starting ConvertCollectionToGraphQLType")

	// If already created, return cached type
	if gqlType, exists := typeRegistry[collection.Name]; exists {
		logger.Log.WithField("collection_name", collection.Name).Debug("Returning cached GraphQL type")
		return gqlType, nil
	}

	// Define GraphQL Fields based on collection attributes
	fields := graphql.Fields{
		"id": &graphql.Field{Type: graphql.ID}, // Default ID field
	}

	for _, attr := range collection.Attributes {
		logger.Log.WithFields(map[string]interface{}{
			"attribute_name": attr.Name,
			"attribute_type": attr.Type,
		}).Debug("Processing attribute")

		// Map attribute types to GraphQL types
		var gqlFieldType graphql.Output
		switch attr.Type {
		case "text":
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
				logger.Log.WithFields(map[string]interface{}{
					"attribute_name":  attr.Name,
					"relation_target": attr.Target,
					"error":           err,
				}).Error("Failed to resolve relation target")
				return nil, err
			}
			if attr.Relation == "oneToMany" || attr.Relation == "manyToMany" {
				gqlFieldType = graphql.NewList(relatedType)
			} else {
				gqlFieldType = relatedType
			}
		default:
			logger.Log.WithFields(map[string]interface{}{
				"attribute_name":   attr.Name,
				"unsupported_type": attr.Type,
			}).Warn("Unsupported attribute type")
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
	logger.Log.WithField("collection_name", collection.Name).Info("GraphQL type created successfully")

	return gqlType, nil
}

// GetOrCreateGraphQLType retrieves or creates a GraphQL type from the database.
func GetOrCreateGraphQLType(collectionName string) (*graphql.Object, error) {
	logger.Log.WithField("collection_name", collectionName).Debug("Starting GetOrCreateGraphQLType")

	// Check if type is already created
	if gqlType, exists := typeRegistry[collectionName]; exists {
		logger.Log.WithField("collection_name", collectionName).Debug("Returning cached GraphQL type")
		return gqlType, nil
	}

	// Fetch collection schema from DB
	var collection models.Collection
	if err := database.DB.Where("name = ?", collectionName).Preload("Attributes").First(&collection).Error; err != nil {
		logger.Log.WithFields(map[string]interface{}{
			"collection_name": collectionName,
			"error":           err,
		}).Warn("Collection not found in database")
		return nil, fmt.Errorf("collection '%s' not found", collectionName)
	}

	// Convert collection to GraphQL type
	logger.Log.WithField("collection_name", collectionName).Info("Collection found, converting to GraphQL type")
	return ConvertCollectionToGraphQLType(collection)
}
