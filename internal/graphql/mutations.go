package graphql

import (
	"fmt"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/graphql-go/graphql"
)

// GenerateGraphQLMutations creates GraphQL mutations for collections
func GenerateGraphQLMutations() (*graphql.Object, error) {
	fields := graphql.Fields{}

	// Fetch all collections from the database
	var collections []models.Collection
	if err := database.DB.Preload("Attributes").Find(&collections).Error; err != nil {
		return nil, err
	}

	logger.Log.WithField("collections", collections).Debug("Generating GraphQL Mutations")

	// Create a mutation for each collection
	for _, collection := range collections {
		gqlType, err := ConvertCollectionToGraphQLType(collection)
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to generate GraphQL type for collection: %s", collection.Name)
			return nil, err
		}

		// Mutation: Create Item
		fields["create"+collection.Name] = &graphql.Field{
			Type: gqlType,
			Args: generateGraphQLInputArgs(collection),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return createCollectionItem(p, collection)
			},
		}

		// Mutation: Update Item
		fields["update"+collection.Name] = &graphql.Field{
			Type: gqlType,
			Args: generateGraphQLInputArgs(collection, "id"),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return updateCollectionItem(p, collection)
			},
		}

		// Mutation: Delete Item
		fields["delete"+collection.Name] = &graphql.Field{
			Type: graphql.Boolean,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return deleteCollectionItem(p, collection)
			},
		}
	}

	logger.Log.WithField("collections", collections).Debug("Generating GraphQL Mutations done")

	// Return root mutation object
	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Mutation",
		Fields: fields,
	}), nil
}

func generateGraphQLInputArgs(collection models.Collection, extraArgs ...string) graphql.FieldConfigArgument {
	args := graphql.FieldConfigArgument{}

	// Add extra arguments (like ID for update/delete)
	for _, arg := range extraArgs {
		args[arg] = &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)}
	}

	// Map collection attributes to GraphQL arguments
	for _, attr := range collection.Attributes {
		var gqlType graphql.Input
		switch attr.Type {
		case "text":
			gqlType = graphql.String
		case "int":
			gqlType = graphql.Int
		case "bool":
			gqlType = graphql.Boolean
		case "float":
			gqlType = graphql.Float
		case "date":
			gqlType = graphql.String
		case "relation":
			// Relations can accept IDs or objects
			gqlType = graphql.ID
		default:
			logger.Log.Warnf("Unsupported attribute type: %s", attr.Type)
			continue
		}
		args[attr.Name] = &graphql.ArgumentConfig{Type: gqlType}
	}
	return args
}

func createCollectionItem(p graphql.ResolveParams, collection models.Collection) (interface{}, error) {
	itemData := map[string]interface{}{}

	for attrName := range p.Args {
		itemData[attrName] = p.Args[attrName]
	}

	newItem := models.Item{
		CollectionID: collection.ID,
		Data:         models.JSONMap(itemData),
	}

	if err := database.DB.Create(&newItem).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to create item in %s", collection.Name)
		return nil, err
	}

	logger.Log.Infof("Created new item in %s with ID: %d", collection.Name, newItem.ID)
	return newItem, nil
}

func updateCollectionItem(p graphql.ResolveParams, collection models.Collection) (interface{}, error) {
	id, _ := p.Args["id"].(string)

	var item models.Item
	if err := database.DB.Where("id = ?", id).First(&item).Error; err != nil {
		return nil, fmt.Errorf("item not found")
	}

	// Update attributes
	updatedData := item.Data
	for attrName := range p.Args {
		if attrName != "id" {
			updatedData[attrName] = p.Args[attrName]
		}
	}

	item.Data = updatedData
	if err := database.DB.Save(&item).Error; err != nil {
		return nil, err
	}

	logger.Log.Infof("Updated item %s ID: %s", collection.Name, id)
	return item, nil
}

func deleteCollectionItem(p graphql.ResolveParams, collection models.Collection) (interface{}, error) {
	id, _ := p.Args["id"].(string)

	if err := database.DB.Where("id = ?", id).Delete(&models.Item{}).Error; err != nil {
		return nil, fmt.Errorf("failed to delete item")
	}

	logger.Log.Infof("Deleted item in %s with ID: %s", collection.Name, id)
	return true, nil
}
