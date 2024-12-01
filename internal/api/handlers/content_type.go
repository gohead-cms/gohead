package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// CreateContentType handles the creation of a new content type.
func CreateContentType(c *gin.Context) {
	var input models.ContentType

	// Bind the JSON payload to the ContentType model
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("CreateContentType: Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Validate the ContentType
	if err := models.ValidateContentType(input); err != nil {
		logger.Log.WithError(err).Warn("CreateContentType: Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save the ContentType to the database
	if err := storage.SaveContentType(&input); err != nil {
		logger.Log.WithError(err).Error("CreateContentType: Failed to save content type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save content type"})
		return
	}

	logger.Log.WithField("content_type", input.Name).Info("Content type created successfully")
	c.JSON(http.StatusCreated, gin.H{"message": "Content type created successfully", "content_type": input})
}
