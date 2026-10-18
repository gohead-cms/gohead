package graphql

import (
	"fmt"
	"regexp"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/internal/types"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/graphql-go/graphql"
)

// typeRegistry caches the GraphQL object built for each collection during a
// schema build, so that relations can refer to the same type. It is reset on
// every (re)build so that schema changes are picked up; builds are serialized
// by buildMu in InitializeGraphQLSchema.
var typeRegistry = make(map[string]*graphql.Object)

// componentTypes and componentInputs cache the types built for components.
var (
	componentTypes  = make(map[string]*graphql.Object)
	componentInputs = make(map[string]*graphql.InputObject)
)

// ResetTypeRegistry drops all cached collection and component types.
func ResetTypeRegistry() {
	typeRegistry = make(map[string]*graphql.Object)
	componentTypes = make(map[string]*graphql.Object)
	componentInputs = make(map[string]*graphql.InputObject)
}

var nonNameChars = regexp.MustCompile(`[^_0-9A-Za-z]`)

// componentTypeName returns a valid GraphQL type name for a component.
func componentTypeName(name string) string {
	return "Component_" + nonNameChars.ReplaceAllString(name, "_")
}

// componentOutputType returns the GraphQL object for a component. Values
// are stored as plain objects, so the default resolver reads the fields.
func componentOutputType(name string) (*graphql.Object, error) {
	if t, ok := componentTypes[name]; ok {
		return t, nil
	}
	cmp, err := storage.GetComponentByName(name)
	if err != nil {
		return nil, err
	}
	obj := graphql.NewObject(graphql.ObjectConfig{
		Name: componentTypeName(cmp.Name),
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			fields := graphql.Fields{}
			for _, a := range cmp.Attributes {
				t, err := componentFieldOutput(a.BaseAttribute)
				if err != nil {
					logger.Log.WithError(err).WithField("attribute", a.Name).Debug("GraphQL: skipping component attribute")
					continue
				}
				fields[a.Name] = &graphql.Field{Type: t}
			}
			if len(fields) == 0 {
				fields["_empty"] = &graphql.Field{Type: graphql.String}
			}
			return fields
		}),
	})
	componentTypes[name] = obj
	return obj, nil
}

func componentFieldOutput(a models.BaseAttribute) (graphql.Output, error) {
	if a.Type != "component" {
		return types.GetGraphQLType(a.Type)
	}
	t, err := componentOutputType(a.ComponentRef)
	if err != nil {
		return nil, err
	}
	if a.Repeatable {
		return graphql.NewList(t), nil
	}
	return t, nil
}

// componentInputType returns the GraphQL input object for a component.
func componentInputType(name string) (*graphql.InputObject, error) {
	if t, ok := componentInputs[name]; ok {
		return t, nil
	}
	cmp, err := storage.GetComponentByName(name)
	if err != nil {
		return nil, err
	}
	in := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: componentTypeName(cmp.Name) + "Input",
		Fields: graphql.InputObjectConfigFieldMapThunk(func() graphql.InputObjectConfigFieldMap {
			fields := graphql.InputObjectConfigFieldMap{}
			for _, a := range cmp.Attributes {
				t, err := componentFieldInput(a.BaseAttribute)
				if err != nil {
					continue
				}
				if a.Required {
					t = graphql.NewNonNull(t)
				}
				fields[a.Name] = &graphql.InputObjectFieldConfig{Type: t}
			}
			if len(fields) == 0 {
				fields["_empty"] = &graphql.InputObjectFieldConfig{Type: graphql.String}
			}
			return fields
		}),
	})
	componentInputs[name] = in
	return in, nil
}

func componentFieldInput(a models.BaseAttribute) (graphql.Input, error) {
	if a.Type == "component" {
		t, err := componentInputType(a.ComponentRef)
		if err != nil {
			return nil, err
		}
		if a.Repeatable {
			return graphql.NewList(graphql.NewNonNull(t)), nil
		}
		return t, nil
	}
	out, err := types.GetGraphQLType(a.Type)
	if err != nil {
		return nil, err
	}
	in, ok := out.(graphql.Input)
	if !ok {
		return nil, fmt.Errorf("type '%s' cannot be used as input", a.Type)
	}
	return in, nil
}

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
				"id":        &graphql.Field{Type: graphql.ID},
				"createdAt": &graphql.Field{Type: graphql.String},
				"updatedAt": &graphql.Field{Type: graphql.String},
			}
			if collection.DraftAndPublish {
				fields["publishedAt"] = &graphql.Field{Type: graphql.String}
				fields["status"] = &graphql.Field{Type: graphql.String}
			}

			// 2. Loop through attributes and build all fields.
			for _, attr := range collection.Attributes {
				localAttr := attr

				var gqlFieldType graphql.Output
				var resolveFunc graphql.FieldResolveFn
				var err error

				if localAttr.Type == "relation" {
					targetType, err := GetOrCreateGraphQLType(localAttr.Target)
					if err != nil {
						// Skip the field rather than crash schema generation.
						logger.Log.WithError(err).WithField("attribute", localAttr.Name).
							Warn("GraphQL: skipping relation with unknown target")
						continue
					}
					gqlFieldType = targetType
					// Matches storage: only manyToMany holds a list of IDs.
					if localAttr.Relation == "manyToMany" {
						gqlFieldType = graphql.NewList(targetType)
					}
					resolveFunc = func(p graphql.ResolveParams) (any, error) {
						return ResolveRelation(p, collection.ID, localAttr)
					}
				} else if localAttr.Type == "component" {
					gqlFieldType, err = componentFieldOutput(models.BaseAttribute{
						Type: "component", ComponentRef: localAttr.ComponentRef, Repeatable: localAttr.Repeatable,
					})
					if err != nil {
						logger.Log.WithError(err).WithField("attribute", localAttr.Name).
							Warn("GraphQL: skipping component attribute with unknown component")
						continue
					}
					resolveFunc = func(p graphql.ResolveParams) (any, error) {
						if sourceMap, ok := p.Source.(map[string]any); ok {
							return sourceMap[localAttr.Name], nil
						}
						return nil, nil
					}
				} else {
					gqlFieldType, err = types.GetGraphQLType(localAttr.Type)
					if err != nil {
						// e.g. dynamiczone attributes are not exposed yet; skip
						// them instead of crashing schema generation.
						logger.Log.WithError(err).WithField("attribute", localAttr.Name).
							Debug("GraphQL: skipping attribute with unsupported type")
						continue
					}
					resolveFunc = func(p graphql.ResolveParams) (any, error) {
						if sourceMap, ok := p.Source.(map[string]any); ok {
							if value, exists := sourceMap[localAttr.Name]; exists {
								return value, nil
							}
						}
						return nil, nil // Return null if the key doesn't exist.
					}
				}

				fields[localAttr.Name] = &graphql.Field{
					Type:    gqlFieldType,
					Resolve: resolveFunc,
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
