package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gitlab.com/sudo.bngz/gohead/internal/models"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

// DynamicCollectionHandler handles CRUD operations for dynamic collections.
func DynamicCollectionHandler(c *gin.Context) {
	// Get user role from context
	role, _ := c.Get("role")
	userRole, ok := role.(string)
	if !ok || userRole == "" {
		logger.Log.Warn("DynamicContentHandler: Missing or invalid user role in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	collectionName := c.Param("collection")
	id := c.Param("id")

	// Retrieve the collection from storage
	ct, err := storage.GetCollectionByName(collectionName)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"collection": collectionName,
		}).Warn("DynamicContentHandler: Collection not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"user_role":      userRole,
		"collection":     collectionName,
		"item_id":        id,
		"request_method": c.Request.Method,
	}).Info("Processing dynamic content request")

	// Handle CRUD operations based on the HTTP method
	switch c.Request.Method {
	case http.MethodPost:
		handleCreate(c, userRole, ct)

	case http.MethodGet:
		handleRead(c, userRole, ct, id)

	case http.MethodPut:
		handleUpdate(c, userRole, ct, id)

	case http.MethodDelete:
		handleDelete(c, userRole, ct, id)

	default:
		logger.Log.WithFields(logrus.Fields{
			"user_role":      userRole,
			"collection":     collectionName,
			"request_method": c.Request.Method,
		}).Warn("Unsupported HTTP method")
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}

// handleCreate handles the creation of an item.
func handleCreate(c *gin.Context, userRole string, ct *models.Collection) {
	if !hasPermission(userRole, "create") {
		logger.Log.WithFields(logrus.Fields{
			"user_role":  userRole,
			"collection": ct.Name,
		}).Warn("Create permission denied")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	CreateItem(*ct)(c)
}

// handleRead handles fetching items or a single item by ID.
func handleRead(c *gin.Context, userRole string, ct *models.Collection, id string) {
	if !hasPermission(userRole, "read") {
		logger.Log.WithFields(logrus.Fields{
			"user_role":  userRole,
			"collection": ct.Name,
		}).Warn("Read permission denied")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	if id == "" {
		GetItems(*ct)(c)
	} else {
		GetItemByID(*ct)(c)
	}
}

// handleUpdate handles updating an item by ID.
func handleUpdate(c *gin.Context, userRole string, ct *models.Collection, id string) {
	if !hasPermission(userRole, "update") {
		logger.Log.WithFields(logrus.Fields{
			"user_role":  userRole,
			"collection": ct.Name,
			"item_id":    id,
		}).Warn("Update permission denied")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	if id != "" {
		UpdateItem(*ct)(c)
	} else {
		logger.Log.Warn("Update operation requires a valid ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required for update"})
	}
}

// handleDelete handles deleting an item by ID.
func handleDelete(c *gin.Context, userRole string, ct *models.Collection, id string) {
	if !hasPermission(userRole, "delete") {
		logger.Log.WithFields(logrus.Fields{
			"user_role":  userRole,
			"collection": ct.Name,
			"item_id":    id,
		}).Warn("Delete permission denied")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	if id != "" {
		DeleteItem(*ct)(c)
	} else {
		logger.Log.Warn("Delete operation requires a valid ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required for deletion"})
	}
}
