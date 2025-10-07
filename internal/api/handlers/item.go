package handlers

import (
	"net/http"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// CreateItem handles nested creations
func CreateItem(collection models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Data models.JSONMap `json:"data"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Set("response", "Invalid request body")
			c.Set("status", http.StatusBadRequest)
			return
		}

		var finalItem models.Item
		txErr := database.DB.Transaction(func(tx *gorm.DB) error {
			processedData, err := models.ValidateItemValues(collection, input.Data, tx)
			if err != nil {
				return err
			}
			item := models.Item{
				CollectionID: collection.ID,
				Data:         processedData,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			finalItem = item
			return nil
		})

		if txErr != nil {
			c.Set("response", txErr.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		finalItem.Data["id"] = finalItem.ID
		c.Set("response", gin.H{
			"id":         finalItem.ID,
			"attributes": finalItem.Data,
		})
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

// UpdateItem now handles nested creations and uses the c.Set response pattern.
func UpdateItem(collection models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		itemID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.Set("response", "Invalid ID format")
			c.Set("status", http.StatusBadRequest)
			return
		}

		var input struct {
			Data models.JSONMap `json:"data"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Set("response", "Invalid request body")
			c.Set("status", http.StatusBadRequest)
			return
		}

		var updatedItem models.Item
		txErr := database.DB.Transaction(func(tx *gorm.DB) error {
			processedData, err := models.ValidateItemValues(collection, input.Data, tx)
			if err != nil {
				return err
			}
			var itemToUpdate models.Item
			if err := tx.Where("id = ? AND collection_id = ?", itemID, collection.ID).First(&itemToUpdate).Error; err != nil {
				return gorm.ErrRecordNotFound
			}
			itemToUpdate.Data = processedData
			if err := tx.Save(&itemToUpdate).Error; err != nil {
				return err
			}
			updatedItem = itemToUpdate
			return nil
		})

		if txErr != nil {
			if txErr == gorm.ErrRecordNotFound {
				c.Set("response", "Item not found")
				c.Set("status", http.StatusNotFound)
				return
			}
			c.Set("response", txErr.Error())
			c.Set("status", http.StatusBadRequest)
			return
		}

		level, _ := strconv.Atoi(c.DefaultQuery("level", "2"))
		hydratedData, err := storage.FetchNestedRelations(collection, updatedItem.Data, uint(level))
		if err != nil {
			c.Set("response", "Failed to populate relations for response")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{
			"id":         updatedItem.ID,
			"attributes": hydratedData,
		})
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
