// Package tools describes the built-in tools agents can use: their names,
// what they do and the JSON schema of their arguments. The implementations
// live in internal/agent/functions; this package only holds the catalog, so
// the agent model, the API and the runner can all read it.
//
// A tool has two names. Its implementation key (impl_key, such as
// "collections.patch_item") is what agents store and run. Its "use" name
// (such as "content.update") is the short, readable one people write. Both
// are accepted wherever a tool is named.
package tools

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
)

// Group of a tool, for people choosing one.
const (
	GroupContent = "content" // read and write items
	GroupAI      = "ai"      // ask a model about a text
	GroupSystem  = "system"
)

// Info describes a built-in tool.
type Info struct {
	ImplKey string `json:"impl_key"`
	// Use is the short name people write ("content.update").
	Use string `json:"use"`
	// Name is the name the model sees when the agent doesn't give one.
	Name        string         `json:"name"`
	Label       string         `json:"label"`
	Group       string         `json:"group"`
	Description string         `json:"description"`
	Parameters  models.JSONMap `json:"parameters"`
	// Scoped: the tool works on collections and accepts an "on" scope.
	Scoped bool `json:"scoped"`
	// Writes: the tool changes content (create, update, delete).
	Writes bool `json:"writes"`
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func num(desc string) map[string]any { return map[string]any{"type": "integer", "description": desc} }
func list(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}
func object(props map[string]any, required ...string) models.JSONMap {
	s := models.JSONMap{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

var textArg = str("The text to work on")

const reasonHelp = "Why: shown to the editor who approves the change (agents that need approval)"

var catalog = []Info{
	// ---- Content
	{ImplKey: "collections.list", Use: "content.types", Name: "list_collections", Group: GroupContent,
		Label: "List content types", Description: "List the collections (only those in scope)",
		Parameters: object(map[string]any{})},
	{ImplKey: "collections.get_schema", Use: "content.schema", Name: "get_schema", Group: GroupContent,
		Label: "Read a content type", Description: "Attributes and types of a collection",
		Parameters: object(map[string]any{"collection_name": str("Collection name")}, "collection_name")},
	{ImplKey: "collections.list_items", Use: "content.list", Name: "list_items", Group: GroupContent,
		Label: "List items", Description: "A page of items of a collection (drafts included)",
		Parameters: object(map[string]any{"collection_name": str("Collection name"), "page": num("Page, from 1"), "pageSize": num("Items per page")}, "collection_name")},
	{ImplKey: "collections.upsert_item", Use: "content.save", Name: "save_item", Group: GroupContent, Writes: true,
		Label: "Create or replace an item", Description: "Create an item, or replace item item_id, validated like API writes",
		Parameters: object(map[string]any{
			"collection_name": str("Collection name"),
			"data":            map[string]any{"type": "object", "description": "The item's attributes"},
			"item_id":         str("ID of the item to replace (omit to create)"),
			"locale":          str("With item_id: write the item's localized attributes in this locale (a translation) instead of its main content; values not given keep their current translation, null removes one"),
			"reason":          str(reasonHelp),
		}, "collection_name", "data")},
	{ImplKey: "collections.patch_item", Use: "content.update", Name: "update_item", Group: GroupContent, Writes: true,
		Label: "Update fields of an item", Description: "Change some attributes of an item and keep the others, validated like API writes",
		Parameters: object(map[string]any{
			"collection_name": str("Collection name"),
			"item_id":         str("ID of the item"),
			"data":            map[string]any{"type": "object", "description": "The attributes to change"},
			"locale":          str("Change the values of this locale (localized attributes only)"),
			"reason":          str(reasonHelp),
		}, "collection_name", "item_id", "data")},
	{ImplKey: "collections.delete_item", Use: "content.delete", Name: "delete_item", Group: GroupContent, Writes: true,
		Label: "Delete an item", Description: "Delete an item by ID",
		Parameters: object(map[string]any{"item_id": str("Item ID"), "collection_name": str("Collection the item must belong to"),
			"reason": str(reasonHelp)}, "item_id")},
	{ImplKey: "collections.list_locales", Use: "content.locales", Name: "list_locales", Group: GroupContent,
		Label: "List locales", Description: "List the content locales (the default first)",
		Parameters: object(map[string]any{})},

	// ---- AI helpers: they call the instance's default model themselves.
	{ImplKey: "llm.summarize_text", Use: "ai.summarize", Name: "summarize_text", Group: GroupAI,
		Label: "Summarize a text", Description: "Summarize a text",
		Parameters: object(map[string]any{"text": textArg, "max_length": num("Maximum length in words"), "style": str("e.g. paragraph, bullet_points")}, "text")},
	{ImplKey: "llm.generate_tags", Use: "ai.tags", Name: "generate_tags", Group: GroupAI,
		Label: "Suggest tags", Description: "Suggest tags for a text",
		Parameters: object(map[string]any{"text": textArg, "max_tags": num("Maximum number of tags")}, "text")},
	{ImplKey: "llm.generate_title", Use: "ai.title", Name: "generate_title", Group: GroupAI,
		Label: "Suggest a title", Description: "Suggest a title for a text",
		Parameters: object(map[string]any{"text": textArg, "max_length": num("Maximum length in characters"), "style": str("e.g. news, academic")}, "text")},
	{ImplKey: "llm.rewrite_text", Use: "ai.rewrite", Name: "rewrite_text", Group: GroupAI,
		Label: "Rewrite a text", Description: "Rewrite a text in another tone or style",
		Parameters: object(map[string]any{"text": textArg, "tone": str("e.g. formal, casual"), "style": str("e.g. concise"), "audience": str("Target audience")}, "text")},
	{ImplKey: "llm.classify_text", Use: "ai.classify", Name: "classify_text", Group: GroupAI,
		Label: "Classify a text", Description: "Pick categories for a text",
		Parameters: object(map[string]any{"text": textArg, "categories": list("Allowed categories")}, "text", "categories")},
	{ImplKey: "llm.moderate_content", Use: "ai.moderate", Name: "moderate_content", Group: GroupAI,
		Label: "Moderate a text", Description: "Flag abusive, spam or unsafe content",
		Parameters: object(map[string]any{"text": textArg}, "text")},
	{ImplKey: "llm.analyze_sentiment", Use: "ai.sentiment", Name: "analyze_sentiment", Group: GroupAI,
		Label: "Analyze sentiment", Description: "Sentiment of a text",
		Parameters: object(map[string]any{"text": textArg}, "text")},
	{ImplKey: "llm.score_quality", Use: "ai.score", Name: "score_quality", Group: GroupAI,
		Label: "Score quality", Description: "Score the quality of a text",
		Parameters: object(map[string]any{"text": textArg, "criteria": list("What to judge (optional)")}, "text")},
	{ImplKey: "llm.evaluate_condition", Use: "ai.check", Name: "evaluate_condition", Group: GroupAI,
		Label: "Check a condition", Description: "Check whether a text meets a condition",
		Parameters: object(map[string]any{"text": textArg, "condition": str("The condition, in plain language")}, "text", "condition")},
	{ImplKey: "llm.extract_entities", Use: "ai.entities", Name: "extract_entities", Group: GroupAI,
		Label: "Extract entities", Description: "Extract people, places, dates…",
		Parameters: object(map[string]any{"text": textArg, "entity_types": list("Entity types to extract (optional)")}, "text")},
	{ImplKey: "llm.extract_structured_data", Use: "ai.extract", Name: "extract_structured_data", Group: GroupAI,
		Label: "Extract structured data", Description: "Extract data matching a JSON schema",
		Parameters: object(map[string]any{"text": textArg, "schema": map[string]any{"type": "object", "description": "Expected structure"}}, "text", "schema")},
	{ImplKey: "llm.rank_items", Use: "ai.rank", Name: "rank_items", Group: GroupAI,
		Label: "Rank items", Description: "Rank items against criteria",
		Parameters: object(map[string]any{"items": list("Items to rank"), "criteria": str("How to rank them")}, "items", "criteria")},
	{ImplKey: "llm.recommend_action", Use: "ai.recommend", Name: "recommend_action", Group: GroupAI,
		Label: "Recommend an action", Description: "Recommend one of several actions",
		Parameters: object(map[string]any{"context": str("The situation"), "actions": list("Possible actions"), "goal": str("Goal (optional)")}, "context", "actions")},

	// ---- System
	{ImplKey: "system.log", Use: "system.log", Name: "log", Group: GroupSystem,
		Label: "Write to the log", Description: "Write a message to the worker log",
		Parameters: object(map[string]any{"message": str("Message"), "data": map[string]any{"type": "object"}}, "message")},
}

var byImplKey, byUse = func() (map[string]Info, map[string]Info) {
	impl, use := map[string]Info{}, map[string]Info{}
	for _, t := range catalog {
		t.Scoped = strings.HasPrefix(t.ImplKey, "collections.")
		impl[t.ImplKey], use[t.Use] = t, t
	}
	return impl, use
}()

// Lookup returns the tool a name refers to: a use name ("content.update")
// or an implementation key ("collections.patch_item").
func Lookup(name string) (Info, bool) {
	if t, ok := byUse[name]; ok {
		return t, true
	}
	t, ok := byImplKey[name]
	return t, ok
}

// ImplKey returns the implementation key a name refers to; names the
// catalog doesn't know (functions registered by a fork) are returned as
// they are.
func ImplKey(name string) string {
	if t, ok := Lookup(name); ok {
		return t.ImplKey
	}
	return name
}

// Use returns the short name of an implementation key, or the key itself
// for functions the catalog doesn't know.
func Use(implKey string) string {
	if t, ok := byImplKey[implKey]; ok {
		return t.Use
	}
	return implKey
}

// Scoped reports whether an implementation works on collections.
func Scoped(implKey string) bool { return strings.HasPrefix(implKey, "collections.") }

// All returns the catalog in display order: content, AI helpers, system.
func All() []Info {
	out := make([]Info, 0, len(catalog))
	for _, t := range catalog {
		out = append(out, byImplKey[t.ImplKey])
	}
	return out
}

// Known returns the given implementation keys the catalog lists, sorted;
// with extra keys of functions registered without a catalog entry.
func Known(extra []string) []Info {
	out := All()
	for _, key := range extra {
		if _, ok := byImplKey[key]; !ok {
			out = append(out, Info{ImplKey: key, Use: key, Name: strings.ReplaceAll(key, ".", "_"),
				Label: key, Group: GroupSystem, Parameters: object(map[string]any{}), Scoped: Scoped(key)})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return groupRank(out[i].Group) < groupRank(out[j].Group) })
	return out
}

func groupRank(g string) int {
	switch g {
	case GroupContent:
		return 0
	case GroupAI:
		return 1
	}
	return 2
}

// EmptySchema reports whether a parameters schema says nothing: absent, or
// an object without properties. Agents with such a schema get the tool's
// own one.
func EmptySchema(s models.JSONMap) bool {
	if len(s) == 0 {
		return true
	}
	props, ok := s["properties"]
	if !ok {
		return len(s) == 1 && s["type"] == "object"
	}
	switch p := props.(type) {
	case map[string]any:
		return len(p) == 0 && len(s) <= 2
	case models.JSONMap:
		return len(p) == 0 && len(s) <= 2
	}
	return false
}

// SameSchema reports whether two schemas are the same JSON.
func SameSchema(a, b models.JSONMap) bool {
	ja, err1 := json.Marshal(a)
	jb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return reflect.DeepEqual(a, b)
	}
	return string(ja) == string(jb)
}
