package openapi

import (
	"github.com/gohead-cms/gohead/internal/models"
)

func jsonContent(schema obj) obj {
	return obj{"application/json": obj{"schema": schema}}
}

func dataEnvelope(schema obj, meta obj) obj {
	props := obj{"data": schema}
	if meta != nil {
		props["meta"] = meta
	}
	return obj{"type": "object", "properties": props, "required": []string{"data"}}
}

func ok(description string, schema obj) obj {
	return obj{"description": description, "content": jsonContent(schema)}
}

func errorResponses(codes ...string) obj {
	out := obj{}
	for _, c := range codes {
		out[c] = obj{"$ref": "#/components/responses/Error"}
	}
	return out
}

func merge(a, b obj) obj {
	for k, v := range b {
		a[k] = v
	}
	return a
}

var (
	paramID    = obj{"name": "id", "in": "path", "required": true, "schema": obj{"type": "integer"}}
	paramLevel = obj{"name": "level", "in": "query", "description": "How deep relations are populated (0: IDs only).", "schema": obj{"type": "integer", "minimum": 0, "maximum": 5, "default": 1}}
)

func statusParam(single bool) obj {
	desc := "Draft-and-publish collections: published (default), draft or all. Drafts need the update permission."
	if single {
		desc = "published to hide drafts; editors get drafts by ID by default."
	}
	return obj{"name": "status", "in": "query", "description": desc, "schema": obj{"type": "string", "enum": []string{"published", "draft", "all"}}}
}

func collectionPaths(ct models.Collection, name string) obj {
	tag := []string{ct.Name}
	item, input := ref(name), ref(name+"Input")
	writeBody := obj{"type": "object", "properties": obj{"data": input}, "required": []string{"data"}}
	if ct.DraftAndPublish {
		writeBody["properties"].(obj)["status"] = obj{"type": "string", "enum": []string{models.StatusDraft, models.StatusPublished}}
	}

	listParams := []any{
		obj{"name": "page", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "default": 1}},
		obj{"name": "pageSize", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "maximum": 100, "default": 10}},
		obj{"name": "sort", "in": "query", "description": "field[:asc|:desc],… (attributes, id, createdAt, updatedAt, publishedAt)", "schema": obj{"type": "string"}},
		obj{"name": "filters", "in": "query", "style": "deepObject", "explode": true,
			"description": "filters[field]=value or filters[field][$op]=value; ops: $eq $ne $lt $lte $gt $gte $contains",
			"schema":      obj{"type": "object", "additionalProperties": true}},
		paramLevel,
	}
	singleParams := []any{paramID, paramLevel}
	if ct.DraftAndPublish {
		listParams = append(listParams, statusParam(false))
		singleParams = append(singleParams, statusParam(true))
	}

	paths := obj{
		"/api/collections/" + ct.Name: obj{
			"get": obj{
				"tags": tag, "operationId": "list_" + name, "summary": "List " + ct.Name,
				"parameters": listParams,
				"responses":  merge(obj{"200": ok("Items", dataEnvelope(obj{"type": "array", "items": item}, obj{"type": "object", "properties": obj{"pagination": ref("Pagination")}}))}, errorResponses("400", "401", "403")),
			},
			"post": obj{
				"tags": tag, "operationId": "create_" + name, "summary": "Create one item, or a batch (all or nothing)",
				"parameters":  []any{paramLevel},
				"requestBody": obj{"required": true, "content": jsonContent(obj{"oneOf": []any{writeBody, obj{"type": "array", "items": writeBody, "minItems": 1}}})},
				"responses":   merge(obj{"201": ok("Created", dataEnvelope(obj{"oneOf": []any{item, obj{"type": "array", "items": item}}}, nil))}, errorResponses("400", "401", "403")),
			},
		},
		"/api/collections/" + ct.Name + "/{id}": obj{
			"get": obj{
				"tags": tag, "operationId": "get_" + name, "summary": "Get one item",
				"parameters": singleParams,
				"responses":  merge(obj{"200": ok("Item", dataEnvelope(item, nil))}, errorResponses("401", "403", "404")),
			},
			"put": obj{
				"tags": tag, "operationId": "update_" + name, "summary": "Replace an item's data",
				"parameters":  []any{paramID, paramLevel},
				"requestBody": obj{"required": true, "content": jsonContent(writeBody)},
				"responses":   merge(obj{"200": ok("Updated", dataEnvelope(item, nil))}, errorResponses("400", "401", "403", "404")),
			},
			"delete": obj{
				"tags": tag, "operationId": "delete_" + name, "summary": "Delete an item",
				"parameters": []any{paramID},
				"responses":  merge(obj{"200": ok("Deleted", obj{"type": "object", "properties": obj{"data": obj{"type": "null"}}})}, errorResponses("401", "403", "404")),
			},
		},
	}
	if ct.DraftAndPublish {
		for _, action := range []string{"publish", "unpublish"} {
			paths["/api/collections/"+ct.Name+"/{id}/"+action] = obj{"post": obj{
				"tags": tag, "operationId": action + "_" + name, "summary": action + " an item",
				"parameters": []any{paramID, paramLevel},
				"responses":  merge(obj{"200": ok("Item", dataEnvelope(item, nil))}, errorResponses("400", "401", "403", "404")),
			}}
		}
	}
	return paths
}

func singletonPath(stName, name string) obj {
	tag := []string{"singleton:" + stName}
	return obj{
		"get": obj{
			"tags": tag, "operationId": "get_" + name, "summary": "Get the " + stName + " singleton",
			"parameters": []any{paramLevel},
			"responses":  merge(obj{"200": ok("Content", dataEnvelope(ref(name), nil))}, errorResponses("401", "403", "404")),
		},
		"put": obj{
			"tags": tag, "operationId": "put_" + name, "summary": "Create or replace the " + stName + " singleton",
			"parameters":  []any{paramLevel},
			"requestBody": obj{"required": true, "content": jsonContent(obj{"type": "object", "properties": obj{"data": ref(name + "Input")}, "required": []string{"data"}})},
			"responses":   merge(obj{"200": ok("Saved", dataEnvelope(ref(name), nil)), "201": ok("Created", dataEnvelope(ref(name), nil))}, errorResponses("400", "401", "403", "404")),
		},
	}
}

func staticPaths() obj {
	media := ref("Media")
	noAuth := []any{}
	credentials := obj{"type": "object", "properties": obj{"username": obj{"type": "string"}, "password": obj{"type": "string", "format": "password"}}, "required": []string{"username", "password"}}
	return obj{
		"/auth/login": obj{"post": obj{
			"tags": []string{"auth"}, "operationId": "login", "summary": "Get a JWT (72 hours)", "security": noAuth,
			"requestBody": obj{"required": true, "content": jsonContent(credentials)},
			"responses":   merge(obj{"200": ok("Token", dataEnvelope(obj{"type": "object", "properties": obj{"token": obj{"type": "string"}}}, nil))}, errorResponses("401", "429")),
		}},
		"/auth/register": obj{"post": obj{
			"tags": []string{"auth"}, "operationId": "register", "summary": "Create a viewer account (when auth.registration is open)", "security": noAuth,
			"requestBody": obj{"required": true, "content": jsonContent(merge(obj{}, obj{"type": "object", "properties": obj{"username": obj{"type": "string"}, "email": obj{"type": "string", "format": "email"}, "password": obj{"type": "string", "format": "password"}}, "required": []string{"username", "email", "password"}}))},
			"responses":   merge(obj{"201": obj{"description": "Created"}}, errorResponses("400", "403", "409", "429")),
		}},
		"/api/media": obj{
			"get": obj{
				"tags": []string{"media"}, "operationId": "list_media", "summary": "List the media library (create permission)",
				"parameters": []any{
					obj{"name": "page", "in": "query", "schema": obj{"type": "integer", "minimum": 1}},
					obj{"name": "pageSize", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "maximum": 100}},
					obj{"name": "mime", "in": "query", "description": "type prefix, e.g. image/", "schema": obj{"type": "string"}},
					obj{"name": "search", "in": "query", "description": "part of the file name", "schema": obj{"type": "string"}},
				},
				"responses": merge(obj{"200": ok("Files", dataEnvelope(obj{"type": "array", "items": media}, obj{"type": "object", "properties": obj{"pagination": ref("Pagination")}}))}, errorResponses("401", "403")),
			},
			"post": obj{
				"tags": []string{"media"}, "operationId": "upload_media", "summary": "Upload files (all or nothing)",
				"requestBody": obj{"required": true, "content": obj{"multipart/form-data": obj{"schema": obj{
					"type": "object",
					"properties": obj{
						"files":   obj{"type": "array", "items": obj{"type": "string", "format": "binary"}},
						"alt":     obj{"type": "string"},
						"caption": obj{"type": "string"},
					},
					"required": []string{"files"},
				}}}},
				"responses": merge(obj{"201": ok("Uploaded", dataEnvelope(obj{"type": "array", "items": media}, nil))}, errorResponses("400", "401", "403", "413", "415", "503")),
			},
		},
		"/api/media/{id}": obj{
			"get": obj{"tags": []string{"media"}, "operationId": "get_media", "parameters": []any{paramID},
				"responses": merge(obj{"200": ok("File", dataEnvelope(media, nil))}, errorResponses("401", "403", "404"))},
			"put": obj{"tags": []string{"media"}, "operationId": "update_media", "parameters": []any{paramID},
				"requestBody": obj{"required": true, "content": jsonContent(obj{"type": "object", "properties": obj{"data": obj{"type": "object", "properties": obj{"name": obj{"type": "string"}, "alt": obj{"type": "string"}, "caption": obj{"type": "string"}}}}, "required": []string{"data"}})},
				"responses":   merge(obj{"200": ok("File", dataEnvelope(media, nil))}, errorResponses("400", "401", "403", "404"))},
			"delete": obj{"tags": []string{"media"}, "operationId": "delete_media", "parameters": []any{paramID},
				"responses": merge(obj{"200": obj{"description": "Deleted"}}, errorResponses("401", "403", "404", "503"))},
		},
		"/api/graphql": obj{"post": obj{
			"tags": []string{"graphql"}, "operationId": "graphql", "summary": "GraphQL queries and mutations",
			"requestBody": obj{"required": true, "content": jsonContent(obj{"type": "object", "properties": obj{"query": obj{"type": "string"}, "variables": obj{"type": "object"}, "operationName": obj{"type": "string"}}, "required": []string{"query"}})},
			"responses":   merge(obj{"200": obj{"description": "GraphQL result"}}, errorResponses("400", "401")),
		}},
	}
}

func baseSchemas() obj {
	return obj{
		"Error": obj{"type": "object", "properties": obj{"error": obj{"type": "object", "properties": obj{
			"status": obj{"type": "integer"}, "name": obj{"type": "string"}, "message": obj{}, "details": obj{},
		}, "required": []string{"status", "name", "message"}}}, "required": []string{"error"}},
		"Pagination": obj{"type": "object", "properties": obj{
			"page": obj{"type": "integer"}, "pageSize": obj{"type": "integer"}, "total": obj{"type": "integer"}, "pageCount": obj{"type": "integer"},
		}},
		"Media": obj{"type": "object", "properties": obj{
			"id": obj{"type": "integer"}, "url": obj{"type": "string"}, "name": obj{"type": "string"}, "mime": obj{"type": "string"},
			"size": obj{"type": "integer"}, "width": nullable(obj{"type": "integer"}), "height": nullable(obj{"type": "integer"}),
			"hash": obj{"type": "string"}, "alt": obj{"type": "string"}, "caption": obj{"type": "string"}, "uploadedBy": obj{"type": "string"},
			"createdAt": obj{"type": "string", "format": "date-time"}, "updatedAt": obj{"type": "string", "format": "date-time"},
		}, "required": []string{"id", "url", "name", "mime", "size"}},
	}
}
