package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// DynamicContentHandler handles CRUD operations for dynamic content types.
func DynamicContentHandler(c *gin.Context) {
	contentTypeName := c.Param("contentType")
	id := c.Param("id")

	// Retrieve the ContentType from storage
	ct, err := storage.GetContentTypeByName(contentTypeName)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"content_type": contentTypeName,
		}).Warn("DynamicContentHandler: Content type not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Content type not found"})
		return
	}

	// Get user role from context
	role, _ := c.Get("role")
	userRole, ok := role.(string)
	if !ok || userRole == "" {
		logger.Log.Warn("DynamicContentHandler: Missing or invalid user role in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"user_role":      userRole,
		"content_type":   contentTypeName,
		"content_item":   id,
		"request_method": c.Request.Method,
	}).Info("Processing dynamic content request")

	switch c.Request.Method {
	case http.MethodPost:
		if !hasPermission(userRole, "create") {
			logger.Log.WithFields(logrus.Fields{
				"user_role":    userRole,
				"content_type": contentTypeName,
			}).Warn("Create permission denied")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		CreateContentItem(*ct)(c)

	case http.MethodGet:
		if !hasPermission(userRole, "read") {
			logger.Log.WithFields(logrus.Fields{
				"user_role":    userRole,
				"content_type": contentTypeName,
			}).Warn("Read permission denied")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if id == "" {
			GetContentItems(*ct)(c)
		} else {
			GetContentItemByID(*ct)(c)
		}

	case http.MethodPut:
		if !hasPermission(userRole, "update") {
			logger.Log.WithFields(logrus.Fields{
				"user_role":    userRole,
				"content_type": contentTypeName,
				"content_item": id,
			}).Warn("Update permission denied")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if id != "" {
			UpdateContentItem(*ct)(c)
		} else {
			logger.Log.Warn("Update operation requires a valid ID")
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required for update"})
		}

	case http.MethodDelete:
		if !hasPermission(userRole, "delete") {
			logger.Log.WithFields(logrus.Fields{
				"user_role":    userRole,
				"content_type": contentTypeName,
				"content_item": id,
			}).Warn("Delete permission denied")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if id != "" {
			DeleteContentItem(*ct)(c)
		} else {
			logger.Log.Warn("Delete operation requires a valid ID")
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required for deletion"})
		}

	default:
		logger.Log.WithFields(logrus.Fields{
			"user_role":      userRole,
			"content_type":   contentTypeName,
			"request_method": c.Request.Method,
		}).Warn("Unsupported HTTP method")
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}
