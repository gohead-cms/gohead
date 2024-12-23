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

// CreateItem handles the creation of a new content item.
func CreateItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		tracer := otel.Tracer("gohead")
		ctx, span := tracer.Start(ctx, "CreateItem")
		defer span.End()

		var itemData map[string]interface{}
		if err := c.ShouldBindJSON(&itemData); err != nil {
			logger.Log.
				WithError(err).
				WithField("collection_id", ct.ID).
				Warn("CreateItem: Invalid input")

			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate the incoming payload against the collection schema (fields)
		if err := models.ValidateItemData(ct, itemData); err != nil {
			logger.Log.
				WithError(err).
				WithField("collection_id", ct.ID).
				Warn("CreateItem: Validation failed")

			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// If you have a separate function to validate relationships, pass the same itemData
		// or extract a nested "relationships" key if your JSON is structured that way.
		// Adjust as needed based on how your relationships are actually represented.
		// if err := models.ValidateRelationships(&ct, itemData); err != nil {
		// 	logger.Log.
		// 		WithError(err).
		// 		WithField("collection_id", ct.ID).
		// 		Warn("CreateItem: Validation failed for relationships")

		// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		// 	return
		// }

		item := models.Item{
			CollectionID: ct.ID,
			Data:         models.JSONMap(itemData),
		}

		// Save the main item
		if err := storage.SaveItem(&item); err != nil {
			logger.Log.
				WithError(err).
				WithField("collection_id", ct.ID).
				Error("CreateItem: Save failed")

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save item"})
			return
		}

		// Optionally, save relationships. This might involve linking itemData or a subset
		// specifically for relationships, depending on your data model.
		if err := storage.SaveRelationship(&ct, item.ID, itemData); err != nil {
			logger.Log.
				WithError(err).
				WithField("item_id", item.ID).
				Error("CreateItem: Relationship save failed")

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save relationships"})
			return
		}

		logger.Log.WithFields(logrus.Fields{
			"item_id":       item.ID,
			"collection_id": ct.ID,
		}).Info("CreateItem: Success")

		c.JSON(http.StatusCreated, gin.H{
			"message": "Item created successfully",
			"item":    item,
		})
	}
}

// GetItems retrieves all items in a collection.
func GetItems(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := storage.GetItems(ct.ID)
		if err != nil {
			logger.Log.
				WithError(err).
				WithField("collection_id", ct.ID).
				Error("GetItems: Retrieval failed")

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve items"})
			return
		}

		logger.Log.WithField("collection_id", ct.ID).Info("GetItems: Success")
		c.JSON(http.StatusOK, items)
	}
}

// GetItemByID retrieves a specific item by ID.
func GetItemByID(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Log.
				WithError(err).
				Warn("GetItemByID: Invalid ID")

			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		item, err := storage.GetItemByID(uint(id))
		if err != nil {
			logger.Log.
				WithError(err).
				WithField("item_id", id).
				Warn("GetItemByID: Not found")

			c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
			return
		}

		// Fetch relationships and attach them to the item data
		relations, err := storage.GetRelationships(ct.ID, item.ID)
		if err != nil {
			logger.Log.
				WithError(err).
				WithField("item_id", id).
				Error("GetItemByID: Relationship fetch failed")

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch relationships"})
			return
		}

		for _, rel := range relations {
			relatedItem, err := storage.GetItemByID(*rel.SourceItemID)
			if err != nil {
				logger.Log.
					WithError(err).
					WithField("relation", rel).
					Error("GetItemByID: Related item fetch failed")

				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch related item"})
				return
			}
			// Attach the related item’s data under the relationship’s field key:
			item.Data[rel.Attribute] = relatedItem.Data
		}

		logger.Log.WithField("item_id", id).Info("GetItemByID: Success")
		c.JSON(http.StatusOK, item)
	}
}

// UpdateItem updates an existing content item.
func UpdateItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Log.
				WithError(err).
				Warn("UpdateItem: Invalid ID")

			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		var itemData map[string]interface{}
		if err := c.ShouldBindJSON(&itemData); err != nil {
			logger.Log.
				WithError(err).
				WithField("item_id", id).
				Warn("UpdateItem: Invalid input")

			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Re-validate the updated payload against the collection schema
		if err := models.ValidateItemData(ct, itemData); err != nil {
			logger.Log.
				WithError(err).
				WithField("item_id", id).
				Warn("UpdateItem: Validation failed")

			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Optionally validate relationships again
		// if err := models.ValidateRelationships(&ct, itemData); err != nil {
		// 	logger.Log.
		// 		WithError(err).
		// 		WithField("item_id", id).
		// 		Warn("UpdateItem: Relationship validation failed")

		// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		// 	return
		// }

		if err := storage.UpdateItem(&ct, uint(id), models.JSONMap(itemData)); err != nil {
			logger.Log.
				WithError(err).
				WithField("item_id", id).
				Error("UpdateItem: Update failed")

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item"})
			return
		}

		logger.Log.WithField("item_id", id).Info("UpdateItem: Success")
		c.JSON(http.StatusOK, gin.H{"message": "Item updated successfully"})
	}
}

// DeleteItem deletes a specific content item.
func DeleteItem(ct models.Collection) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Log.
				WithError(err).
				Warn("DeleteItem: Invalid ID")

			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		if err := storage.DeleteItem(uint(id)); err != nil {
			logger.Log.
				WithError(err).
				WithField("item_id", id).
				Error("DeleteItem: Deletion failed")

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item"})
			return
		}

		logger.Log.WithField("item_id", id).Info("DeleteItem: Success")
		c.JSON(http.StatusOK, gin.H{"message": "Item deleted successfully"})
	}
}
