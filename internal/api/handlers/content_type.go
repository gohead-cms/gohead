// internal/api/handlers/content_type.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

func CreateContentType(c *gin.Context) {
	var ct models.ContentType
	if err := c.ShouldBindJSON(&ct); err != nil {
		logger.Log.Warn("Type: Error during create type", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate content type fields
	if err := models.ValidateContentType(ct); err != nil {
		logger.Log.Warn("Type: Error during validate type", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save the content type
	storage.SaveContentType(ct)

	logger.Log.Info("Content Type created successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Content type created"})
}
