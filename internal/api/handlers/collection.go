package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// GetCollection retrieves a specific collection by its name.
func GetCollection(c *gin.Context) {
	name := c.Param("name")

	// Retrieve collection from storage
	ct, err := storage.GetCollectionByName(name)
	if err != nil {
		logger.Log.WithField("name", name).Warn("GetCollection: collection not found")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Flatten the response
	response := map[string]interface{}{
		"name":          ct.Name,
		"fields":        ct.Fields,
		"relationships": ct.Relationships,
	}

	logger.Log.WithField("name", name).Info("GetCollection: collection retrieved successfully")
	c.JSON(http.StatusOK, response)
}

// CreateCollection handles the creation of a new collection.
func CreateCollection(c *gin.Context) {
	var input models.Collection

	// Bind the JSON payload to the Collection model
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("CreateCollection: Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Validate the Collection
	if err := models.ValidateCollection(input); err != nil {
		logger.Log.WithError(err).Warn("CreateCollection: Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Save the Collection to the database
	if err := storage.SaveCollection(&input); err != nil {
		logger.Log.WithError(err).Error("CreateCollection: Failed to save collection")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to save collection", "details": err.Error()})
		return
	}

	logger.Log.WithField("collection", input.Name).Info("collection created successfully")
	c.JSON(http.StatusCreated, gin.H{
		"message":    "collection created successfully",
		"collection": input,
	})
}

// UpdateCollection handles updating an existing collection.
func UpdateCollection(c *gin.Context) {
	// Extract the collection name from the path
	CollectionName := c.Param("name")

	// Parse the incoming request body into a Collection model
	var input models.Collection
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("UpdateCollection: Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	logger.Log.WithField("collection", CollectionName).Info("Attempting to update collection")

	// Validate the input collection
	if err := models.ValidateCollection(input); err != nil {
		logger.Log.WithError(err).Warn("UpdateCollection: Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed: " + err.Error()})
		return
	}

	// Attempt to update the collection
	if err := storage.UpdateCollection(CollectionName, &input); err != nil {
		logger.Log.WithError(err).WithField("collection", CollectionName).Error("UpdateCollection: Failed to update collection")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update collection",
			"details": err.Error(),
		})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"collection": CollectionName,
	}).Info("collection updated successfully")

	// Respond with success
	c.JSON(http.StatusOK, gin.H{
		"message": "collection updated successfully",
	})
}

// DeleteCollectionHandler handles deleting a collection by its name.
func DeleteCollection(c *gin.Context) {
	CollectionName := c.Param("name")

	// Fetch the collection by its name
	Collection, err := storage.GetCollectionByName(CollectionName)
	if err != nil {
		logger.Log.WithError(err).Warn("DeleteCollectionHandler: collection not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	// Call the storage function to delete the collection
	if err := storage.DeleteCollection(Collection.ID); err != nil {
		logger.Log.WithError(err).Error("DeleteCollectionHandler: Failed to delete collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete collection"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"collection": CollectionName,
	}).Info("collection deleted successfully")

	c.JSON(http.StatusOK, gin.H{"message": "collection deleted successfully"})
}
