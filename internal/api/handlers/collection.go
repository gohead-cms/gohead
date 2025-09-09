package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"

	"github.com/gin-gonic/gin"
)

// GetCollections retrieves a list of collections with optional filtering, sorting, and pagination.
func GetCollections(c *gin.Context) {
	logger.Log.Debug("Handler.GetCollections")

	// Pagination management
	filterParam := c.Query("filter")
	rangeParam := c.Query("range")
	sortParam := c.Query("sort")
	pageParam := c.DefaultQuery("page", "1")
	pageSizeParam := c.DefaultQuery("pageSize", "10")
	page, _ := strconv.Atoi(pageParam)
	pageSize, _ := strconv.Atoi(pageSizeParam)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var filters map[string]any
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

	pageCount := (total + pageSize - 1) / pageSize

	// Format response
	c.Header("Content-Range", formatContentRange(len(collections), total))
	c.Set("response", utils.FormatCollectionsSchema(collections))
	c.Set("status", http.StatusOK)
	c.Set("meta", gin.H{
		"pagination": gin.H{
			"page":      page,
			"pageSize":  pageSize,
			"pageCount": pageCount,
			"total":     total,
		},
	})
	c.Set("status", http.StatusOK)

}

func GetCollection(c *gin.Context) {
	name := c.Param("name")

	// Retrieve collection
	ct, err := storage.GetCollectionByName(name)
	if err != nil {
		c.Set("response", "Collection not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	c.Set("response", utils.FormatCollectionSchema(ct))
	c.Set("status", http.StatusOK)
}

// CreateCollection handles creating a new collection.
func CreateCollection(c *gin.Context) {
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}

	logger.Log.WithField("input", input).Debug("CreateCollection")

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

	// Check if collection with same name already exists
	existing, err := storage.GetCollectionByName(collection.Name)
	if err == nil && existing != nil {
		c.Set("response", "This model already exist")
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
	c.Set("response", utils.FormatCollectionSchema(&collection))
	c.Set("meta", gin.H{"message": "Collection created successfully"})
	c.Set("status", http.StatusCreated)
}

// UpdateCollection handles updating an existing collection.
func UpdateCollection(c *gin.Context) {
	name := c.Param("name") // Collection name (not ID)

	var input map[string]any
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

	// Fetch the existing collection by name
	existing, err := storage.GetCollectionByName(name)
	if err != nil {
		logger.Log.WithError(err).Warn("UpdateCollection: Collection not found")
		c.Set("response", "Collection not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Update in DB
	if err := storage.UpdateCollection(existing.ID, &collection); err != nil {
		logger.Log.WithError(err).Error("UpdateCollection: Failed to update collection")
		c.Set("response", "Failed to update collection")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	// Fetch updated collection
	updated, err := storage.GetCollectionByID(existing.ID)
	if err != nil {
		logger.Log.WithError(err).Error("UpdateCollection: Failed to fetch updated collection")
		c.Set("response", "Failed to fetch updated collection")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	c.Set("response",
		utils.FormatCollectionSchema(updated),
	)
	c.Set("status", http.StatusOK)
}

// DeleteCollectionByIDHandler handles deleting a collection by its ID.
func DeleteCollection(c *gin.Context) {
	name := c.Param("name")

	logger.Log.WithField("collection_name", name).Debug("Handler:DeleteCollection")

	// Fetch the collection by name
	collection, err := storage.GetCollectionByName(name)
	if err != nil {
		logger.Log.WithError(err).WithField("collection_name", name).Warn("DeleteCollection: Collection not found")
		c.Set("response", "Collection not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Now delete using the collection's ID
	if err := storage.DeleteCollection(collection.ID); err != nil {
		logger.Log.WithError(err).WithField("collection_name", name).Error("DeleteCollection: Failed to delete collection")
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	logger.Log.WithField("collection_name", name).Info("Collection deleted successfully")
	c.Set("response", nil)
	c.Set("meta", gin.H{
		"message": "Collection deleted successfully",
	})
	c.Set("status", http.StatusOK)
}

// Helper function to format Content-Range header for pagination
func formatContentRange(count, total int) string {
	if total == 0 {
		return "items */0"
	}
	return "items 0-" + strconv.Itoa(count-1) + "/" + strconv.Itoa(total)
}
