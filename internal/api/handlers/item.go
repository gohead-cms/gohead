package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
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

		// Parse incoming JSON payload
		var input struct {
			Data map[string]interface{} `json:"data"` // Strapi-style payload
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			logger.Log.
				WithError(err).
				WithField("collection_id", ct.ID).
				Warn("CreateItem: Invalid input")

			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format"})
			return
		}

		itemData := input.Data

		// Validate the incoming payload against the collection schema
		if err := models.ValidateItemValues(ct, itemData); err != nil {
			logger.Log.
				WithError(err).
				WithField("collection_id", ct.ID).
				Warn("CreateItem: Validation failed")

			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create the main item
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

func GetItems(ct models.Collection, level uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			logger.Log.WithError(err).WithField("id", id).Error("Failed to fetch nested relations")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch item relations"})
			return
		}

		// Use storage package to fetch the item
		item, err := storage.GetItemByID(uint(ct.ID), uint(id))
		if err != nil {
			logger.Log.WithError(err).WithField("item_id", id).Warn("Item not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
			return
		}

		// Fetch nested relations
		data, err := fetchNestedRelations(ct, item.Data, level)
		if err != nil {
			logger.Log.WithError(err).WithField("item_id", id).Error("Failed to fetch nested relations")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch item relations"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": data})
	}
}

// fetchNestedRelations recursively fetches nested relationships up to a specified level.
func fetchNestedRelations(ct models.Collection, data map[string]interface{}, level uint) (map[string]interface{}, error) {
	if level <= 0 {
		return data, nil // Stop recursion when level is zero
	}
	logger.Log.WithField("data", data).Debug("fetchNestedRelations:data")
	result := make(map[string]interface{})
	for key, value := range data {
		// Check if the key is a relationship
		attribute := findAttribute(ct.Attributes, key)
		if attribute == nil || attribute.Type != "relation" {
			result[key] = value
			continue
		}

		// Handle relationships
		switch attribute.Relation {
		case "oneToOne", "oneToMany":
			id, ok := value.(float64)
			if !ok {
				result[key] = value
				continue
			}
			relatedData, err := fetchRelatedItem(attribute.Target, uint(id), level-1)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch relation '%s': %w", key, err)
			}
			result[key] = relatedData

		case "manyToMany":
			array, ok := value.([]interface{})
			if !ok {
				result[key] = value
				continue
			}
			relatedItems := []map[string]interface{}{}
			for _, element := range array {
				if id, isID := element.(float64); isID {
					relatedData, err := fetchRelatedItem(attribute.Target, uint(id), level-1)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch relation '%s': %w", key, err)
					}
					relatedItems = append(relatedItems, relatedData)
				}
			}
			result[key] = relatedItems

		default:
			result[key] = value
		}
	}
	return result, nil
}

// fetchRelatedItem fetches a single related item and its relations.
func fetchRelatedItem(target string, id uint, level uint) (map[string]interface{}, error) {
	var relatedCollection models.Collection
	if err := database.DB.Where("name = ?", target).First(&relatedCollection).Error; err != nil {
		return nil, fmt.Errorf("target collection '%s' not found: %w", target, err)
	}

	var relatedItem models.Item
	if err := database.DB.Where("id = ? AND collection_id = ?", id, relatedCollection.ID).First(&relatedItem).Error; err != nil {
		return nil, fmt.Errorf("related item not found in collection '%s': %w", target, err)
	}

	return fetchNestedRelations(relatedCollection, relatedItem.Data, level)
}

// findAttribute retrieves an attribute from a collection by name.
func findAttribute(attributes []models.Attribute, name string) *models.Attribute {
	for _, attr := range attributes {
		if attr.Name == name {
			return &attr
		}
	}
	return nil
}

// GetItemsByID fetches an item by ID, including nested relations up to the specified level.
func GetItemByID(ct models.Collection, id uint, level uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse item ID from the URL
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Log.WithError(err).WithField("id", idStr).Warn("Invalid item ID format")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID format"})
			return
		}

		// Fetch the item from the storage layer
		item, err := storage.GetItemByID(uint(ct.ID), uint(id))
		if err != nil {
			logger.Log.WithError(err).WithField("item_id", id).Warn("Item not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
			return
		}

		// Fetch nested relationships for the item
		data, err := storage.FetchNestedRelations(ct, item.Data, level)
		if err != nil {
			logger.Log.WithError(err).WithField("item_id", id).Error("Failed to fetch nested relations")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch item relations"})
			return
		}

		// Respond with the fetched item and nested relationships
		c.JSON(http.StatusOK, gin.H{"data": data})
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
		if err := models.ValidateItemValues(ct, itemData); err != nil {
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

		if err := storage.UpdateItem(uint(id), models.JSONMap(itemData)); err != nil {
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
