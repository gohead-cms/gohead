package handlers

import (
	"net/http"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"

	"github.com/gin-gonic/gin"
)

// CreateItem handles the creation of a new content item.
func CreateItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Data map[string]any `json:"data"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Set("response", "Invalid input format")
			c.Set("status", http.StatusBadRequest)
			return
		}

		itemData := input.Data
		if err := models.ValidateItemValues(ct, itemData); err != nil {
			c.Set("response", err.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		item := models.Item{
			CollectionID: ct.ID,
			Data:         itemData,
		}

		if err := storage.SaveItem(&item); err != nil {
			c.Set("response", "Failed to save item")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", utils.FormatCollectionItem(&item, &ct))
		c.Set("meta", gin.H{})
		c.Set("status", http.StatusCreated)
	}
}

// GetItems handles pagination and relation hydrating.
func GetItems(collection models.Collection, level uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle pagination from query params
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}

		items, total, err := storage.GetItems(collection.ID, page, pageSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch items"})
			return
		}

		var results []models.JSONMap
		for _, item := range items {
			logger.Log.WithField("item_id", item).Debug("Fetch relation")
			item.Data["id"] = item.ID
			// Use the storage layer to fetch relations
			hydratedData, err := storage.FetchNestedRelations(collection, item.Data, level)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to populate relations"})
				return
			}
			logger.Log.WithField("hydratedData", hydratedData).Debug("Hydrated")
			results = append(results, hydratedData)
		}

		// Construct response with pagination metadata
		c.JSON(http.StatusOK, gin.H{
			"data": results,
			"meta": gin.H{
				"pagination": gin.H{
					"page":      page,
					"pageSize":  pageSize,
					"total":     total,
					"pageCount": (int(total) + pageSize - 1) / pageSize, // Ceiling division
				},
			},
		})
	}
}

// GetItemByID retrieves a single item by ID.
func GetItemByID(ct models.Collection, id uint, level uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.Set("response", "Invalid item ID format")
			c.Set("status", http.StatusBadRequest)
			return
		}

		item, err := storage.GetItemByID(uint(ct.ID), uint(id))
		if err != nil {
			c.Set("response", "Item not found")
			c.Set("status", http.StatusNotFound)
			return
		}

		data, err := storage.FetchNestedRelations(ct, item.Data, level)
		if err != nil {
			c.Set("response", "Failed to fetch item relations")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", utils.FormatNestedItems(item.ID, data, &ct))
		c.Set("status", http.StatusOK)
		c.Set("meta", gin.H{})
	}
}

// UpdateItem updates an existing content item.
func UpdateItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.Set("response", "Invalid ID")
			c.Set("details", err.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		var itemData map[string]any
		if err := c.ShouldBindJSON(&itemData); err != nil {
			c.Set("response", "Invalid input")
			c.Set("details", err.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		if err := models.ValidateItemValues(ct, itemData); err != nil {
			c.Set("response", err.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		if err := storage.UpdateItem(uint(id), models.JSONMap(itemData)); err != nil {
			c.Set("response", "Failed to update item")
			c.Set("details", err.Error())
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{"message": "Item updated successfully"})
		c.Set("status", http.StatusOK)
	}
}

func DeleteItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.Set("response", "Invalid ID")
			c.Set("details", err.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		// Make sure the item belongs to this collection!
		_, err = storage.GetItemByID(ct.ID, uint(id))
		if err != nil {
			c.Set("response", "Item not found in this collection")
			c.Set("details", err.Error())
			c.Set("status", http.StatusNotFound)
			return
		}

		// Now it is safe to delete!
		if err := storage.DeleteItem(uint(id)); err != nil {
			c.Set("response", "Failed to delete item")
			c.Set("details", err.Error())
			c.Set("status", http.StatusInternalServerError)
			return
		}

		logger.Log.WithField("iem of ", ct.Name).Info("Collection deleted successfully")
		c.Set("response", nil)
		c.Set("meta", gin.H{})
		c.Set("status", http.StatusOK)
	}
}
