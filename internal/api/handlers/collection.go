package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gohead/internal/models"
	"gohead/pkg/logger"
	"gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// GetCollections retrieves a list of collections with optional filtering, sorting, and pagination.
func GetCollections(c *gin.Context) {
	logger.Log.Debug("Handler:GetCollections")

	// Extract optional query parameters
	filterParam := c.Query("filter")
	rangeParam := c.Query("range")
	sortParam := c.Query("sort")

	var filters map[string]interface{}
	var rangeValues []int
	var sortValues []string

	// Parse filter (JSON object)
	if filterParam != "" {
		if err := json.Unmarshal([]byte(filterParam), &filters); err != nil {
			logger.Log.WithError(err).Warn("Invalid filter format")
			c.Set("response", "Invalid filter format")
			c.Set("status", http.StatusBadRequest)
			return
		}
	}

	// Parse range (JSON array [start, end])
	if rangeParam != "" {
		if err := json.Unmarshal([]byte(rangeParam), &rangeValues); err != nil || len(rangeValues) != 2 {
			logger.Log.WithError(err).Warn("Invalid range format")
			c.Set("response", "Invalid range format")
			c.Set("status", http.StatusBadRequest)
			return
		}
	}

	// Parse sort (JSON array ["field", "ASC/DESC"])
	if sortParam != "" {
		if err := json.Unmarshal([]byte(sortParam), &sortValues); err != nil || len(sortValues) != 2 {
			logger.Log.WithError(err).Warn("Invalid sort format")
			c.Set("response", "Invalid sort format")
			c.Set("status", http.StatusBadRequest)
			return
		}
	} else {
		// Default sorting: ID ASC
		sortValues = []string{"id", "ASC"}
	}

	// Retrieve collections from storage with optional filters, sorting, and pagination
	collections, total, err := storage.GetAllCollections(filters, sortValues, rangeValues)
	if err != nil {
		logger.Log.WithError(err).Warn("GetCollections: failed to retrieve collections")
		c.Set("response", "Failed to fetch collections")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	// Format response
	c.Set("response", collections)
	c.Header("Content-Range", formatContentRange(len(collections), total))
	c.Set("status", http.StatusOK)
}

// GetCollectionByID retrieves a specific collection by its ID.
func GetCollection(c *gin.Context) {
	idParam := c.Param("id")

	// Convert ID to uint
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		logger.Log.WithField("id", idParam).Warn("GetCollectionByID: Invalid collection ID format")
		c.Set("response", "Invalid collection ID format")
		c.Set("status", http.StatusBadRequest)
		return
	}

	logger.Log.WithField("id", id).Debug("Handler:GetCollectionByID")
	ct, err := storage.GetCollectionByID(uint(id))
	if err != nil {
		logger.Log.WithField("id", id).Warn("GetCollectionByID: Collection not found")
		c.Set("response", "Collection not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Format response
	response := map[string]interface{}{
		"id":         ct.ID,
		"name":       ct.Name,
		"attributes": ct.Attributes,
	}

	logger.Log.WithField("id", id).Info("GetCollectionByID: Collection retrieved successfully")
	c.Set("response", response)
	c.Set("status", http.StatusOK)
}

// CreateCollection handles creating a new collection.
func CreateCollection(c *gin.Context) {
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}

	logger.Log.WithField("input", input).Info("CreateCollection")

	collection, err := models.ParseCollectionInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := models.ValidateCollectionSchema(collection); err != nil {
		logger.Log.WithError(err).Warn("CreateCollection: Validation failed")
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := storage.SaveCollection(&collection); err != nil {
		logger.Log.WithError(err).Error("CreateCollection: Failed to save collection")
		c.Set("response", "Failed to save collection")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithField("collection", collection.Name).Info("Collection created successfully")
	c.Set("response", gin.H{"message": "Collection created successfully", "collection": collection})
	c.Set("status", http.StatusCreated)
}

// UpdateCollection handles updating an existing collection.
func UpdateCollection(c *gin.Context) {
	id := c.Param("id")

	// Convert ID from string to uint
	idInt, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		logger.Log.WithField("id", id).Warn("DeleteCollectionByIDHandler: invalid collection ID format")
		c.Set("response", "Invalid collection ID")
		c.Set("status", http.StatusBadRequest)
		return
	}

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}

	collection, err := models.ParseCollectionInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := models.ValidateCollectionSchema(collection); err != nil {
		logger.Log.WithError(err).Warn("UpdateCollection: Validation failed")
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	if err := storage.UpdateCollection(uint(idInt), &collection); err != nil {
		logger.Log.WithError(err).Error("UpdateCollection: Failed to update collection")
		c.Set("response", "Failed to update collection")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithField("collection", collection.Name).Info("Collection updated successfully")
	c.Set("response", "Collection updated successfully")
	c.Set("status", http.StatusOK)
}

// DeleteCollectionByIDHandler handles deleting a collection by its ID.
func DeleteCollection(c *gin.Context) {
	idParam := c.Param("id")

	// Convert ID from string to uint
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		logger.Log.WithField("id", idParam).Warn("DeleteCollectionByIDHandler: invalid collection ID format")
		c.Set("response", "Invalid collection ID")
		c.Set("status", http.StatusBadRequest)
		return
	}

	logger.Log.WithField("collection_id", id).Debug("Handler:DeleteCollectionByID")

	// Directly delete the collection without fetching it
	if err := storage.DeleteCollection(uint(id)); err != nil {
		logger.Log.WithError(err).WithField("collection_id", id).Error("DeleteCollectionByIDHandler: Failed to delete collection")
		c.Set("response", "Failed to delete collection")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithField("collection_id", id).Info("Collection deleted successfully")
	c.Set("response", "Collection deleted successfully")
	c.Set("status", http.StatusOK)
}

// Helper function to format Content-Range header for pagination
func formatContentRange(count, total int) string {
	if total == 0 {
		return "items */0"
	}
	return "items 0-" + strconv.Itoa(count-1) + "/" + strconv.Itoa(total)
}
