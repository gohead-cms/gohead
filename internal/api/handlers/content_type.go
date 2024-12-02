package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// GetContentType retrieves a specific content type by its name.
func GetContentType(c *gin.Context) {
	// Extract content type name from the URL parameters
	contentTypeName := c.Param("name")

	// Fetch the content type from storage
	contentType, err := storage.GetContentType(contentTypeName)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"content_type_name": contentTypeName,
		}).Warn("Content type not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Content type not found"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"content_type_name": contentTypeName,
	}).Info("Content type retrieved successfully")
	c.JSON(http.StatusOK, gin.H{"content_type": contentType})
}

// CreateContentType handles the creation of a new content type.
func CreateContentType(c *gin.Context) {
	var input models.ContentType

	// Bind the JSON payload to the ContentType model
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Log.WithError(err).Warn("CreateContentType: Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Validate the ContentType
	if err := models.ValidateContentType(input); err != nil {
		logger.Log.WithError(err).Warn("CreateContentType: Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Save the ContentType to the database
	if err := storage.SaveContentType(&input); err != nil {
		logger.Log.WithError(err).Error("CreateContentType: Failed to save content type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save content type", "details": err.Error()})
		return
	}

	logger.Log.WithField("content_type", input.Name).Info("Content type created successfully")
	c.JSON(http.StatusCreated, gin.H{
		"message":      "Content type created successfully",
		"content_type": input,
	})
}

// DeleteContentType handles deleting a content type along with its fields, items, and related data.
func DeleteContentType(c *gin.Context) {
	contentTypeName := c.Param("name")

	// Check if the user is an admin
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		logger.Log.Warn("Unauthorized attempt to delete content type")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Validate content type existence
	contentType, err := storage.GetContentType(contentTypeName)
	if err != nil {
		logger.Log.WithField("content_type", contentTypeName).Warn("Content type not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Content type not found"})
		return
	}

	// Delete content items and related data
	if err := storage.DeleteContentItemsByType(contentType.Name); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"content_type": contentType.Name,
		}).Error("Failed to delete content items for content type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete content items for content type"})
		return
	}

	// Delete fields for the content type
	if err := storage.DeleteFieldsByContentType(contentType.Name); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"content_type": contentType.Name,
		}).Error("Failed to delete fields for content type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete fields for content type"})
		return
	}

	// Delete the content type
	if err := storage.DeleteContentType(contentType.Name); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"content_type": contentType.Name,
		}).Error("Failed to delete content type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete content type"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"content_type": contentType.Name,
	}).Info("Content type deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Content type deleted successfully"})
}
