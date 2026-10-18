package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DynamicCollectionHandler handles CRUD operations for dynamic collections.
func DynamicCollectionHandler(c *gin.Context) {
	// Get user role from context
	role, _ := c.Get("role")
	userRole, ok := role.(string)
	if !ok || userRole == "" {
		logger.Log.Warn("DynamicContentHandler: Missing or invalid user role in context")
		c.Set("response", "Unauthorized")
		c.Set("status", http.StatusUnauthorized)
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
		c.Set("response", "Collection not found")
		c.Set("status", http.StatusNotFound)
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
		c.Set("response", "Method not allowed")
		c.Set("status", http.StatusMethodNotAllowed)
	}
}

// handleCreate handles the creation of a single item or a batch of items.
func handleCreate(c *gin.Context, userRole string, ct *models.Collection) {
	// 1. Permission Check
	if !hasPermission(userRole, "create") {
		logger.Log.WithFields(logrus.Fields{
			"user_role":  userRole,
			"collection": ct.Name,
		}).Warn("Create permission denied")
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}

	// 2. Read the raw request body to detect if it's an object or an array
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.Set("response", "Invalid JSON format")
		c.Set("status", http.StatusBadRequest)
		return
	}

	body := bytes.TrimSpace(raw)
	isBatch := len(body) > 0 && body[0] == '['

	var batch []models.JSONMap
	if isBatch {
		var inputs []struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(body, &inputs); err != nil || len(inputs) == 0 {
			c.Set("response", "Invalid JSON array format. Expecting a non-empty array of objects with a 'data' key.")
			c.Set("status", http.StatusBadRequest)
			return
		}
		for _, in := range inputs {
			batch = append(batch, in.Data)
		}
	} else if len(body) > 0 && body[0] == '{' {
		var input struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(body, &input); err != nil || input.Data == nil {
			c.Set("response", "Invalid JSON object format. Expecting an object with a 'data' key.")
			c.Set("status", http.StatusBadRequest)
			return
		}
		batch = []models.JSONMap{input.Data}
	} else {
		c.Set("response", "Invalid or empty JSON body")
		c.Set("status", http.StatusBadRequest)
		return
	}

	// All items are validated and stored in a single transaction.
	created, err := storage.SaveItems(*ct, batch)
	if err != nil {
		var verr *storage.ItemValidationError
		if errors.As(err, &verr) {
			if isBatch {
				c.Set("response", "Validation failed for an item in the batch")
				c.Set("details", gin.H{"item_index": verr.Index, "error": verr.Error()})
			} else {
				c.Set("response", verr.Error())
			}
			c.Set("status", http.StatusBadRequest)
			return
		}
		logger.Log.WithError(err).WithField("collection", ct.Name).Error("Failed to save items")
		c.Set("response", "Failed to save item")
		c.Set("status", http.StatusInternalServerError)
		return
	}

	if !isBatch {
		c.Set("response", utils.FormatCollectionItem(created[0], ct))
		c.Set("meta", gin.H{})
		c.Set("status", http.StatusCreated)
		return
	}
	out := make([]map[string]any, 0, len(created))
	for _, item := range created {
		out = append(out, utils.FormatCollectionItem(item, ct))
	}
	c.Set("response", out)
	c.Set("meta", gin.H{"created_count": len(out)})
	c.Set("status", http.StatusCreated)
}

// handleRead handles fetching items or a single item by ID
// maxRelationLevel caps how deep relations are populated, to bound the work
// (and queries) a single request can trigger.
const maxRelationLevel = 5

func handleRead(c *gin.Context, userRole string, ct *models.Collection, id string) {
	if !hasPermission(userRole, "read") {
		logger.Log.WithFields(logrus.Fields{
			"user_role":  userRole,
			"collection": ct.Name,
		}).Warn("Read permission denied")
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}

	levelParam := c.Query("level")
	level := 1
	if levelParam != "" {
		parsedLevel, err := strconv.Atoi(levelParam)
		if err != nil || parsedLevel < 1 || parsedLevel > maxRelationLevel {
			logger.Log.WithField("level_param", levelParam).Warn("Invalid level parameter")
			c.Set("response", "Invalid level parameter")
			c.Set("status", http.StatusBadRequest)
			return
		}
		level = parsedLevel
	}

	// Handle fetching all items or a single item by ID
	if id == "" {
		GetItems(*ct, uint(level))(c)
	} else {
		itemID, err := strconv.Atoi(id)
		if err != nil {
			c.Set("response", "Invalid ID format")
			c.Set("status", http.StatusBadRequest)
			return
		}
		GetItemByID(*ct, uint(itemID), uint(level))(c)
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
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}
	if id != "" {
		UpdateItem(*ct)(c)
	} else {
		logger.Log.Warn("Update operation requires a valid ID")
		c.Set("response", "ID is required for update")
		c.Set("status", http.StatusBadRequest)
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
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}
	if id != "" {
		DeleteItem(*ct)(c)
	} else {
		logger.Log.Warn("Delete operation requires a valid ID")
		c.Set("response", "ID is required for deletion")
		c.Set("status", http.StatusBadRequest)
	}
}
