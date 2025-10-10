package graphql

import (
	"fmt"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/internal/types" // Use the centralized types package
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/graphql-go/graphql"
)

// GenerateGraphQLMutations creates the root GraphQL mutation object for all collections.
func GenerateGraphQLMutations() (*graphql.Object, error) {
	fields := graphql.Fields{}

	// 1. Fetch all collections from the database.
	var collections []models.Collection
	if err := database.DB.Preload("Attributes").Find(&collections).Error; err != nil {
		return nil, err
	}

	// 2. Loop through each collection to generate its specific mutations.
	for _, collection := range collections {
		// We need a local copy for the closure to capture it correctly.
		localCollection := collection

		// Generate the output type for returning data.
		gqlOutputType, err := ConvertCollectionToGraphQLType(localCollection)
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to generate GraphQL type for collection: %s", localCollection.Name)
			return nil, err
		}

		// Generate the InputObject type for mutation arguments.
		gqlInputType := generateGraphQLInputType(localCollection)

		// --- Mutation: Create Item ---
		fields["create"+localCollection.Name] = &graphql.Field{
			Type: gqlOutputType,
			Args: graphql.FieldConfigArgument{
				// Use the generated InputObject for the 'input' argument.
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(gqlInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				// Extract the input map from the arguments.
				inputData, _ := p.Args["input"].(map[string]any)
				return createCollectionItem(inputData, localCollection)
			},
		}

		// --- Mutation: Update Item ---
		fields["update"+localCollection.Name] = &graphql.Field{
			Type: gqlOutputType,
			Args: graphql.FieldConfigArgument{
				"id":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(gqlInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				id, _ := p.Args["id"].(string)
				inputData, _ := p.Args["input"].(map[string]any)
				return updateCollectionItem(id, inputData, localCollection)
			},
		}

		// --- Mutation: Delete Item ---
		fields["delete"+localCollection.Name] = &graphql.Field{
			Type: graphql.Boolean,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				return deleteCollectionItem(p, localCollection)
			},
		}
	}
	if len(fields) == 0 {
		fields["_placeholder"] = &graphql.Field{
			Type:        graphql.String,
			Description: "This is a placeholder mutation. Create a collection to see real mutations here.",
		}
	}
	// 3. Return the root mutation object.
	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Mutation",
		Fields: fields,
	}), nil
}

// generateGraphQLInputType dynamically creates a GraphQL InputObject using the centralized TypeRegistry.
func generateGraphQLInputType(collection models.Collection) *graphql.InputObject {
	// (This is the function from the previous step, included here for completeness)
	fields := graphql.InputObjectConfigFieldMap{}
	for _, attr := range collection.Attributes {
		var gqlType graphql.Input
		if attr.Type == "relation" {
			gqlType = graphql.ID
		} else {
			outputType, err := types.GetGraphQLType(attr.Type)
			if err != nil {
				logger.Log.Warnf("Skipping attribute '%s' in '%sInput': %v", attr.Name, collection.Name, err)
				continue
			}
			var ok bool
			gqlType, ok = outputType.(graphql.Input)
			if !ok {
				logger.Log.Warnf("Skipping attribute '%s': Type '%s' is not a valid GraphQL Input type", attr.Name, attr.Type)
				continue
			}
		}
		fieldConfig := &graphql.InputObjectFieldConfig{Type: gqlType}
		if attr.Required {
			fieldConfig.Type = graphql.NewNonNull(gqlType)
		}
		fields[attr.Name] = fieldConfig
	}
	if len(fields) == 0 {
		fields["_placeholder"] = &graphql.InputObjectFieldConfig{
			Type:        graphql.String,
			Description: "This is a placeholder mutation. Create a collection to see real mutations here.",
		}
	}
	return graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   collection.Name + "Input",
		Fields: fields,
	})
}

// --- Resolver Functions (with security fixes) ---

func createCollectionItem(inputData map[string]any, collection models.Collection) (any, error) {
	itemData := map[string]any{}
	// Avoid mass assignment by iterating over defined attributes.
	for _, attr := range collection.Attributes {
		if val, ok := inputData[attr.Name]; ok {
			itemData[attr.Name] = val
		}
	}

	item := models.Item{
		CollectionID: collection.ID,
		Data:         models.JSONMap(itemData),
	}
	if err := database.DB.Create(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to create item in %s: %w", collection.Name, err)
	}
	return item, nil
}

func updateCollectionItem(id string, inputData map[string]any, collection models.Collection) (any, error) {
	var item models.Item
	if err := database.DB.Where("id = ? AND collection_id = ?", id, collection.ID).First(&item).Error; err != nil {
		return nil, fmt.Errorf("item with id %s not found in collection %s", id, collection.Name)
	}

	updatedData := item.Data
	for _, attr := range collection.Attributes {
		if val, ok := inputData[attr.Name]; ok {
			updatedData[attr.Name] = val
		}
	}

	item.Data = updatedData
	if err := database.DB.Save(&item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func deleteCollectionItem(p graphql.ResolveParams, collection models.Collection) (any, error) {
	id, ok := p.Args["id"].(string)
	if !ok {
		return false, fmt.Errorf("invalid ID format")
	}

	result := database.DB.Where("id = ? AND collection_id = ?", id, collection.ID).Delete(&models.Item{})
	if result.Error != nil {
		return false, fmt.Errorf("failed to delete item")
	}
	if result.RowsAffected == 0 {
		return false, fmt.Errorf("item not found in collection %s", collection.Name)
	}
	return true, nil
}
