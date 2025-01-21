package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
	"go.opentelemetry.io/otel"
)

// GetSingleItem retrieves the content (single item) for a given single type name.
func GetSingleItem(c *gin.Context) {
	singleTypeName := c.Param("name")
	logger.Log.Debugf("Fetching single item for type: %s", singleTypeName)

	// Attempt to get the SingleItem corresponding to this singleTypeName
	item, err := storage.GetSingleItemByType(singleTypeName)
	if err != nil {
		// Strapi-like error response
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

	response := map[string]interface{}{
		"id":         item.ID,
		"attributes": item.Data,
	}

	c.Set("response", response)
	c.Set("status", http.StatusOK)
}

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
	c.Set("response", gin.H{"message": "single type created/updated successfully", "single-type": input})
	c.Set("status", http.StatusCreated)
}

// DeleteSingleType handles deleting a single type by its name (optional).
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

// CreateOrUpdateSingleTypeValue handles the creation or update of a single type content item.
// Assumes you're storing the content in a SingleItem table, separate from the SingleType schema.
func CreateOrUpdateSingleTypeItem(c *gin.Context) {
	// Start OpenTelemetry span
	ctx := c.Request.Context()
	tracer := otel.Tracer("gohead")
	ctx, span := tracer.Start(ctx, "CreateOrUpdateSingleTypeValue")
	defer span.End()

	// The single type name from URL (e.g. /single-types/:name)
	singleTypeName := c.Param("name")

	// Fetch the single type schema
	st, err := storage.GetSingleTypeByName(singleTypeName)
	if err != nil {
		logger.Log.WithError(err).WithField("singleType", singleTypeName).
			Error("Failed to retrieve single type")
		c.Set("response", "Single type not found")
		c.Set("status", http.StatusNotFound)
		return
	}

	// Parse the input JSON -> { "data": { ... } }
	var input struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Set("response", "Invalid input format")
		c.Set("status", http.StatusBadRequest)
		return
	}
	valueData := input.Data

	// Validate user-provided data against the single type’s schema
	if err := models.ValidateSingleItemValues(*st, valueData); err != nil {
		c.Set("response", err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Check if a SingleItem already exists for this SingleType
	existingItem, err := storage.GetSingleItemByType(singleTypeName)
	if err != nil {
		// If the error indicates "no single item found", we proceed to create a new one
		// If it's another DB error, handle accordingly
		logger.Log.WithError(err).WithField("singleType", singleTypeName).
			Warn("Could not retrieve single item - may not exist yet")
		existingItem = nil
	}

	if existingItem != nil {
		// Update the existing single item
		updatedItem, updateErr := storage.UpdateSingleItem(singleTypeName, valueData)
		if updateErr != nil {
			logger.Log.WithError(updateErr).WithField("singleType", singleTypeName).
				Error("Failed to update single type item")
			c.Set("response", "Failed to update single type value")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{"message": "single type updated successfully", "single-type": updatedItem})
		c.Set("detail", updatedItem)
		c.Set("status", http.StatusOK)
	} else {
		// Create a new SingleItem if one does not exist
		newItem, createErr := storage.CreateSingleItem(st, valueData)
		if createErr != nil {
			logger.Log.WithError(createErr).WithField("singleType", singleTypeName).
				Error("Failed to create single type item")
			c.Set("response", "Failed to save single type value")
			c.Set("status", http.StatusInternalServerError)
			return
		}

		c.Set("response", gin.H{"message": "single type updated successfully", "single-type": newItem})
		c.Set("status", http.StatusCreated)
	}
}
