package graphql

import (
	"fmt"
	"gohead/internal/models"
	"gohead/pkg/database"

	"github.com/graphql-go/graphql"
)

// GenerateGraphQLQueries dynamically creates GraphQL queries for each collection.
func GenerateGraphQLQueries() (*graphql.Object, error) {
	fields := graphql.Fields{}

	// Fetch all collections from the database
	var collections []models.Collection
	if err := database.DB.Preload("Attributes").Find(&collections).Error; err != nil {
		return nil, err
	}

	// Create a resolver for each collection
	for _, collection := range collections {
		gqlType, err := ConvertCollectionToGraphQLType(collection)
		if err != nil {
			return nil, err
		}

		fields[collection.Name] = &graphql.Field{
			Type: gqlType,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.ID},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				id, ok := p.Args["id"].(string)
				if !ok {
					return nil, fmt.Errorf("missing 'id' argument")
				}

				var item models.Item
				if err := database.DB.Where("id = ?", id).First(&item).Error; err != nil {
					return nil, fmt.Errorf("item not found")
				}

				return item, nil
			},
		}
	}

	// Create the root query object
	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: fields,
	}), nil
}
