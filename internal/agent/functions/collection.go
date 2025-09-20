package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"
	"gorm.io/gorm"
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

	collectionNames := make([]string, len(collections))
	for i, c := range collections {
		collectionNames[i] = c.Name
	}

	resultBytes, _ := json.Marshal(collectionNames)
	return string(resultBytes), nil
}

// getCollectionSchema retrieves the structure of a specific collection.
func getCollectionSchema(ctx context.Context, args any) (string, error) {
	argMap, ok := args.(map[string]any)
	if !ok {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	collectionName, _ := argMap["collection_name"].(string)
	if collectionName == "" {
		return `{"status": "error", "message": "missing required parameter: collection_name"}`, nil
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
	argString, ok := args.(string)
	if !ok {
		// This would happen if something other than a string is passed.
		return `{"status": "error", "message": "invalid arguments type, expected a string"}`, nil
	}

	var argMap map[string]any
	err := json.Unmarshal([]byte(argString), &argMap)
	if err != nil {
		// This handles cases where the LLM sends a malformed JSON string.
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

	// 1. Get Collection ID from Name
	collection, err := storage.GetCollectionByName(collectionName)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	itemID, _ := strconv.ParseUint(itemIDStr, 10, 64)

	// 2. Decide whether to UPDATE or CREATE
	if itemID > 0 {
		// Attempt to UPDATE an existing item
		_, err := storage.GetItemByID(collection.ID, uint(itemID))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Sprintf(`{"status": "error", "message": "item with ID %d not found in collection '%s' for update"}`, itemID, collectionName), nil
			}
			return fmt.Sprintf(`{"status": "error", "message": "error checking for item: %s"}`, err.Error()), nil
		}

		err = storage.UpdateItem(uint(itemID), data)
		if err != nil {
			return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
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
		err = storage.SaveItem(newItem)
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
	argMap, ok := args.(map[string]any)
	if !ok {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	collectionName, _ := argMap["collection_name"].(string)
	if collectionName == "" {
		return `{"status": "error", "message": "missing required parameter: collection_name"}`, nil
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
	argMap, ok := args.(map[string]any)
	if !ok {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	itemIDStr, _ := argMap["item_id"].(string)
	if itemIDStr == "" {
		return `{"status": "error", "message": "missing required parameter: item_id"}`, nil
	}

	itemID, err := strconv.ParseUint(itemIDStr, 10, 64)
	if err != nil || itemID == 0 {
		return `{"status": "error", "message": "invalid item_id format"}`, nil
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
