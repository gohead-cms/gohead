// internal/api/handlers/content_item.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

func CreateContentItem(ct models.ContentType) gin.HandlerFunc {
	return func(c *gin.Context) {
		var itemData map[string]interface{}
		if err := c.ShouldBindJSON(&itemData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate item fields based on ct.Fields
		if err := models.ValidateItemData(ct, itemData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create the content item
		item := models.ContentItem{
			ContentType: ct.Name,
			Data:        itemData,
		}

		if err := storage.SaveContentItem(&item); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, item)
	}
}

func GetContentItems(ct models.ContentType) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := storage.GetContentItems(ct.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, items)
	}
}

func GetContentItemByID(ct models.ContentType) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		item, err := storage.GetContentItemByID(ct.Name, uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, item)
	}
}

func UpdateContentItem(ct models.ContentType) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		var itemData map[string]interface{}
		if err := c.ShouldBindJSON(&itemData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate item fields based on ct.Fields
		if err := models.ValidateItemData(ct, itemData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update the content item
		if err := storage.UpdateContentItem(ct.Name, uint(id), itemData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Content item updated"})
	}
}

func DeleteContentItem(ct models.ContentType) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		if err := storage.DeleteContentItem(ct.Name, uint(id)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Content item deleted"})
	}
}
