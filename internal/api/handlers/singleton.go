package handlers

import (
	"net/http"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GetSingleItem retrieves the content (single item) for a given single type name.
func GetSingleItem(c *gin.Context) {
	SingletonName := c.Param("name")
	logger.Log.Debugf("Fetching single item for type: %s", SingletonName)

	// Attempt to get the SingleItem corresponding to this SingletonName
	item, err := storage.GetSingleItemByType(SingletonName)
	if err != nil {
		c.Set("response", gin.H{
			"data": nil,
			"error": gin.H{
				"status":  http.StatusNotFound,
				"name":    "NotFoundError",
				"message": err.Error(),
				"details": gin.H{},
			},
		})
		c.Set("status", http.StatusNotFound)
		return
	}

	response := map[string]any{
		"id":         item.ID,
		"attributes": item.Data,
	}

	c.Set("response", response)
	c.Set("status", http.StatusOK)
}

// GetSingleton retrieves a single type by its name.
func GetSingleton(c *gin.Context) {
	name := c.Param("name")
	logger.Log.WithField("name", name).Debug("Handler.GetSingleton")

	// Retrieve single type from storage
	st, err := storage.GetSingletonByName(name)
	if err != nil {
		logger.Log.WithField("name", name).Warn("GetSingleton: single type not found")
		c.Set("response", "Single type not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Prepare response
	response := map[string]any{
		"name":       st.Name,
		"attributes": st.Attributes,
	}

	logger.Log.WithField("name", name).Info("GetSingleton: single type retrieved successfully")
	c.Set("response", response)
	c.Set("status", http.StatusOK)
}

// CreateOrUpdateSingleton handles the creation or updating of a single type.
func CreateOrUpdateSingleton(c *gin.Context) {
	name := c.Param("name")

	// Parse the JSON input into a map
	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}
	logger.Log.WithField("payload", input).Info("CreateOrUpdateSingleton payload")

	// Transform the map into a Singleton struct
	Singleton, err := models.ParseSingletonInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Optionally, ensure the name from the route matches the name field
	// or fallback to the route name if not provided in the input.
	if Singleton.Name == "" {
		Singleton.Name = name
	}

	// Validate the single type
	if err := models.ValidateSingletonSchema(Singleton); err != nil {
		logger.Log.WithError(err).Warn("CreateOrUpdateSingleton: validation failed")
		c.Set("response", gin.H{"message": "Validation schema for failed", "single_type": input})
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Save or update the Singleton in the database
	if err := storage.SaveOrUpdateSingleton(&Singleton); err != nil {
		logger.Log.WithError(err).Error("CreateOrUpdateSingleton: Failed to save single type")
		c.Set("response", gin.H{"message": "Failed to save single type", "single_type": input})
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithField("Singleton", Singleton.Name).Info("single type created or updated successfully")
	c.Set("response", gin.H{"message": "single type created/updated successfully", "single_type": input})
	c.Set("status", http.StatusCreated)
}

// DeleteSingleton handles deleting a single type by its name (optional).
func DeleteSingleton(c *gin.Context) {
	name := c.Param("name")

	// Fetch the single type by its name
	st, err := storage.GetSingletonByName(name)
	if err != nil {
		logger.Log.WithError(err).Warn("DeleteSingleton: single type not found")
		c.Set("response", gin.H{"message": "Single type not found", "details": err})
		c.Set("status", http.StatusNotFound)
		return
	}

	// Call the storage function to delete the single type
	if err := storage.DeleteSingleton(st.ID); err != nil {
		logger.Log.WithError(err).Error("DeleteSingleton: Failed to delete single type")
		c.Set("response", gin.H{"message": "Failed to delete single type", "details": err})
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"Singleton": name,
	}).Info("single type deleted successfully")

	c.Set("response", gin.H{"message": "single type delete successfully", "single_type": name})
	c.Set("status", http.StatusOK)
}

// CreateOrUpdateSingletonValue handles the creation or update of a single type content item.
func CreateOrUpdateSingletonItem(c *gin.Context) {
	// The single type name from URL (e.g. /single-types/:name)
	SingletonName := c.Param("name")

	// Fetch the single type schema
	st, err := storage.GetSingletonByName(SingletonName)
	if err != nil {
		logger.Log.WithError(err).WithField("Singleton", SingletonName).
			Error("Failed to retrieve single type")
		c.Set("response", gin.H{"message": "Single type not found", "details": err})
		c.Set("status", http.StatusNotFound)
		return
	}
	logger.Log.WithField("single_type", st).Debug("handler:CreateOrUpdateSingletonItem")

	// Parse the input JSON -> { "data": { ... } }
	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", gin.H{"message": "Invalid input format", "details": err})
		c.Set("status", http.StatusBadRequest)
		return
	}
	valueData := input.Data

	// Validate user-provided data against the single type’s schema
	if err := models.ValidateSingleItemValues(*st, valueData); err != nil {
		c.Set("response", gin.H{"message": "Failed to validate single type value", "details": err.Error()})
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Check if a SingleItem already exists for this Singleton
	existingItem, err := storage.GetSingleItemByType(SingletonName)
	if err != nil {
		// If the error indicates "no single item found", we proceed to create a new one
		// If it's another DB error, handle accordingly
		logger.Log.WithError(err).WithField("Singleton", SingletonName).
			Warn("Could not retrieve single item - may not exist yet")
		existingItem = nil
	}

	if existingItem != nil {
		// Update the existing single item
		updatedItem, updateErr := storage.UpdateSingleItem(SingletonName, valueData)
		if updateErr != nil {
			logger.Log.WithError(updateErr).WithField("Singleton", SingletonName).
				Error("Failed to update single type item")
			c.Set("response", gin.H{"message": "Failed to update single type value", "details": err})
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{"message": "single type updated successfully", "single_type": updatedItem})
		c.Set("detail", updatedItem)
		c.Set("status", http.StatusOK)
	} else {
		// Create a new SingleItem if one does not exist
		newItem, createErr := storage.CreateSingleItem(st, valueData)
		if createErr != nil {
			logger.Log.WithError(createErr).WithField("Singleton", SingletonName).
				Error("Failed to create single type item")
			c.Set("response", gin.H{"message": "Failed to save single type value", "details": err})
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{"message": "single type updated successfully", "single_type": newItem})
		c.Set("status", http.StatusCreated)
	}
}
