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
	logger.Log.WithField("name", name).Debug("Handler:GetCollection")
	ct, err := storage.GetCollectionByName(name)
	if err != nil {
		logger.Log.WithField("name", name).Warn("GetCollection: collection not found")
		c.Set("response", gin.H{"error": "Collection not found"})
		c.Set("status", http.StatusNotFound)
		return
	}

	// Flatten the response
	response := map[string]interface{}{
		"name":       ct.Name,
		"attributes": ct.Attributes,
	}

	logger.Log.WithField("name", name).Info("GetCollection: collection retrieved successfully")
	c.Set("response", response)
	c.Set("status", http.StatusOK)
}

// CreateCollection handles the creation of a new collection.
func CreateCollection(c *gin.Context) {
	// Parse the JSON input into a map
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", gin.H{"error": "Invalid JSON input"})
		c.Set("status", http.StatusBadRequest)
		return
	}
	logger.Log.WithField("name", input).Info("CreateCollection")

	// Transform the map into a Collection struct
	collection, err := models.ParseCollectionInput(input)
	if err != nil {
		c.Set("response", gin.H{"error": err.Error()})
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Validate the Collection
	if err := models.ValidateCollectionSchema(collection); err != nil {
		logger.Log.WithError(err).Warn("CreateCollection: Validation failed")
		c.Set("response", gin.H{"error": "Validation failed", "details": err.Error()})
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Save the Collection to the database
	if err := storage.SaveCollection(&collection); err != nil {
		logger.Log.WithError(err).Error("CreateCollection: Failed to save collection")
		c.Set("response", gin.H{"error": "Failed to save collection", "details": err.Error()})
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithField("collection", collection.Name).Info("collection created successfully")
	c.Set("response", gin.H{"message": "collection created successfully", "collection": input})
	c.Set("status", http.StatusCreated)
}

// UpdateCollection handles updating an existing collection.
func UpdateCollection(c *gin.Context) {
	// Extract the collection name from the path
	CollectionName := c.Param("name")

	// Parse the JSON input into a map
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", gin.H{"error": "Invalid JSON input"})
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Transform the map into a Collection struct
	collection, err := models.ParseCollectionInput(input)
	if err != nil {
		c.Set("response", gin.H{"error": err.Error()})
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Validate the input collection
	if err := models.ValidateCollectionSchema(collection); err != nil {
		logger.Log.WithError(err).Warn("UpdateCollection: Validation failed")
		c.Set("response", gin.H{"error": "Validation failed: " + err.Error()})
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Attempt to update the collection
	if err := storage.UpdateCollection(CollectionName, &collection); err != nil {
		logger.Log.WithError(err).WithField("collection", CollectionName).Error("UpdateCollection: Failed to update collection")
		c.Set("response", gin.H{"error": "Failed to update collection", "details": err.Error()})
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"collection": CollectionName,
	}).Info("collection updated successfully")

	// Respond with success
	c.Set("response", gin.H{"message": "collection updated successfully"})
	c.Set("status", http.StatusOK)
}

// DeleteCollectionHandler handles deleting a collection by its name.
func DeleteCollection(c *gin.Context) {
	CollectionName := c.Param("name")

	// Fetch the collection by its name
	Collection, err := storage.GetCollectionByName(CollectionName)
	if err != nil {
		logger.Log.WithError(err).Warn("DeleteCollectionHandler: collection not found")
		c.Set("response", gin.H{"error": "Collection not found"})
		c.Set("status", http.StatusNotFound)
		return
	}

	// Call the storage function to delete the collection
	if err := storage.DeleteCollection(Collection.ID); err != nil {
		logger.Log.WithError(err).Error("DeleteCollectionHandler: Failed to delete collection")
		c.Set("response", gin.H{"error": "Failed to delete collection"})
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"collection": CollectionName,
	}).Info("collection deleted successfully")

	c.Set("response", gin.H{"message": "collection deleted successfully"})
	c.Set("status", http.StatusOK)
}
