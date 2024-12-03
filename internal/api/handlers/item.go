// internal/api/handlers/content_item.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"go.opentelemetry.io/otel"
)

// internal/api/handlers/content_item.go
func CreateItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		tracer := otel.Tracer("gohead")
		ctx, span := tracer.Start(ctx, "CreateItem")
		defer span.End()

		var itemData map[string]interface{}
		if err := c.ShouldBindJSON(&itemData); err != nil {
			logger.Log.WithError(err).Warn("Item: Bad Request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate item data, including relationships
		if err := models.ValidateItemData(ct, itemData); err != nil {
			logger.Log.WithError(err).Warn("Item: Error during validation")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create the content item
		item := models.Item{
			CollectionID: ct.ID,
			Data:         models.JSONMap(itemData),
		}

		if err := storage.SaveItem(&item); err != nil {
			logger.Log.WithError(err).Warn("Item: Cannot save item")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Save relationships (if any)
		if err := storage.SaveRelationship(&ct, item.ID, itemData); err != nil {
			logger.Log.WithError(err).Warn("Item: Cannot save item relationship")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		logger.Log.WithFields(logrus.Fields{
			"item_id":       item.ID,
			"collection_id": ct.ID,
		}).Info("Item content created successfully")

		c.JSON(http.StatusCreated, gin.H{
			"message": "Item created successfully",
			"item":    item,
		})
	}
}

func GetItems(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := storage.GetItems(ct.Name)
		if err != nil {
			logger.Log.Warn("Item: cannot get items", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		logger.Log.Info("Item fetch successfully")
		c.JSON(http.StatusOK, items)
	}
}

func GetItemByID(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item: cannot find item", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		item, err := storage.GetItemByID(uint(id))
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item: Item not found", err)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		// Fetch relationships
		relations, err := storage.GetRelationships(ct.Name, item.ID)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item: cannot find content relationship", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Add related items to the response
		for _, rel := range relations {
			relatedItem, err := storage.GetItemByID(rel.RelatedItemID)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{
					"item_id": id,
				}).Warn("Item: internal error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Overwrite the field in item.Data
			item.Data[rel.FieldName] = relatedItem.Data
		}
		logger.Log.WithFields(logrus.Fields{
			"item_id": id,
		}).Info("Item fetch successfully")
		c.JSON(http.StatusOK, item)
	}
}

func UpdateItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item Update: Invalid ID", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		var itemData map[string]interface{}
		if err := c.ShouldBindJSON(&itemData); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item Update: Item not well formatted", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate item data, including relationships
		if err := models.ValidateItemData(ct, itemData); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item Update: Data not valid", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update the content item
		if err := storage.UpdateItem(ct, uint(id), models.JSONMap(itemData)); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item Update: Error during update", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		logger.Log.WithFields(logrus.Fields{
			"item_id": id,
		}).Info("Item updated successfully")
		c.JSON(http.StatusOK, gin.H{"message": "Content item updated"})
	}
}

func DeleteItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item delete: Invalid ID", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		if err := storage.DeleteItem(ct, uint(id)); err != nil {
			logger.Log.WithFields(logrus.Fields{
				"item_id": id,
			}).Warn("Item delete: Internal server error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		logger.Log.WithFields(logrus.Fields{
			"item_id": id,
		}).Info("Item deleted successfully")
		c.JSON(http.StatusOK, gin.H{"message": "Content item deleted"})
	}
}
