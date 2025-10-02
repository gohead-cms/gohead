package graphql

import (
	"fmt"
	"sync"

	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/graphql-go/graphql"
)

// Schema holds the generated GraphQL schema.
// It is protected by a RWMutex to allow for safe concurrent access and hot-reloading.
var (
	Schema      graphql.Schema
	schemaMutex sync.RWMutex
)

// InitializeGraphQLSchema dynamically generates the GraphQL schema and protects it with a write lock.
// This function can be called again at runtime to "hot-reload" the schema if collections change.
func InitializeGraphQLSchema() error {
	rootQuery, err := GenerateGraphQLQueries()
	if err != nil {
		return err
	}

	mutation, err := GenerateGraphQLMutations()
	if err != nil {
		return fmt.Errorf("failed to generate GraphQL mutations: %w", err)
	}

	schemaConfig := graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: mutation,
	}

	newSchema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return err
	}

	// Use a write lock to update the global schema safely
	schemaMutex.Lock()
	defer schemaMutex.Unlock()
	Schema = newSchema

	logger.Log.Info("GraphQL schema initialized/updated successfully")
	return nil
}

// GetSchema safely returns the current GraphQL schema using a read lock.
// All GraphQL handlers should use this function to get the schema.
func GetSchema() graphql.Schema {
	schemaMutex.RLock()
	defer schemaMutex.RUnlock()
	return Schema
}

// GenerateGraphQLQueries dynamically creates GraphQL queries for each collection.
func GenerateGraphQLQueries() (*graphql.Object, error) {
	logger.Log.Debug("Generating GraphQL queries...")

	fields := graphql.Fields{}

	var collections []models.Collection
	if err := database.DB.Preload("Attributes").Find(&collections).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch collections from database")
		return nil, err
	}

	logger.Log.WithField("collections_count", len(collections)).Info("Collections retrieved for schema generation")

	for _, collection := range collections {
		coll := collection

		logger.Log.WithField("collection_name", coll.Name).Debug("Processing collection for GraphQL schema")

		gqlType, err := ConvertCollectionToGraphQLType(coll)
		if err != nil {
			logger.Log.WithFields(map[string]any{
				"collection": coll.Name,
				"error":      err,
			}).Error("Failed to convert collection to GraphQL type")
			return nil, err
		}

		fields[coll.Name] = &graphql.Field{
			// Use graphql.NewList to signify that this query can return multiple items.
			Type: graphql.NewList(gqlType),
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type:        graphql.ID,
					Description: "Fetch a single item by its ID.",
				},
				"limit": &graphql.ArgumentConfig{
					Type:         graphql.Int,
					DefaultValue: 25,
					Description:  "The number of items to return.",
				},
				"offset": &graphql.ArgumentConfig{
					Type:         graphql.Int,
					DefaultValue: 0,
					Description:  "The number of items to skip for pagination.",
				},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				logger.Log.WithField("collection", coll.Name).Debug("Resolver triggered")

				// Case 1: Fetch a single item by ID
				if idArg, ok := p.Args["id"].(string); ok && idArg != "" {
					parsedID, err := strconv.ParseUint(idArg, 10, 32)
					if err != nil {
						logger.Log.WithError(err).Warn("Failed to parse 'id' argument to uint")
						return nil, fmt.Errorf("invalid 'id' argument")
					}

					item, err := storage.GetItemByID(coll.ID, uint(parsedID))
					if err != nil {
						// For "not found", GraphQL typically returns null data and no error.
						return nil, nil
					}

					return []any{mapItemToGraphQLResult(*item, coll.Attributes)}, nil
				}

				// Case 2: Fetch a list of items with pagination
				limit := p.Args["limit"].(int)
				offset := p.Args["offset"].(int)

				items, _, err := storage.GetItems(coll.ID, offset, limit)
				if err != nil {
					logger.Log.WithError(err).Warn("Failed to fetch items for collection", coll.Name)
					return nil, fmt.Errorf("failed to fetch items")
				}

				results := make([]map[string]any, 0, len(items))
				for _, item := range items {
					results = append(results, mapItemToGraphQLResult(item, coll.Attributes))
				}

				return results, nil
			},
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: fields,
	}), nil
}

// mapItemToGraphQLResult is a helper function to convert a storage item
// into a map suitable for a GraphQL response, reducing code duplication.
func mapItemToGraphQLResult(item models.Item, attributes []models.Attribute) map[string]any {
	result := map[string]any{"id": item.ID}
	for _, attr := range attributes {
		if value, exists := item.Data[attr.Name]; exists {
			result[attr.Name] = value
		}
	}
	return result
}
