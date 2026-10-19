package functions

import (
	"sort"

	"github.com/gohead-cms/gohead/internal/models"
)

// Info describes a built-in function for people configuring agents: what
// it does and the JSON schema of its arguments, ready to use as a function's
// "parameters".
type Info struct {
	ImplKey     string         `json:"impl_key"`
	Description string         `json:"description"`
	Parameters  models.JSONMap `json:"parameters"`
	// Scoped: the function accepts a "collections" scope.
	Scoped bool `json:"scoped"`
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

var catalog = map[string]Info{
	"collections.list": {Description: "List the collections (only those in scope)", Parameters: object(map[string]any{})},
	"collections.get_schema": {Description: "Attributes and types of a collection",
		Parameters: object(map[string]any{"collection_name": str("Collection name")}, "collection_name")},
	"collections.list_items": {Description: "A page of items of a collection (drafts included)",
		Parameters: object(map[string]any{"collection_name": str("Collection name"), "page": num("Page, from 1"), "pageSize": num("Items per page")}, "collection_name")},
	"collections.upsert_item": {Description: "Create an item, or replace item item_id, validated like API writes",
		Parameters: object(map[string]any{
			"collection_name": str("Collection name"),
			"data":            map[string]any{"type": "object", "description": "The item's attributes"},
			"item_id":         str("ID of the item to replace (omit to create)"),
			"locale":          str("With item_id: write the item's localized attributes in this locale (a translation) instead of its main content; values not given keep their current translation, null removes one"),
		}, "collection_name", "data")},
	"collections.list_locales": {Description: "List the content locales (the default first)", Parameters: object(map[string]any{})},
	"collections.delete_item": {Description: "Delete an item by ID",
		Parameters: object(map[string]any{"item_id": str("Item ID"), "collection_name": str("Collection the item must belong to")}, "item_id")},
	"system.log": {Description: "Write a message to the worker log",
		Parameters: object(map[string]any{"message": str("Message"), "data": map[string]any{"type": "object"}}, "message")},
	"llm.analyze_sentiment": {Description: "Sentiment of a text", Parameters: object(map[string]any{"text": textArg}, "text")},
	"llm.classify_text": {Description: "Pick categories for a text",
		Parameters: object(map[string]any{"text": textArg, "categories": list("Allowed categories")}, "text", "categories")},
	"llm.moderate_content": {Description: "Flag abusive, spam or unsafe content", Parameters: object(map[string]any{"text": textArg}, "text")},
	"llm.score_quality": {Description: "Score the quality of a text",
		Parameters: object(map[string]any{"text": textArg, "criteria": list("What to judge (optional)")}, "text")},
	"llm.summarize_text": {Description: "Summarize a text",
		Parameters: object(map[string]any{"text": textArg, "max_length": num("Maximum length in words"), "style": str("e.g. paragraph, bullet_points")}, "text")},
	"llm.rewrite_text": {Description: "Rewrite a text in another tone or style",
		Parameters: object(map[string]any{"text": textArg, "tone": str("e.g. formal, casual"), "style": str("e.g. concise"), "audience": str("Target audience")}, "text")},
	"llm.generate_tags": {Description: "Suggest tags for a text",
		Parameters: object(map[string]any{"text": textArg, "max_tags": num("Maximum number of tags")}, "text")},
	"llm.generate_title": {Description: "Suggest a title for a text",
		Parameters: object(map[string]any{"text": textArg, "max_length": num("Maximum length in characters"), "style": str("e.g. news, academic")}, "text")},
	"llm.evaluate_condition": {Description: "Check whether a text meets a condition",
		Parameters: object(map[string]any{"text": textArg, "condition": str("The condition, in plain language")}, "text", "condition")},
	"llm.rank_items": {Description: "Rank items against criteria",
		Parameters: object(map[string]any{"items": list("Items to rank"), "criteria": str("How to rank them")}, "items", "criteria")},
	"llm.recommend_action": {Description: "Recommend one of several actions",
		Parameters: object(map[string]any{"context": str("The situation"), "actions": list("Possible actions"), "goal": str("Goal (optional)")}, "context", "actions")},
	"llm.extract_entities": {Description: "Extract people, places, dates…",
		Parameters: object(map[string]any{"text": textArg, "entity_types": list("Entity types to extract (optional)")}, "text")},
	"llm.extract_structured_data": {Description: "Extract data matching a JSON schema",
		Parameters: object(map[string]any{"text": textArg, "schema": map[string]any{"type": "object", "description": "Expected structure"}}, "text", "schema")},
}

// Catalog lists the built-in functions that can be used as impl_key, sorted.
// Functions registered without catalog entry are listed with an empty
// argument schema.
func Catalog() []Info {
	out := make([]Info, 0, len(StaticFunctionMap))
	for key := range StaticFunctionMap {
		info, ok := catalog[key]
		if !ok {
			info = Info{Parameters: object(map[string]any{})}
		}
		info.ImplKey = key
		info.Scoped = len(key) > len("collections.") && key[:len("collections.")] == "collections."
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ImplKey < out[j].ImplKey })
	return out
}
