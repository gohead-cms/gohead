// internal/api/handlers/dynamic_content.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// internal/api/handlers/dynamic_content.go
func DynamicContentHandler(c *gin.Context) {
	contentTypeName := c.Param("contentType")
	id := c.Param("id")

	// Retrieve the ContentType
	ct, exists := storage.GetContentType(contentTypeName)
	if !exists {
		logger.Log.Warn("Dynamic Content Type: Content Type not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Content type not found"})
		return
	}

	// Get user role from context
	role, _ := c.Get("role")
	userRole := role.(string)

	switch c.Request.Method {
	case http.MethodPost:
		if !hasPermission(userRole, "create") {
			logger.Log.WithFields(logrus.Fields{
				"user_id":      id,
				"content_type": contentTypeName,
			}).Warn("Create denied")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		CreateContentItem(ct)(c)
	case http.MethodGet:
		if !hasPermission(userRole, "read") {
			logger.Log.WithFields(logrus.Fields{
				"user_id":      id,
				"content_type": contentTypeName,
			}).Warn("Read denied")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if id == "" {
			GetContentItems(ct)(c)
		} else {
			GetContentItemByID(ct)(c)
		}
	case http.MethodPut:
		if !hasPermission(userRole, "update") {
			logger.Log.WithFields(logrus.Fields{
				"user_id":      id,
				"content_type": contentTypeName,
			}).Warn("Update denied")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if id != "" {
			logger.Log.WithFields(logrus.Fields{
				"user_id":      id,
				"content_type": contentTypeName,
			}).Info("Content Type updated successfully")
			UpdateContentItem(ct)(c)
		} else {
			logger.Log.WithFields(logrus.Fields{
				"user_id":      id,
				"content_type": contentTypeName,
			}).Warn("ID is required for update")
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required for update"})
		}
	case http.MethodDelete:
		if !hasPermission(userRole, "delete") {
			logger.Log.WithFields(logrus.Fields{
				"user_id":      id,
				"content_type": contentTypeName,
			}).Warn("Delete denied")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if id != "" {
			logger.Log.WithFields(logrus.Fields{
				"user_id":      id,
				"content_type": contentTypeName,
			}).Info("Content Type delete successfully")
			DeleteContentItem(ct)(c)
		} else {
			logger.Log.WithFields(logrus.Fields{
				"user_id":      id,
				"content_type": contentTypeName,
			}).Warn("ID is required for deletion")
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required for deletion"})
		}
	default:
		logger.Log.WithFields(logrus.Fields{
			"user_id":      id,
			"content_type": contentTypeName,
		}).Warn("Method not allowed")
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}
