package graphql

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/internal/types" // Use the centralized types package
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/graphql-go/graphql"
	"gorm.io/gorm"
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

		// Generate the InputObject types for mutation arguments. Updates are
		// partial, so their input type has no required fields.
		gqlInputType := generateGraphQLInputType(localCollection, false)
		gqlUpdateInputType := generateGraphQLInputType(localCollection, true)

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
				return createCollectionItem(p, inputData, localCollection)
			},
		}

		// --- Mutation: Update Item ---
		fields["update"+localCollection.Name] = &graphql.Field{
			Type: gqlOutputType,
			Args: graphql.FieldConfigArgument{
				"id":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(gqlUpdateInputType)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				id, _ := p.Args["id"].(string)
				inputData, _ := p.Args["input"].(map[string]any)
				return updateCollectionItem(p, id, inputData, localCollection)
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
func generateGraphQLInputType(collection models.Collection, partial bool) *graphql.InputObject {
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
		if attr.Required && !partial {
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
	name := collection.Name + "Input"
	if partial {
		name = collection.Name + "UpdateInput"
	}
	return graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   name,
		Fields: fields,
	})
}

// --- Resolver Functions ---
//
// Every resolver checks the caller's role (set by the HTTP handler) with the
// same permission table as the REST API, and writes go through the storage
// layer so validation, nested relations and events behave identically.

func requirePermission(p graphql.ResolveParams, action string) error {
	role := auth.RoleFromContext(p.Context)
	if !auth.HasPermission(role, action) {
		return fmt.Errorf("access denied: role '%s' cannot %s items", role, action)
	}
	return nil
}

func createCollectionItem(p graphql.ResolveParams, inputData map[string]any, collection models.Collection) (any, error) {
	if err := requirePermission(p, auth.ActionCreate); err != nil {
		return nil, err
	}
	item, err := storage.SaveItem(collection, models.JSONMap(inputData))
	if err != nil {
		var verr *storage.ItemValidationError
		if errors.As(err, &verr) {
			return nil, verr
		}
		logger.Log.WithError(err).WithField("collection", collection.Name).Error("GraphQL create failed")
		return nil, fmt.Errorf("failed to create item in %s", collection.Name)
	}
	return mapItemToGraphQLResult(*item, collection.Attributes), nil
}

func updateCollectionItem(p graphql.ResolveParams, id string, inputData map[string]any, collection models.Collection) (any, error) {
	if err := requirePermission(p, auth.ActionUpdate); err != nil {
		return nil, err
	}
	itemID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid id '%s'", id)
	}
	existing, err := storage.GetItemByID(collection.ID, uint(itemID))
	if err != nil {
		return nil, fmt.Errorf("item with id %s not found in collection %s", id, collection.Name)
	}

	merged := models.JSONMap{}
	for k, v := range existing.Data {
		merged[k] = v
	}
	for k, v := range inputData {
		merged[k] = v
	}

	var processed models.JSONMap
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var verr error
		processed, verr = models.ValidateItemValues(collection, merged, tx)
		return verr
	}); err != nil {
		return nil, err
	}
	if err := storage.UpdateItem(existing.ID, processed); err != nil {
		return nil, fmt.Errorf("failed to update item %s", id)
	}
	existing.Data = processed
	return mapItemToGraphQLResult(*existing, collection.Attributes), nil
}

func deleteCollectionItem(p graphql.ResolveParams, collection models.Collection) (any, error) {
	if err := requirePermission(p, auth.ActionDelete); err != nil {
		return false, err
	}
	id, ok := p.Args["id"].(string)
	if !ok {
		return false, fmt.Errorf("invalid ID format")
	}
	itemID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid id '%s'", id)
	}
	// Scope the delete to this collection.
	if _, err := storage.GetItemByID(collection.ID, uint(itemID)); err != nil {
		return false, fmt.Errorf("item not found in collection %s", collection.Name)
	}
	if err := storage.DeleteItem(uint(itemID)); err != nil {
		return false, fmt.Errorf("failed to delete item")
	}
	return true, nil
}
