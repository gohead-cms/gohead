package graphql

import (
	"fmt"
	"gohead/internal/models"
	"gohead/pkg/database"
	"gohead/pkg/logger"
	"gohead/pkg/storage"
	"strconv"

	"github.com/graphql-go/graphql"
)

var Schema graphql.Schema

// InitializeGraphQLSchema dynamically generates the GraphQL schema.
func InitializeGraphQLSchema() error {
	rootQuery, err := GenerateGraphQLQueries()
	if err != nil {
		return err
	}

	mutation, err := GenerateGraphQLMutations()
	if err != nil {
		return fmt.Errorf("failed to generate GraphQL mutations: %w", err)
	}

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: mutation,
	})
	if err != nil {
		return err
	}

	Schema = schema
	return nil
}

// GenerateGraphQLQueries dynamically creates GraphQL queries for each collection.
func GenerateGraphQLQueries() (*graphql.Object, error) {
	logger.Log.Debug("Starting GenerateGraphQLQueries...")

	fields := graphql.Fields{}

	// Fetch all collections from the database
	var collections []models.Collection
	if err := database.DB.Preload("Attributes").Find(&collections).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch collections from database")
		return nil, err
	}

	logger.Log.WithField("collections_count", len(collections)).Info("Collections retrieved successfully")

	// Create a resolver for each collection
	for _, collection := range collections {
		logger.Log.WithField("collection_name", collection.Name).Debug("Processing collection...")

		gqlType, err := ConvertCollectionToGraphQLType(collection)
		if err != nil {
			logger.Log.WithFields(map[string]interface{}{
				"collection": collection.Name,
				"error":      err,
			}).Error("Failed to convert collection to GraphQL type")
			return nil, err
		}

		logger.Log.WithField("collection", collection.Name).Info("GraphQL type generated successfully")

		fields[collection.Name] = &graphql.Field{
			Type: gqlType,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.ID},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				logger.Log.Debug("Resolve function triggered for collection query")

				id, ok := p.Args["id"].(string)
				if !ok {
					logger.Log.Warn("Missing 'id' argument in query")
					return nil, fmt.Errorf("missing 'id' argument")
				}

				// Convert the string ID to uint
				parsedID, err := strconv.ParseUint(id, 10, 32)
				if err != nil {
					logger.Log.WithError(err).Warn("Failed to parse 'id' argument to uint")
					return nil, fmt.Errorf("invalid 'id' argument")
				}

				logger.Log.WithField("query_id", id).Debug("Fetching item by ID for collection", collection.Name)

				item, err := storage.GetItemByID(collection.ID, uint(parsedID))
				if err != nil {
					logger.Log.WithFields(map[string]interface{}{
						"query_id": id,
						"error":    err,
					}).Warn("Item not found in storage for collection", collection.Name)
					return nil, fmt.Errorf("item not found")
				}

				logger.Log.WithField("query_id", id).Info("Item retrieved successfully for collection", collection.Name)
				return item, nil
			},
		}
	}

	logger.Log.Info("GraphQL queries generated successfully")

	// Create the root query object
	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: fields,
	}), nil
}
