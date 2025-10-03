package graphql

import (
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/internal/types"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/graphql-go/graphql"
)

var typeRegistry = make(map[string]*graphql.Object)

func ConvertCollectionToGraphQLType(collection models.Collection) (*graphql.Object, error) {
	if gqlType, exists := typeRegistry[collection.Name]; exists {
		return gqlType, nil
	}

	// Create the GraphQL object. All field logic is defined inside the FieldsThunk.
	gqlType := graphql.NewObject(graphql.ObjectConfig{
		Name: collection.Name,
		// This function is the "thunk". It's delayed work.
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			// 1. Create the map of fields inside the thunk.
			fields := graphql.Fields{
				"id": &graphql.Field{Type: graphql.ID},
			}

			// 2. Loop through attributes and build all fields.
			for _, attr := range collection.Attributes {
				localAttr := attr

				var gqlFieldType graphql.Output
				var err error

				if localAttr.Type == "relation" {
					relatedType, err := GetOrCreateGraphQLType(localAttr.Target)
					if err != nil {
						panic(fmt.Sprintf("schema error: failed to resolve relation '%s': %v", localAttr.Name, err))
					}
					gqlFieldType = relatedType // Simplified for clarity
				} else {
					gqlFieldType, err = types.GetGraphQLType(localAttr.Type)
					if err != nil {
						panic(fmt.Sprintf("schema error: bad type for attribute '%s': %v", localAttr.Name, err))
					}
				}

				fields[localAttr.Name] = &graphql.Field{
					Type: gqlFieldType,
					// Add your resolver logic here
				}
			}
			// 3. Return the completed map.
			return fields
		}),
	})

	// Cache the type so it can be looked up by other relations.
	typeRegistry[collection.Name] = gqlType

	logger.Log.WithField("collection_name", collection.Name).Info("GraphQL type created successfully")
	return gqlType, nil
}

// GetOrCreateGraphQLType retrieves a type from the cache or creates it by fetching its schema.
func GetOrCreateGraphQLType(collectionName string) (*graphql.Object, error) {
	if gqlType, exists := typeRegistry[collectionName]; exists {
		return gqlType, nil
	}

	collection, err := storage.GetCollectionByName(collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection '%s' not found for relation", collectionName)
	}

	return ConvertCollectionToGraphQLType(*collection)
}
