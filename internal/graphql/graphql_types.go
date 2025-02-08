package graphql

import (
	"fmt"
	"gohead/internal/models"
	"gohead/pkg/logger"
	"gohead/pkg/storage"

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
		// Capture attr in a local variable for closure safety
		localAttr := attr

		var gqlFieldType graphql.Output
		switch localAttr.Type {
		case "text":
			gqlFieldType = graphql.String
		case "int":
			gqlFieldType = graphql.Int
		case "bool":
			gqlFieldType = graphql.Boolean
		case "float":
			gqlFieldType = graphql.Float
		case "date":
			gqlFieldType = graphql.String // or a custom date type
		case "relation":
			// Resolve related types as needed (omitted for brevity)
			relatedType, err := GetOrCreateGraphQLType(localAttr.Target)
			if err != nil {
				logger.Log.WithFields(map[string]interface{}{
					"attribute_name":  localAttr.Name,
					"relation_target": localAttr.Target,
					"error":           err,
				}).Error("Failed to resolve relation target")
				return nil, err
			}
			if localAttr.Relation == "oneToMany" || localAttr.Relation == "manyToMany" {
				gqlFieldType = graphql.NewList(relatedType)
			} else {
				gqlFieldType = relatedType
			}
		default:
			logger.Log.WithFields(map[string]interface{}{
				"attribute_name":   localAttr.Name,
				"unsupported_type": localAttr.Type,
			}).Warn("Unsupported attribute type")
			return nil, fmt.Errorf("unsupported attribute type: %s", localAttr.Type)
		}

		fields[localAttr.Name] = &graphql.Field{
			Type: gqlFieldType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				// p.Source is expected to be of type models.Item
				if item, ok := p.Source.(models.Item); ok {
					// Return the attribute value from the Data map
					if value, exists := item.Data[localAttr.Name]; exists {
						return value, nil
					}
				}
				return nil, nil
			},
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

	// Fetch collection schema using the storage package instead of querying the DB directly
	collection, err := storage.GetCollectionByName(collectionName)
	if err != nil {
		logger.Log.WithFields(map[string]interface{}{
			"collection_name": collectionName,
			"error":           err,
		}).Warn("Collection not found in storage")
		return nil, fmt.Errorf("collection '%s' not found", collectionName)
	}

	// Convert collection to GraphQL type
	logger.Log.WithField("collection_name", collectionName).Info("Collection found, converting to GraphQL type")
	return ConvertCollectionToGraphQLType(*collection)
}
