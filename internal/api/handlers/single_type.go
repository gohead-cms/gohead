package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// GetSingleType retrieves a single type by its name.
func GetSingleType(c *gin.Context) {
	name := c.Param("name")
	logger.Log.WithField("name", name).Debug("Handler:GetSingleType")

	// Retrieve single type from storage
	st, err := storage.GetSingleTypeByName(name)
	if err != nil {
		logger.Log.WithField("name", name).Warn("GetSingleType: single type not found")
		c.Set("response", "Single type not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"name":       st.Name,
		"attributes": st.Attributes,
	}

	logger.Log.WithField("name", name).Info("GetSingleType: single type retrieved successfully")
	c.Set("response", response)
	c.Set("status", http.StatusOK)
}

// CreateOrUpdateSingleType handles the creation or updating of a single type.
func CreateOrUpdateSingleType(c *gin.Context) {
	name := c.Param("name")

	// Parse the JSON input into a map
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid JSON input")
		c.Set("status", http.StatusBadRequest)
		return
	}
	logger.Log.WithField("payload", input).Info("CreateOrUpdateSingleType payload")

	// Transform the map into a SingleType struct
	singleType, err := models.ParseSingleTypeInput(input)
	if err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Optionally, ensure the name from the route matches the name field
	// or fallback to the route name if not provided in the input.
	if singleType.Name == "" {
		singleType.Name = name
	}

	// Validate the single type
	if err := models.ValidateSingleTypeSchema(singleType); err != nil {
		logger.Log.WithError(err).Warn("CreateOrUpdateSingleType: validation failed")
		c.Set("response", "Validation failed")
		c.Set("details", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Save or update the SingleType in the database
	if err := storage.SaveOrUpdateSingleType(&singleType); err != nil {
		logger.Log.WithError(err).Error("CreateOrUpdateSingleType: Failed to save single type")
		c.Set("response", "Failed to save single type")
		c.Set("details", err.Error())
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithField("singleType", singleType.Name).Info("single type created or updated successfully")
	c.Set("response", gin.H{"message": "single type created/updated successfully", "data": input})
	c.Set("status", http.StatusCreated)
}

// DeleteSingleType handles deleting a single type by its name (optional).
// Strapi doesn’t usually delete single types, but you may implement it if needed.
func DeleteSingleType(c *gin.Context) {
	name := c.Param("name")

	// Fetch the single type by its name
	st, err := storage.GetSingleTypeByName(name)
	if err != nil {
		logger.Log.WithError(err).Warn("DeleteSingleType: single type not found")
		c.Set("response", "Single type not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Call the storage function to delete the single type
	if err := storage.DeleteSingleType(st.ID); err != nil {
		logger.Log.WithError(err).Error("DeleteSingleType: Failed to delete single type")
		c.Set("response", "Failed to delete single type")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"singleType": name,
	}).Info("single type deleted successfully")

	c.Set("response", "single type deleted successfully")
	c.Set("status", http.StatusOK)
}
