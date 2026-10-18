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

		createArgs := graphql.FieldConfigArgument{
			// Use the generated InputObject for the 'input' argument.
			"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(gqlInputType)},
		}
		updateArgs := graphql.FieldConfigArgument{
			"id":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
			"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(gqlUpdateInputType)},
		}
		if localCollection.DraftAndPublish {
			statusArg := &graphql.ArgumentConfig{
				Type:        graphql.String,
				Description: "draft or published. New items are drafts by default; updates keep the current status.",
			}
			createArgs["status"] = statusArg
			updateArgs["status"] = statusArg
		}

		// --- Mutation: Create Item ---
		fields["create"+localCollection.Name] = &graphql.Field{
			Type: gqlOutputType,
			Args: createArgs,
			Resolve: func(p graphql.ResolveParams) (any, error) {
				// Extract the input map from the arguments.
				inputData, _ := p.Args["input"].(map[string]any)
				return createCollectionItem(p, inputData, localCollection)
			},
		}

		// --- Mutation: Update Item ---
		fields["update"+localCollection.Name] = &graphql.Field{
			Type: gqlOutputType,
			Args: updateArgs,
			Resolve: func(p graphql.ResolveParams) (any, error) {
				id, _ := p.Args["id"].(string)
				inputData, _ := p.Args["input"].(map[string]any)
				return updateCollectionItem(p, id, inputData, localCollection)
			},
		}

		// --- Mutations: Publish / Unpublish ---
		if localCollection.DraftAndPublish {
			for _, publish := range []bool{true, false} {
				name := "publish" + localCollection.Name
				if !publish {
					name = "unpublish" + localCollection.Name
				}
				fields[name] = &graphql.Field{
					Type: gqlOutputType,
					Args: graphql.FieldConfigArgument{
						"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					},
					Resolve: func(p graphql.ResolveParams) (any, error) {
						return setPublished(p, localCollection, publish)
					},
				}
			}
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
			if attr.Relation == "manyToMany" {
				gqlType = graphql.NewList(graphql.ID)
			}
		} else if attr.Type == "component" {
			in, err := componentFieldInput(models.BaseAttribute{
				Type: "component", ComponentRef: attr.ComponentRef, Repeatable: attr.Repeatable,
			})
			if err != nil {
				logger.Log.Warnf("Skipping attribute '%s' in '%sInput': %v", attr.Name, collection.Name, err)
				continue
			}
			gqlType = in
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

// statusArg converts the optional "status" argument of a write mutation.
func statusArg(p graphql.ResolveParams) (*bool, error) {
	status, ok := p.Args["status"].(string)
	if !ok {
		return nil, nil
	}
	switch status {
	case models.StatusPublished:
		t := true
		return &t, nil
	case models.StatusDraft:
		f := false
		return &f, nil
	}
	return nil, fmt.Errorf("invalid status '%s' (expected draft or published)", status)
}

func createCollectionItem(p graphql.ResolveParams, inputData map[string]any, collection models.Collection) (any, error) {
	collection = currentCollection(collection)
	if err := requirePermission(p, auth.ActionCreate); err != nil {
		return nil, err
	}
	publish, err := statusArg(p)
	if err != nil {
		return nil, err
	}
	items, err := storage.CreateItems(collection, []storage.ItemInput{{
		Data:    models.JSONMap(inputData),
		Publish: publish != nil && *publish,
	}})
	if err != nil {
		var verr *storage.ItemValidationError
		if errors.As(err, &verr) {
			return nil, verr
		}
		logger.Log.WithError(err).WithField("collection", collection.Name).Error("GraphQL create failed")
		return nil, fmt.Errorf("failed to create item in %s", collection.Name)
	}
	return mapItemToGraphQLResult(*items[0], collection, false), nil
}

func updateCollectionItem(p graphql.ResolveParams, id string, inputData map[string]any, collection models.Collection) (any, error) {
	collection = currentCollection(collection)
	if err := requirePermission(p, auth.ActionUpdate); err != nil {
		return nil, err
	}
	publish, err := statusArg(p)
	if err != nil {
		return nil, err
	}
	itemID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid id '%s'", id)
	}
	existing, err := storage.GetItemByID(collection.ID, uint(itemID))
	if err != nil {
		return nil, fmt.Errorf("item with id %s not found in collection %s", id, collection.Name)
	}

	// Updates are partial: merge the input over the stored data.
	merged := models.JSONMap{}
	for k, v := range existing.Data {
		merged[k] = v
	}
	for k, v := range inputData {
		merged[k] = v
	}

	item, err := storage.UpdateItemContent(collection, existing.ID, merged, publish)
	if err != nil {
		var verr *storage.ItemValidationError
		if errors.As(err, &verr) {
			return nil, verr
		}
		logger.Log.WithError(err).WithField("collection", collection.Name).Error("GraphQL update failed")
		return nil, fmt.Errorf("failed to update item %s", id)
	}
	return mapItemToGraphQLResult(*item, collection, false), nil
}

func setPublished(p graphql.ResolveParams, collection models.Collection, publish bool) (any, error) {
	collection = currentCollection(collection)
	if err := requirePermission(p, auth.ActionUpdate); err != nil {
		return nil, err
	}
	id, _ := p.Args["id"].(string)
	itemID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid id '%s'", id)
	}
	item, err := storage.SetItemPublished(collection, uint(itemID), publish)
	if err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			return nil, fmt.Errorf("item with id %s not found in collection %s", id, collection.Name)
		}
		return nil, err
	}
	return mapItemToGraphQLResult(*item, collection, false), nil
}
func deleteCollectionItem(p graphql.ResolveParams, collection models.Collection) (any, error) {
	if err := requirePermission(p, auth.ActionDelete); err != nil {
		return false, err
	}
	id, ok := p.Args["id"].(string)
	if !ok {
		return false, fmt.Errorf("invalid ID format")
	}
	itemID, err := strconv.ParseUint(id, 10, 32)
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
