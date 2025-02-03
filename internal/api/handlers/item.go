package handlers

import (
	"net/http"
	"strconv"

	"gohead/internal/models"
	"gohead/pkg/logger"
	"gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// CreateItem handles the creation of a new content item.
func CreateItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Data map[string]interface{} `json:"data"`
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
			Data:         models.JSONMap(itemData),
		}
		if err := storage.SaveItem(&item); err != nil {
			c.Set("response", "Failed to save item")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{
			"message": "Item created successfully",
			ct.Name:   item,
		})
		c.Set("status", http.StatusCreated)
	}
}

// GetItems retrieves items from a collection with optional nested relationships and pagination.
func GetItems(ct models.Collection, level uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		pageParam := c.DefaultQuery("page", "1")
		pageSizeParam := c.DefaultQuery("pageSize", "10")
		page, _ := strconv.Atoi(pageParam)
		pageSize, _ := strconv.Atoi(pageSizeParam)

		items, totalItems, err := storage.GetItems(ct.ID, page, pageSize)
		if err != nil {
			c.Set("response", "Failed to fetch items")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		logger.Log.WithField("totalItems", totalItems).Debug("handler:GetItems:totalItems")
		totalPages := (totalItems + pageSize - 1) / pageSize
		c.Set("response", gin.H{ct.Name: items})
		c.Set("meta", gin.H{
			"pagination": gin.H{
				"totalItems":  totalItems,
				"totalPages":  totalPages,
				"currentPage": page,
				"pageSize":    pageSize,
			},
		})
		c.Set("status", http.StatusOK)
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

		c.Set("response", gin.H{ct.Name: data})
		c.Set("status", http.StatusOK)
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

		var itemData map[string]interface{}
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

// DeleteItem deletes a specific content item.
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

		if err := storage.DeleteItem(uint(id)); err != nil {
			c.Set("response", "Failed to delete item")
			c.Set("details", err.Error())
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{"message": "Item deleted successfully"})
		c.Set("status", http.StatusOK)
	}
}
