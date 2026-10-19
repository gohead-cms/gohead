// Package openapi describes the REST API as an OpenAPI 3.1 document,
// generated from the collections, components and singletons the same way
// the GraphQL schema is, so typed clients and Swagger UI always match the
// current content model.
package openapi

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
)

// obj is a JSON object.
type obj = map[string]any

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func schemaName(name string) string { return unsafeName.ReplaceAllString(name, "_") }

// reservedSchemas are the fixed schema names.
var reservedSchemas = map[string]bool{"Error": true, "Pagination": true, "Media": true, "Revision": true, "RevisionDetail": true}

// collectionNames maps each collection to a schema name that cannot clash
// with the fixed schemas, the Component_/Singleton_ names, or another
// collection's "<name>Input". It is filled by Build before any schema is
// generated.
var collectionNames = map[string]string{}

// assignCollectionNames gives every collection a unique schema name: its
// own name when possible, "Collection_<name>" otherwise.
func assignCollectionNames(collections []models.Collection) map[string]string {
	plain := map[string]bool{}
	for _, ct := range collections {
		plain[schemaName(ct.Name)] = true
	}
	out := map[string]string{}
	used := map[string]bool{}
	for _, ct := range collections {
		n := schemaName(ct.Name)
		clash := reservedSchemas[n] || strings.HasPrefix(n, "Component_") || strings.HasPrefix(n, "Singleton_") ||
			strings.HasPrefix(n, "Collection_") || (strings.HasSuffix(n, "Input") && plain[strings.TrimSuffix(n, "Input")])
		if clash {
			n = "Collection_" + n
		}
		for i := 2; used[n]; i++ { // names that sanitize to the same string
			n = fmt.Sprintf("%s_%d", strings.TrimRight(n, "_0123456789"), i)
		}
		used[n] = true
		out[ct.Name] = n
	}
	return out
}

// collectionSchema is the schema name of a collection ("" if unknown).
func collectionSchema(name string) string { return collectionNames[name] }

func ref(name string) obj { return obj{"$ref": "#/components/schemas/" + name} }

func nullable(s obj) obj { return obj{"oneOf": []any{s, obj{"type": "null"}}} }

// buildMu serializes builds, which share collectionNames.
var buildMu sync.Mutex

// Build returns the OpenAPI document for the current content model.
func Build(version string) (obj, error) {
	buildMu.Lock()
	defer buildMu.Unlock()
	var collections []models.Collection
	if err := database.DB.Preload("Attributes").Order("name").Find(&collections).Error; err != nil {
		return nil, err
	}
	var components []models.Component
	if err := database.DB.Preload("Attributes").Order("name").Find(&components).Error; err != nil {
		return nil, err
	}
	var singletons []models.Singleton
	if err := database.DB.Preload("Attributes").Order("name").Find(&singletons).Error; err != nil {
		return nil, err
	}

	collectionNames = assignCollectionNames(collections)
	schemas := baseSchemas()
	paths := obj{}
	tags := []any{}

	for _, cmp := range components {
		attrs := make([]models.Attribute, 0, len(cmp.Attributes))
		for _, a := range cmp.Attributes {
			attrs = append(attrs, a.ToAttribute())
		}
		props, required := properties(attrs, false)
		s := obj{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			s["required"] = required
		}
		if cmp.Description != "" {
			s["description"] = cmp.Description
		}
		schemas["Component_"+schemaName(cmp.Name)] = s
	}

	for _, ct := range collections {
		name := collectionSchema(ct.Name)
		schemas[name] = itemSchema(ct)
		schemas[name+"Input"] = inputSchema(ct.Attributes)
		tags = append(tags, obj{"name": ct.Name, "description": firstNonEmpty(ct.Description, "Items of the "+ct.Name+" collection")})
		for p, v := range collectionPaths(ct, name) {
			paths[p] = v
		}
	}

	for _, st := range singletons {
		name := "Singleton_" + schemaName(st.Name)
		props, _ := properties(st.Attributes, false)
		props["id"] = obj{"type": "integer"}
		props["createdAt"] = obj{"type": "string", "format": "date-time"}
		props["updatedAt"] = obj{"type": "string", "format": "date-time"}
		schemas[name] = obj{"type": "object", "properties": props, "required": []string{"id", "createdAt", "updatedAt"}}
		schemas[name+"Input"] = inputSchema(st.Attributes)
		tags = append(tags, obj{"name": "singleton:" + st.Name, "description": firstNonEmpty(st.Description, "The "+st.Name+" singleton")})
		paths["/api/singleton/"+st.Name] = singletonPath(st.Name, name)
		for p, v := range singletonHistoryPaths(st.Name, name) {
			paths[p] = v
		}
	}

	for p, v := range staticPaths() {
		paths[p] = v
	}
	tags = append(tags,
		obj{"name": "media", "description": "Uploaded files"},
		obj{"name": "auth", "description": "Sessions"},
		obj{"name": "graphql", "description": "GraphQL endpoint"},
		obj{"name": "history", "description": "Content revisions across entries"},
	)

	return obj{
		"openapi": "3.1.0",
		"info": obj{
			"title":       "GoHead API",
			"version":     version,
			"description": "REST API generated from this GoHead instance's content model. Authenticate with a user JWT (POST /auth/login) or an API token (ght_…).",
		},
		"servers":  []any{obj{"url": "/"}},
		"security": []any{obj{"bearerAuth": []any{}}},
		"tags":     tags,
		"paths":    paths,
		"components": obj{
			"schemas": schemas,
			"securitySchemes": obj{
				"bearerAuth": obj{"type": "http", "scheme": "bearer", "description": "A user JWT from POST /auth/login, or an API token (ght_…) from /admin/tokens."},
			},
			"responses": obj{
				"Error": obj{"description": "Error", "content": obj{"application/json": obj{"schema": ref("Error")}}},
			},
		},
	}, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// attributeSchema describes the value of one attribute in responses
// (input=false) or requests (input=true).
func attributeSchema(a models.Attribute, input bool) obj {
	var s obj
	switch a.Type {
	case "string", "text", "richtext", "email", "uid":
		s = obj{"type": "string"}
		if a.Type == "email" {
			s["format"] = "email"
		}
		if a.Pattern != "" {
			s["pattern"] = a.Pattern
		}
	case "password":
		s = obj{"type": "string", "description": "stored and returned as given"}
	case "int", "integer":
		s = obj{"type": "integer"}
		if a.Min != nil {
			s["minimum"] = *a.Min
		}
		if a.Max != nil {
			s["maximum"] = *a.Max
		}
	case "float", "decimal":
		s = obj{"type": "number"}
	case "bool", "boolean":
		s = obj{"type": "boolean"}
	case "date":
		s = obj{"type": "string", "format": "date"}
	case "datetime":
		s = obj{"type": "string", "format": "date-time"}
	case "time":
		s = obj{"type": "string"}
	case "enum", "enumeration":
		s = obj{"type": "string", "enum": a.Options}
	case "json":
		s = obj{"description": "Any JSON value"}
	case "media":
		one := obj{"oneOf": []any{ref("Media"), obj{"type": "integer", "description": "file ID (level=0)"}, obj{"type": "string", "description": "external URL"}}}
		if input {
			one = obj{"oneOf": []any{obj{"type": "integer", "description": "file ID from POST /api/media"}, obj{"type": "object", "properties": obj{"id": obj{"type": "integer"}}, "required": []string{"id"}}, obj{"type": "string", "description": "file ID, or an external URL"}}}
		}
		if a.Multiple {
			s = obj{"type": "array", "items": one}
		} else {
			s = one
		}
	case "relation":
		target := collectionSchema(a.Target)
		if target == "" { // target collection missing: IDs only
			s = obj{"description": "relation to the missing collection " + a.Target}
			break
		}
		one := obj{"oneOf": []any{ref(target), obj{"type": "integer", "description": "item ID (level=0, or beyond the requested level)"}}}
		if input {
			one = obj{"oneOf": []any{obj{"type": "integer", "description": "ID of an existing item"}, ref(target + "Input")}, "description": "an ID, {\"id\": …}, or a new item created in the same transaction"}
		}
		if a.Relation == "manyToMany" {
			s = obj{"type": "array", "items": one}
		} else {
			s = nullable(one)
		}
	case "component":
		c := ref("Component_" + schemaName(a.ComponentRef))
		if a.Repeatable {
			s = obj{"type": "array", "items": c}
			if a.Min != nil {
				s["minItems"] = *a.Min
			}
			if a.Max != nil {
				s["maxItems"] = *a.Max
			}
		} else {
			s = c
		}
	default:
		s = obj{"description": "type " + a.Type}
	}
	return s
}

func properties(attrs []models.Attribute, input bool) (obj, []string) {
	props := obj{}
	var required []string
	for _, a := range attrs {
		props[a.Name] = attributeSchema(a, input)
		if a.Required {
			required = append(required, a.Name)
		}
	}
	sort.Strings(required)
	return props, required
}

// itemSchema is an item as returned: attributes plus system fields.
func itemSchema(ct models.Collection) obj {
	props, _ := properties(ct.Attributes, false)
	props["id"] = obj{"type": "integer"}
	props["createdAt"] = obj{"type": "string", "format": "date-time"}
	props["updatedAt"] = obj{"type": "string", "format": "date-time"}
	required := []string{"id", "createdAt", "updatedAt"}
	if ct.DraftAndPublish {
		props["status"] = obj{"type": "string", "enum": []string{models.StatusDraft, models.StatusPublished}}
		props["publishedAt"] = nullable(obj{"type": "string", "format": "date-time"})
		required = append(required, "status", "publishedAt")
	}
	s := obj{"type": "object", "properties": props, "required": required}
	if ct.Description != "" {
		s["description"] = ct.Description
	}
	return s
}

// inputSchema is the "data" of a write request.
func inputSchema(attrs []models.Attribute) obj {
	props, required := properties(attrs, true)
	s := obj{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}
