package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"
)

var StaticFunctionMap = make(map[string]ToolFunc)

func init() {
	// Register all your collection-related functions here
	StaticFunctionMap["collections.list"] = listCollections
	StaticFunctionMap["collections.get_schema"] = getCollectionSchema
	StaticFunctionMap["collections.upsert_item"] = upsertCollectionItem
	StaticFunctionMap["collections.list_items"] = listCollectionItems // Renamed from query_items
	StaticFunctionMap["collections.delete_item"] = deleteCollectionItem
	StaticFunctionMap["collections.list_locales"] = listLocales
	StaticFunctionMap["collections.patch_item"] = patchCollectionItem
}

// listLocales lists the content locales.
func listLocales(ctx context.Context, args any) (string, error) {
	locales, err := storage.ListLocales()
	if err != nil {
		return `{"status": "error", "message": "failed to list locales"}`, nil
	}
	out := make([]map[string]any, 0, len(locales))
	for _, l := range locales {
		out = append(out, map[string]any{"code": l.Code, "name": l.Name, "default": l.IsDefault})
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// listCollections lists all available collection names.
func listCollections(ctx context.Context, args any) (string, error) {
	collections, _, err := storage.GetAllCollections(nil, nil, nil)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "failed to retrieve collections: %s"}`, err.Error()), nil
	}

	collectionNames := make([]string, 0, len(collections))
	for _, c := range collections {
		if collectionAllowed(ctx, c.Name) {
			collectionNames = append(collectionNames, c.Name)
		}
	}

	resultBytes, _ := json.Marshal(collectionNames)
	return string(resultBytes), nil
}

// getCollectionSchema retrieves the structure of a specific collection.
func getCollectionSchema(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	collectionName, _ := argMap["collection_name"].(string)
	if collectionName == "" {
		return `{"status": "error", "message": "missing required parameter: collection_name"}`, nil
	}
	if !collectionAllowed(ctx, collectionName) {
		return scopeError(collectionName), nil
	}

	collection, err := storage.GetCollectionByName(collectionName)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	formattedSchema := utils.FormatCollectionSchema(collection)
	resultBytes, _ := json.Marshal(formattedSchema)
	return string(resultBytes), nil
}

// upsertCollectionItem creates a new item or updates an existing one.
// NOTE: It treats `item_id` as the primary key. If `item_id` is "0" or empty, it creates a new record.
func upsertCollectionItem(ctx context.Context, args any) (string, error) {
	// Arguments arrive as a JSON string from the runner or as a map.
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid JSON format in arguments"}`, nil
	}

	// 3. Success! You can now use argMap as a normal map.
	collectionName, ok := argMap["collection_name"].(string)
	if !ok {
		return `{"status": "error", "message": "missing or invalid 'collection_name'"}`, nil
	}

	itemIDStr, _ := argMap["item_id"].(string)
	data, _ := argMap["data"].(map[string]any)

	if collectionName == "" || data == nil {
		return `{"status": "error", "message": "missing required parameters: collection_name and data"}`, nil
	}
	if !collectionAllowed(ctx, collectionName) {
		return scopeError(collectionName), nil
	}

	// 1. Get Collection ID from Name
	collection, err := storage.GetCollectionByName(collectionName)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	// Nested relation objects can look up or create items in other
	// collections: those must be in scope too.
	if name, err := nestedScopeViolation(ctx, collection, data, 0); err != nil {
		msg, _ := json.Marshal(err.Error())
		return fmt.Sprintf(`{"status": "error", "message": %s}`, msg), nil
	} else if name != "" {
		return scopeError(name), nil
	}

	itemID, _ := strconv.ParseUint(itemIDStr, 10, 32)
	locale, err := translationLocale(argMap)
	if err != nil {
		return errorResult(err), nil
	}
	reason, _ := argMap["reason"].(string)
	if itemID == 0 {
		if locale != "" {
			def, _ := storage.DefaultLocale()
			return errorResult(fmt.Errorf("create items in the default locale (%s), then write translations with item_id and locale", def)), nil
		}
		return writeItem(ctx, collection, models.ProposalCreate, 0, "", data, reason), nil
	}
	if locale != "" {
		// Merge over the locale's current values, so translating one
		// attribute keeps the others.
		data = mergeOver(currentValues(collection, uint(itemID), locale), data)
	}
	return writeItem(ctx, collection, models.ProposalUpdate, uint(itemID), locale, data, reason), nil
}

// patchCollectionItem changes some attributes of an item, keeping the
// others (upsert_item with item_id replaces the whole item).
func patchCollectionItem(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid JSON format in arguments"}`, nil
	}
	collectionName, _ := argMap["collection_name"].(string)
	data, _ := argMap["data"].(map[string]any)
	var itemID uint64
	switch v := argMap["item_id"].(type) {
	case float64:
		if v > 0 && v == float64(uint32(v)) {
			itemID = uint64(v)
		}
	case string:
		itemID, _ = strconv.ParseUint(v, 10, 32)
	}
	if collectionName == "" || itemID == 0 || data == nil {
		return `{"status": "error", "message": "missing required parameters: collection_name, item_id and data"}`, nil
	}
	if !collectionAllowed(ctx, collectionName) {
		return scopeError(collectionName), nil
	}
	collection, err := storage.GetCollectionByName(collectionName)
	if err != nil {
		return errorResult(err), nil
	}
	if name, err := nestedScopeViolation(ctx, collection, data, 0); err != nil {
		return errorResult(err), nil
	} else if name != "" {
		return scopeError(name), nil
	}
	locale, err := translationLocale(argMap)
	if err != nil {
		return errorResult(err), nil
	}
	if _, err := storage.GetItemByID(collection.ID, uint(itemID)); err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "item with ID %d not found in collection '%s'"}`, itemID, collectionName), nil
	}
	reason, _ := argMap["reason"].(string)
	merged := mergeOver(currentValues(collection, uint(itemID), locale), data)
	return writeItem(ctx, collection, models.ProposalUpdate, uint(itemID), locale, merged, reason), nil
}

// translationLocale returns the "locale" argument when it names another
// locale than the default ("" otherwise).
func translationLocale(argMap map[string]any) (string, error) {
	locale, _ := argMap["locale"].(string)
	if locale == "" {
		return "", nil
	}
	code, def, err := storage.ResolveLocale(locale)
	if err != nil {
		return "", fmt.Errorf("unknown locale '%s' (see collections.list_locales)", locale)
	}
	if code == def {
		return "", nil
	}
	return code, nil
}

// currentValues returns an item's data, or its values in a locale (only
// attributes that are localized).
func currentValues(ct *models.Collection, itemID uint, locale string) map[string]any {
	existing, err := storage.GetItemByID(ct.ID, itemID)
	if err != nil {
		return map[string]any{}
	}
	if locale == "" {
		return existing.Data
	}
	out := map[string]any{}
	localized := models.LocalizedNames(ct.Attributes)
	for k, v := range models.Translation(existing.Translations, locale) {
		if localized[k] {
			out[k] = v
		}
	}
	return out
}

func mergeOver(base, changes map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range changes {
		out[k] = v
	}
	return out
}

// listCollectionItems finds and retrieves items from a collection with pagination.
func listCollectionItems(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	collectionName, _ := argMap["collection_name"].(string)
	if collectionName == "" {
		return `{"status": "error", "message": "missing required parameter: collection_name"}`, nil
	}
	if !collectionAllowed(ctx, collectionName) {
		return scopeError(collectionName), nil
	}

	// Handle optional pagination parameters
	page, _ := argMap["page"].(float64) // JSON numbers are often float64
	pageSize, _ := argMap["pageSize"].(float64)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// 1. Get Collection ID from Name
	collection, err := storage.GetCollectionByName(collectionName)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	// 2. Call storage layer
	items, total, err := storage.GetItems(collection.ID, int(page), int(pageSize))
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	// 3. Format a rich response for the LLM
	pageCount := (total + int(pageSize) - 1) / int(pageSize)
	response := map[string]any{
		"items": items,
		"pagination": map[string]any{
			"page":      int(page),
			"pageSize":  int(pageSize),
			"total":     total,
			"pageCount": pageCount,
		},
	}

	resultBytes, _ := json.Marshal(response)
	return string(resultBytes), nil
}

// deleteCollectionItem removes a specific item from a collection using its ID.
func deleteCollectionItem(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	itemIDStr, _ := argMap["item_id"].(string)
	if itemIDStr == "" {
		return `{"status": "error", "message": "missing required parameter: item_id"}`, nil
	}

	itemID, err := strconv.ParseUint(itemIDStr, 10, 32)
	if err != nil || itemID == 0 {
		return `{"status": "error", "message": "invalid item_id format"}`, nil
	}

	// The scope applies to the collection the item really belongs to, and
	// an optional collection_name must match it.
	owner, err := storage.ItemCollectionName(uint(itemID))
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "item '%d' not found"}`, itemID), nil
	}
	if want, _ := argMap["collection_name"].(string); want != "" && want != owner {
		return fmt.Sprintf(`{"status": "error", "message": "item '%d' is not in collection '%s'"}`, itemID, want), nil
	}
	if !collectionAllowed(ctx, owner) {
		return scopeError(owner), nil
	}

	ct, err := storage.GetCollectionByName(owner)
	if err != nil {
		return errorResult(err), nil
	}
	reason, _ := argMap["reason"].(string)
	return writeItem(ctx, ct, models.ProposalDelete, uint(itemID), "", nil, reason), nil
}
