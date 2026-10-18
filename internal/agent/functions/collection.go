package functions

import (
	"context"
	"encoding/json"
	"errors"
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

	itemID, _ := strconv.ParseUint(itemIDStr, 10, 32)

	// 2. Decide whether to UPDATE or CREATE
	if itemID > 0 {
		// UPDATE an existing item, validated like API writes.
		if _, err := storage.UpdateItemContent(*collection, uint(itemID), data, nil); err != nil {
			if errors.Is(err, storage.ErrItemNotFound) {
				return fmt.Sprintf(`{"status": "error", "message": "item with ID %d not found in collection '%s' for update"}`, itemID, collectionName), nil
			}
			msg, _ := json.Marshal(err.Error())
			return fmt.Sprintf(`{"status": "error", "message": %s}`, msg), nil
		}

		result := map[string]any{"status": "success", "item_id": itemID, "action": "updated"}
		resultBytes, _ := json.Marshal(result)
		return string(resultBytes), nil

	} else {
		// CREATE a new item
		newItem := &models.Item{
			CollectionID: collection.ID,
			Data:         data,
		}
		newItem, err := storage.SaveItem(*collection, newItem.Data)
		if err != nil {
			return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
		}

		result := map[string]any{"status": "success", "item_id": newItem.ID, "action": "created"}
		resultBytes, _ := json.Marshal(result)
		return string(resultBytes), nil
	}
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

	err = storage.DeleteItem(uint(itemID))
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "failed to delete item: %s"}`, err.Error()), nil
	}

	result := map[string]any{
		"status":  "success",
		"message": fmt.Sprintf("Item '%d' was successfully deleted.", itemID),
	}
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}
