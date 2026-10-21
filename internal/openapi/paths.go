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

// localeParams are the query parameters of reads in localized types.
func localeParams() []any {
	return []any{
		obj{"name": "locale", "in": "query", "description": "Locale of localized attributes (GET /api/locales); default: the default locale.", "schema": obj{"type": "string"}},
		obj{"name": "fallback", "in": "query", "description": "Missing localized values fall back to the default locale (default true); false leaves them out, even required ones.", "schema": obj{"type": "boolean", "default": true}},
	}
}

// writeLocaleParam is the locale of a write: another locale than the
// default replaces that locale's localized values (a translation).
var writeLocaleParam = obj{"name": "locale", "in": "query", "description": "Write the values of this locale (localized attributes only; {} removes the translation). Default: the default locale.", "schema": obj{"type": "string"}}

// addLocaleProps adds the locale fields of localized entries.
func addLocaleProps(props obj) {
	props["locale"] = obj{"type": "string", "description": "Locale of the localized values"}
	props["locales"] = obj{"type": "array", "items": obj{"type": "string"}, "description": "Locales with content: the default, then translations"}
}

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
	putBody := writeBody
	if models.HasLocalized(ct.Attributes) {
		// With ?locale= another locale: a translation.
		putBody = obj{"type": "object", "properties": obj{"data": obj{"anyOf": []any{input, ref(name + "Translation")}}}, "required": []string{"data"}}
	}
	if ct.DraftAndPublish {
		writeBody["properties"].(obj)["status"] = obj{"type": "string", "enum": []string{models.StatusDraft, models.StatusPublished}}
		putBody["properties"].(obj)["status"] = obj{"type": "string", "enum": []string{models.StatusDraft, models.StatusPublished}}
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
	writeParams := []any{paramID, paramLevel}
	if ct.DraftAndPublish {
		listParams = append(listParams, statusParam(false))
		singleParams = append(singleParams, statusParam(true))
	}
	if models.HasLocalized(ct.Attributes) {
		listParams = append(listParams, localeParams()...)
		singleParams = append(singleParams, localeParams()...)
		writeParams = append(writeParams, writeLocaleParam)
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
				"parameters":  writeParams,
				"requestBody": obj{"required": true, "content": jsonContent(putBody)},
				"responses":   merge(obj{"200": ok("Updated", dataEnvelope(item, nil))}, errorResponses("400", "401", "403", "404")),
			},
			"delete": obj{
				"tags": tag, "operationId": "delete_" + name, "summary": "Delete an item",
				"parameters": []any{paramID},
				"responses":  merge(obj{"200": ok("Deleted", obj{"type": "object", "properties": obj{"data": obj{"type": "null"}}})}, errorResponses("401", "403", "404")),
			},
		},
	}
	for p, v := range historyPaths("/api/collections/"+ct.Name+"/{id}", tag, name, []any{paramID}, item) {
		paths[p] = v
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

// historyPaths describes the revision routes under an entry's path.
// params are the entry's path parameters; restored is the content schema.
func historyPaths(base string, tag []string, name string, params []any, restored obj) obj {
	version := obj{"name": "version", "in": "path", "required": true, "schema": obj{"type": "integer", "minimum": 1}}
	withVersion := append(append([]any{}, params...), version)
	compare := obj{"name": "compare", "in": "query", "description": "Version to diff against: a number, or current. Default: the previous version.", "schema": obj{"type": "string"}}
	pageParams := []any{
		obj{"name": "page", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "default": 1}},
		obj{"name": "pageSize", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "maximum": 100, "default": 20}},
	}
	return obj{
		base + "/revisions": obj{"get": obj{
			"tags": tag, "operationId": "list_revisions_" + name, "summary": "History of changes, newest first (editors and admins)",
			"parameters": append(append([]any{}, params...), pageParams...),
			"responses": merge(obj{"200": ok("Revisions", dataEnvelope(obj{"type": "array", "items": ref("Revision")},
				obj{"type": "object", "properties": obj{"pagination": ref("Pagination")}}))}, errorResponses("400", "401", "403", "404")),
		}},
		base + "/revisions/{version}": obj{"get": obj{
			"tags": tag, "operationId": "get_revision_" + name, "summary": "One version, with its data and the changes from another version",
			"parameters": append(withVersion, compare),
			"responses":  merge(obj{"200": ok("Revision", dataEnvelope(ref("RevisionDetail"), nil))}, errorResponses("400", "401", "403", "404")),
		}},
		base + "/revisions/{version}/restore": obj{"post": obj{
			"tags": tag, "operationId": "restore_revision_" + name, "summary": "Make a version the current content (a new revision; brings back deleted content)",
			"parameters": withVersion,
			"responses": merge(obj{"200": ok("Restored", dataEnvelope(restored, obj{"type": "object", "properties": obj{
				"revision":  ref("Revision"),
				"dropped":   obj{"type": "array", "items": obj{"type": "string"}, "description": "Attributes of the old version the type no longer has"},
				"undeleted": obj{"type": "boolean"},
				"unchanged": obj{"type": "boolean", "description": "The entry already had this content: no revision was recorded"},
			}}))}, errorResponses("400", "401", "403", "404")),
		}},
	}
}

func singletonPath(stName, name string, localized bool) obj {
	tag := []string{"singleton:" + stName}
	readParams, writeParams := []any{paramLevel}, []any{paramLevel}
	singletonData := ref(name + "Input")
	if localized {
		readParams = append(readParams, localeParams()...)
		writeParams = append(writeParams, writeLocaleParam)
		singletonData = obj{"anyOf": []any{ref(name + "Input"), ref(name + "Translation")}}
	}
	return obj{
		"get": obj{
			"tags": tag, "operationId": "get_" + name, "summary": "Get the " + stName + " singleton",
			"parameters": readParams,
			"responses":  merge(obj{"200": ok("Content", dataEnvelope(ref(name), nil))}, errorResponses("401", "403", "404")),
		},
		"put": obj{
			"tags": tag, "operationId": "put_" + name, "summary": "Create or replace the " + stName + " singleton",
			"parameters":  writeParams,
			"requestBody": obj{"required": true, "content": jsonContent(obj{"type": "object", "properties": obj{"data": singletonData}, "required": []string{"data"}})},
			"responses":   merge(obj{"200": ok("Saved", dataEnvelope(ref(name), nil)), "201": ok("Created", dataEnvelope(ref(name), nil))}, errorResponses("400", "401", "403", "404")),
		},
	}
}

func singletonHistoryPaths(stName, name string) obj {
	return historyPaths("/api/singleton/"+stName, []string{"singleton:" + stName}, name, nil, ref(name))
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
		"/api/schema/collections":        schemaPath("List collection schemas", "listCollectionSchemas", true),
		"/api/schema/collections/{name}": schemaPath("A collection schema", "getCollectionSchema", false),
		"/api/schema/singletons":         schemaPath("List singleton schemas", "listSingletonSchemas", true),
		"/api/schema/singletons/{name}":  schemaPath("A singleton schema", "getSingletonSchema", false),
		"/api/schema/components":         schemaPath("List component schemas", "listComponentSchemas", true),
		"/api/schema/components/{name}":  schemaPath("A component schema", "getComponentSchema", false),
		"/auth/me": obj{"get": obj{
			"tags": []string{"auth"}, "operationId": "me", "summary": "The authenticated user or API token",
			"responses": merge(obj{"200": ok("Caller", dataEnvelope(obj{"type": "object", "properties": obj{
				"kind": obj{"type": "string", "enum": []string{"user", "token"}}, "id": obj{"type": "integer"},
				"username": obj{"type": "string"}, "email": obj{"type": "string"}, "name": obj{"type": "string"},
				"role": obj{"type": "string"},
				"permissions": obj{"type": "object", "description": "What the role may do", "properties": obj{
					"content": obj{"type": "object", "description": "Actions by collection name, singleton:<name> or * (other types)",
						"additionalProperties": obj{"type": "array", "items": obj{"type": "string", "enum": []string{"read", "create", "update", "delete", "publish"}}}},
					"media": obj{"type": "array", "items": obj{"type": "string", "enum": []string{"read", "create", "update", "delete"}}},
				}},
			}}, nil))}, errorResponses("401", "404")),
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
		"/api/locales": obj{"get": obj{
			"tags": []string{"locales"}, "operationId": "list_locales", "summary": "Content locales, the default first",
			"responses": merge(obj{"200": ok("Locales", dataEnvelope(obj{"type": "array", "items": ref("Locale")}, nil))}, errorResponses("401")),
		}},
		"/api/revisions": obj{"get": obj{
			"tags": []string{"history"}, "operationId": "list_revisions", "summary": "Changes across all entries, newest first (editors and admins)",
			"parameters": []any{
				obj{"name": "collection", "in": "query", "schema": obj{"type": "string"}},
				obj{"name": "singleton", "in": "query", "schema": obj{"type": "string"}},
				obj{"name": "action", "in": "query", "schema": obj{"type": "string", "enum": revisionActions}},
				obj{"name": "actorKind", "in": "query", "schema": obj{"type": "string", "enum": actorKinds}},
				obj{"name": "actor", "in": "query", "description": "username, token name or agent name", "schema": obj{"type": "string"}},
				obj{"name": "agentRun", "in": "query", "schema": obj{"type": "integer"}},
				obj{"name": "page", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "default": 1}},
				obj{"name": "pageSize", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "maximum": 100, "default": 20}},
			},
			"responses": merge(obj{"200": ok("Revisions", dataEnvelope(obj{"type": "array", "items": ref("Revision")},
				obj{"type": "object", "properties": obj{"pagination": ref("Pagination")}}))}, errorResponses("400", "401", "403")),
		}},
		"/api/proposals": obj{"get": obj{
			"tags": []string{"proposals"}, "operationId": "list_proposals", "summary": "Changes agents proposed, newest first (only collections the caller may change)",
			"parameters": []any{
				obj{"name": "status", "in": "query", "schema": obj{"type": "string", "enum": []string{models.ProposalPending, models.ProposalApproved, models.ProposalRejected, models.ProposalFailed, "all"}, "default": models.ProposalPending}},
				obj{"name": "collection", "in": "query", "schema": obj{"type": "string"}},
				obj{"name": "agent", "in": "query", "schema": obj{"type": "string"}},
				obj{"name": "agentRun", "in": "query", "schema": obj{"type": "integer"}},
				obj{"name": "page", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "default": 1}},
				obj{"name": "pageSize", "in": "query", "schema": obj{"type": "integer", "minimum": 1, "maximum": 100, "default": 20}},
			},
			"responses": merge(obj{"200": ok("Proposals", dataEnvelope(obj{"type": "array", "items": ref("Proposal")},
				obj{"type": "object", "properties": obj{"pagination": ref("Pagination")}}))}, errorResponses("400", "401", "403")),
		}},
		"/api/proposals/{id}": obj{"get": obj{
			"tags": []string{"proposals"}, "operationId": "get_proposal", "summary": "A proposal, the item as it is now and the changes",
			"parameters": []any{paramID},
			"responses": merge(obj{"200": ok("Proposal", dataEnvelope(obj{"type": "object", "properties": obj{
				"proposal": ref("Proposal"), "stale": obj{"type": "boolean", "description": "The item changed since the agent proposed this"},
				"current":       obj{"type": "object", "additionalProperties": true, "description": "The item's values now (pending proposals)"},
				"proposed":      obj{"type": "object", "additionalProperties": true, "description": "What approving writes: for a stale update, the item now with the agent's changes"},
				"currentStatus": obj{"type": "string"},
				"changes":       obj{"type": "array", "items": obj{"type": "object"}},
			}, "required": []string{"proposal", "stale"}}, nil))}, errorResponses("400", "401", "404")),
		}},
		"/api/proposals/{id}/approve": obj{"post": obj{
			"tags": []string{"proposals"}, "operationId": "approve_proposal", "summary": "Apply a proposal, optionally with edited values",
			"parameters": []any{paramID},
			"requestBody": obj{"content": jsonContent(obj{"type": "object", "properties": obj{
				"data":  obj{"type": "object", "additionalProperties": true, "description": "Values to apply instead of the agent's"},
				"force": obj{"type": "boolean", "description": "Apply although the item changed since"},
			}})},
			"responses": merge(obj{"200": ok("Applied", dataEnvelope(obj{"type": "object", "properties": obj{
				"proposal": ref("Proposal"), "item": obj{"type": "object", "additionalProperties": true},
			}}, nil))}, errorResponses("400", "401", "403", "404", "409", "422")),
		}},
		"/api/proposals/{id}/reject": obj{"post": obj{
			"tags": []string{"proposals"}, "operationId": "reject_proposal", "summary": "Discard a proposal",
			"parameters":  []any{paramID},
			"requestBody": obj{"content": jsonContent(obj{"type": "object", "properties": obj{"note": obj{"type": "string"}}})},
			"responses":   merge(obj{"200": ok("Rejected", dataEnvelope(obj{"type": "object", "properties": obj{"proposal": ref("Proposal")}}, nil))}, errorResponses("401", "404", "409")),
		}},
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
		"Locale": obj{"type": "object", "properties": obj{
			"id": obj{"type": "integer"}, "code": obj{"type": "string", "description": "BCP 47 code, e.g. fr or pt-BR"},
			"name": obj{"type": "string"}, "isDefault": obj{"type": "boolean"}, "createdAt": obj{"type": "string", "format": "date-time"},
		}, "required": []string{"code", "name", "isDefault"}},
		"Revision":       revisionSchema(false),
		"RevisionDetail": revisionSchema(true),
		"Proposal":       proposalSchema(),
		"Media": obj{"type": "object", "properties": obj{
			"id": obj{"type": "integer"}, "url": obj{"type": "string"}, "name": obj{"type": "string"}, "mime": obj{"type": "string"},
			"size": obj{"type": "integer"}, "width": nullable(obj{"type": "integer"}), "height": nullable(obj{"type": "integer"}),
			"hash": obj{"type": "string"}, "alt": obj{"type": "string"}, "caption": obj{"type": "string"}, "uploadedBy": obj{"type": "string"},
			"createdAt": obj{"type": "string", "format": "date-time"}, "updatedAt": obj{"type": "string", "format": "date-time"},
		}, "required": []string{"id", "url", "name", "mime", "size"}},
	}
}

// schemaPath describes a read-only content model endpoint. The schemas are
// free-form objects (the admin format), not part of the generated types.
func schemaPath(summary, operationID string, list bool) obj {
	var body obj = obj{"type": "object"}
	if list {
		body = obj{"type": "array", "items": obj{"type": "object"}}
	}
	op := obj{
		"tags": []string{"schema"}, "operationId": operationID, "summary": summary,
		"responses": merge(obj{"200": ok("Schema", dataEnvelope(body, nil))}, errorResponses("401", "404")),
	}
	if !list {
		op["parameters"] = []any{obj{"name": "name", "in": "path", "required": true, "schema": obj{"type": "string"}}}
	}
	return obj{"get": op}
}

var (
	revisionActions = []string{models.RevisionCreate, models.RevisionUpdate, models.RevisionPublish, models.RevisionUnpublish, models.RevisionDelete, models.RevisionRestore, models.RevisionBaseline}
	actorKinds      = []string{models.ActorUser, models.ActorToken, models.ActorAgent, models.ActorSystem}
)

// revisionSchema is a revision; detail adds its data and the diff.
func revisionSchema(detail bool) obj {
	props := obj{
		"id": obj{"type": "integer"}, "entityType": obj{"type": "string", "enum": []string{models.RevisionEntityItem, models.RevisionEntitySingleton}},
		"entityId": obj{"type": "integer"}, "version": obj{"type": "integer"}, "typeName": obj{"type": "string"},
		"action":    obj{"type": "string", "enum": revisionActions},
		"status":    obj{"type": "string", "enum": []string{models.StatusDraft, models.StatusPublished}},
		"actorKind": obj{"type": "string", "enum": actorKinds}, "actorName": obj{"type": "string"},
		"agentRunId": obj{"type": "integer"}, "restoredFrom": obj{"type": "integer"},
		"createdAt": obj{"type": "string", "format": "date-time"},
	}
	required := []string{"id", "entityType", "entityId", "version", "typeName", "action", "actorKind", "actorName", "createdAt"}
	if detail {
		props["data"] = obj{"type": "object", "additionalProperties": true, "description": "The stored data after the change (relations as IDs)"}
		props["translations"] = obj{"type": "object", "additionalProperties": obj{"type": "object"}, "description": "Localized values in other locales than the default, by locale"}
		props["comparedTo"] = nullable(obj{"type": "integer"})
		props["changes"] = obj{"type": "array", "items": obj{"type": "object", "properties": obj{
			"field":  obj{"type": "string"},
			"change": obj{"type": "string", "enum": []string{"added", "removed", "changed"}},
			"before": obj{}, "after": obj{},
		}, "required": []string{"field", "change"}}}
		required = append(required, "data", "changes")
	}
	return obj{"type": "object", "properties": props, "required": required}
}

// proposalSchema is a change an agent proposed.
func proposalSchema() obj {
	return obj{"type": "object", "properties": obj{
		"id": obj{"type": "integer"}, "agentName": obj{"type": "string"}, "agentRunId": nullable(obj{"type": "integer"}),
		"action": obj{"type": "string", "enum": []string{models.ProposalCreate, models.ProposalUpdate, models.ProposalDelete, models.ProposalCall},
			"description": "call: a call to a connected app's tool (admins only)"},
		"collection": obj{"type": "string", "description": "The collection, or app:<connection> for a call"}, "itemId": nullable(obj{"type": "integer"}),
		"locale": obj{"type": "string", "description": "Set when the change is a translation"},
		"data":   nullable(obj{"type": "object", "additionalProperties": true, "description": "The item's values after the change (null for a deletion); for a call, the tool and its arguments"}),
		"reason": obj{"type": "string"}, "baseVersion": obj{"type": "integer"},
		"status":     obj{"type": "string", "enum": []string{models.ProposalPending, "applying", models.ProposalApproved, models.ProposalRejected, models.ProposalFailed}},
		"reviewedBy": obj{"type": "string"}, "reviewNote": obj{"type": "string"}, "error": obj{"type": "string"},
		"stale":     obj{"type": "boolean", "description": "In lists of pending proposals: the item changed since"},
		"createdAt": obj{"type": "string", "format": "date-time"}, "decidedAt": nullable(obj{"type": "string", "format": "date-time"}),
	}, "required": []string{"id", "agentName", "action", "collection", "status", "createdAt"}}
}
