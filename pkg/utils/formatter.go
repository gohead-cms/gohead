package utils

import (
	"math"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	agents "github.com/gohead-cms/gohead/internal/models/agents"

	"github.com/gertd/go-pluralize"
)

var pluralizeClient = pluralize.NewClient()

func FormatCollectionSchema(collection *models.Collection) map[string]any {
	singular := pluralizeClient.Singular(collection.Name)
	plural := pluralizeClient.Plural(collection.Name)

	attrSchema := map[string]any{}
	for _, attr := range collection.Attributes {
		attrDef := map[string]any{
			"type":     attr.Type,
			"required": attr.Required,
		}
		// Every setting the input accepts is returned, so a schema read,
		// edited and sent back (PUT replaces it) keeps them.
		if attr.Unique {
			attrDef["unique"] = true
		}
		if attr.Localized {
			attrDef["localized"] = true
		}
		if len(attr.Options) > 0 {
			attrDef["options"] = attr.Options
		}
		if attr.Min != nil {
			attrDef["min"] = *attr.Min
		}
		if attr.Max != nil {
			attrDef["max"] = *attr.Max
		}
		if attr.Pattern != "" {
			attrDef["pattern"] = attr.Pattern
		}
		if attr.Type == "relation" {
			attrDef["target"] = "api::" + attr.Target + "." + attr.Target
			attrDef["relation"] = attr.Relation
		}
		if attr.Type == "component" {
			attrDef["component"] = attr.ComponentRef
			attrDef["repeatable"] = attr.Repeatable
		}
		if attr.Type == "media" {
			attrDef["multiple"] = attr.Multiple
			if len(attr.AllowedTypes) > 0 {
				attrDef["allowed_types"] = attr.AllowedTypes
			}
		}
		attrSchema[attr.Name] = attrDef
	}

	return map[string]any{
		"id":  collection.ID,
		"uid": "api::" + collection.Name + "." + collection.Name,
		"schema": map[string]any{
			"collectionName": collection.Name,
			"description":    collection.Description,
			"info": map[string]any{
				"singularName": singular,
				"pluralName":   plural,
				"displayName":  capitalize(collection.Name),
			},
			"options":    map[string]any{"draftAndPublish": collection.DraftAndPublish},
			"attributes": attrSchema,
		},
	}
}

// FormatCollectionsSchema: returns an array of collection schemas for admin endpoints.
func FormatCollectionsSchema(collections []models.Collection) []map[string]any {
	formatted := make([]map[string]any, 0, len(collections))
	for _, col := range collections {
		formatted = append(formatted, FormatCollectionSchema(&col))
	}
	return formatted
}

// FormatCollectionData returns an array of collections as DATA (not schema), Strapi-style for /collections API.
func FormatCollectionsData(collections []models.Collection) []map[string]any {
	formatted := make([]map[string]any, 0, len(collections))
	for _, col := range collections {
		formatted = append(formatted, map[string]any{
			"id": col.ID,
			"attributes": map[string]any{
				"name":        col.Name,
				"kind":        col.Kind,
				"description": col.Description,
				// You may add more, but do NOT nest attributes definition here!
			},
		})
	}
	return formatted
}

// FormatCollectionItem formats a single item for a Strapi-like /api/[collection] response.
func FormatCollectionItem(item *models.Item, collection *models.Collection) map[string]any {
	if item == nil || collection == nil {
		return nil
	}

	attributes := map[string]any{}
	for _, attr := range collection.Attributes {
		val, exists := item.Data[attr.Name]
		switch attr.Type {
		case "relation":
			switch attr.Relation {
			case "oneToOne":
				var relData any
				if exists && val != nil {
					relData = map[string]any{"id": toInt(val)}
				}
				attributes[attr.Name] = map[string]any{"data": relData}
			case "manyToMany":
				relArr := []map[string]any{}
				if exists && val != nil {
					if ids, ok := val.([]any); ok {
						for _, id := range ids {
							relArr = append(relArr, map[string]any{"id": toInt(id)})
						}
					}
				}
				attributes[attr.Name] = map[string]any{"data": relArr}
			}
		default:
			if exists {
				attributes[attr.Name] = val
			}
		}
	}

	return map[string]any{
		"id":         item.ID,
		"attributes": attributes,
	}
}

// FormatCollectionItems returns multiple items formatted for Strapi
func FormatCollectionItems(items []models.Item, collection *models.Collection) []map[string]any {
	formatted := make([]map[string]any, 0, len(items))
	for i := range items {
		formatted = append(formatted, FormatCollectionItem(&items[i], collection))
	}
	return formatted
}

// FormatNestedItemToStrapi wraps your nested JSONMap
func FormatNestedItems(itemID uint, data models.JSONMap, collection *models.Collection) map[string]any {
	attributes := map[string]any{}

	for _, attr := range collection.Attributes {
		val, exists := data[attr.Name]
		switch attr.Type {
		case "relation":
			switch attr.Relation {
			case "oneToOne":
				if !exists || val == nil {
					attributes[attr.Name] = nil
				} else if nested, ok := val.(map[string]any); ok {
					// Remove "id" if present from attributes
					attrCopy := map[string]any{}
					for k, v := range nested {
						if k != "id" {
							attrCopy[k] = v
						}
					}
					attributes[attr.Name] = attrCopy
				} else {
					attributes[attr.Name] = val // fallback: put value directly
				}
			case "manyToMany":
				arr := []any{}
				if exists && val != nil {
					switch list := val.(type) {
					case []any:
						for _, elem := range list {
							arr = append(arr, elem)
						}
					case []models.JSONMap:
						for _, nested := range list {
							arr = append(arr, nested)
						}
					}
				}
				attributes[attr.Name] = arr

			}
		default:
			if exists {
				attributes[attr.Name] = val
			}
		}
	}

	return map[string]any{
		"id":         itemID,
		"attributes": attributes,
	}
}

func toUintIfPresent(val any) (uint, bool) {
	switch v := val.(type) {
	case int:
		return uint(v), true
	case int64:
		return uint(v), true
	case float64:
		return uint(v), true
	case uint:
		return v, true
	case uint64:
		return uint(v), true
	}
	return 0, false
}

// --- Helpers ---

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func toInt(val interface{}) int {
	// Relation IDs may be stored as any numeric type (uint after a nested
	// create, float64 after a JSON round-trip, ...).
	u := models.ToUint(val)
	if u > uint(math.MaxInt) {
		return 0
	}
	return int(u)
}

// SecretMask replaces stored secrets in API responses. Sending it back
// unchanged on update keeps the stored value.
const SecretMask = "********"

func maskSecret(v string) string {
	if v == "" {
		return ""
	}
	return SecretMask
}

// FormatAgentSchema formats a single agent into a schema map for API
// endpoints. Credentials (LLM API key/secret, webhook token) are masked.
func FormatAgentSchema(agent *agents.Agent) map[string]any {
	if agent == nil {
		return nil
	}
	llmCfg := agent.LLMConfig
	llmCfg.APIKey = maskSecret(llmCfg.APIKey)
	llmCfg.APISecret = maskSecret(llmCfg.APISecret)
	trigger := agent.Trigger
	trigger.WebhookToken = maskSecret(trigger.WebhookToken)

	return map[string]any{
		"id":   agent.ID,
		"name": agent.Name,
		"schema": map[string]any{
			"uid":          "api::" + agent.Name + "." + agent.Name,
			"name":         agent.Name,
			"systemPrompt": agent.SystemPrompt,
			"maxTurns":     agent.MaxTurns,
			"llmConfig":    llmCfg,
			"memory":       agent.Memory,
			"trigger":      trigger,
			"functions":    agent.Functions,
			"limits":       agent.Limits,
		},
	}
}

// FormatAgentsSchema returns an array of agent schemas for admin endpoints.
func FormatAgentsSchema(agents []agents.Agent) []map[string]any {
	formatted := make([]map[string]any, 0, len(agents))
	for _, agent := range agents {
		formatted = append(formatted, FormatAgentSchema(&agent))
	}
	return formatted
}

// FormatAgentData returns an array of agents as DATA (not schema).
func FormatAgentData(agents []agents.Agent) []map[string]any {
	formatted := make([]map[string]any, 0, len(agents))
	for _, agent := range agents {
		formatted = append(formatted, map[string]any{
			"id": agent.ID,
			"attributes": map[string]any{
				"name":         agent.Name,
				"systemPrompt": agent.SystemPrompt,
				"maxTurns":     agent.MaxTurns,
			},
		})
	}
	return formatted
}
