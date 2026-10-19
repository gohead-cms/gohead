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

	rc, ok := writeContext(c)
	if !ok {
		return
	}
	if rc.translated() {
		validationError(c, "create entries in the default locale ("+rc.defaultLocale+"), then add translations with PUT ?locale="+rc.locale)
		return
	}

	var bodies []itemWriteBody
	if isBatch {
		if err := json.Unmarshal(body, &bodies); err != nil || len(bodies) == 0 {
			c.Set("response", "Invalid JSON array format. Expecting a non-empty array of objects with a 'data' key.")
			c.Set("status", http.StatusBadRequest)
			return
		}
	} else if len(body) > 0 && body[0] == '{' {
		var input itemWriteBody
		if err := json.Unmarshal(body, &input); err != nil || input.Data == nil {
			c.Set("response", "Invalid JSON object format. Expecting an object with a 'data' key.")
			c.Set("status", http.StatusBadRequest)
			return
		}
		bodies = []itemWriteBody{input}
	} else {
		c.Set("response", "Invalid or empty JSON body")
		c.Set("status", http.StatusBadRequest)
		return
	}

	inputs := make([]storage.ItemInput, 0, len(bodies))
	for i, b := range bodies {
		if b.Data == nil {
			c.Set("response", "Validation failed for an item in the batch")
			c.Set("details", gin.H{"item_index": i, "error": "missing 'data'"})
			c.Set("status", http.StatusBadRequest)
			return
		}
		publish, err := publishFlag(*ct, b.Status)
		if err != nil {
			if errors.Is(err, storage.ErrDraftAndPublishDisabled) {
				err = errors.New("collection '" + ct.Name + "' does not use draft and publish")
			}
			c.Set("response", err.Error())
			if isBatch {
				c.Set("details", gin.H{"item_index": i})
			}
			c.Set("status", http.StatusBadRequest)
			return
		}
		inputs = append(inputs, storage.ItemInput{Data: b.Data, Publish: publish != nil && *publish})
	}

	// All items are validated and stored in a single transaction.
	created, err := storage.CreateItems(actorFrom(c), *ct, inputs)
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

	out := make([]map[string]any, 0, len(created))
	for _, item := range created {
		resp, err := itemResponse(*ct, *item, rc)
		if err != nil {
			c.Set("response", "Failed to populate relations for response")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		out = append(out, resp)
	}
	if !isBatch {
		c.Set("response", out[0])
		c.Set("status", http.StatusCreated)
		return
	}
	c.Set("response", out)
	c.Set("meta", gin.H{"created_count": len(out)})
	c.Set("status", http.StatusCreated)
}

// readContext is what a read request may see.
type readContext struct {
	level         uint
	publishedOnly bool   // hide drafts (also in populated relations)
	listStatus    string // status filter for lists
	localeContext
}

// writeContext is the read context of a write's response: drafts included,
// in the request's locale.
func writeContext(c *gin.Context) (readContext, bool) {
	level, ok := parseLevel(c)
	if !ok {
		return readContext{}, false
	}
	lc, ok := parseLocale(c)
	if !ok {
		return readContext{}, false
	}
	return readContext{level: level, localeContext: lc}, true
}

// parseLevel reads the "level" query parameter (relation population depth,
// 0 to maxRelationLevel, default 1). It answers 400 and returns false when
// the value is invalid.
func parseLevel(c *gin.Context) (uint, bool) {
	levelParam := c.Query("level")
	if levelParam == "" {
		return 1, true
	}
	level, err := strconv.Atoi(levelParam)
	if err != nil || level < 0 || level > maxRelationLevel {
		c.Set("response", "Invalid level parameter (expected 0 to 5)")
		c.Set("status", http.StatusBadRequest)
		return 0, false
	}
	return uint(level), true
}

// parseReadContext resolves level and draft visibility. Drafts are only
// visible to roles that can update content; lists default to published
// items and accept status=draft|all, while fetching by ID returns drafts to
// those roles.
func parseReadContext(c *gin.Context, userRole string, ct *models.Collection, single bool) (readContext, bool) {
	level, ok := parseLevel(c)
	if !ok {
		return readContext{}, false
	}
	rc := readContext{level: level}
	if rc.localeContext, ok = parseLocale(c); !ok {
		return rc, false
	}
	canSeeDrafts := hasPermission(userRole, "update")
	status := c.Query("status")

	switch status {
	case "", models.StatusPublished, models.StatusDraft, "all":
	default:
		c.Set("response", "Invalid status parameter (expected published, draft or all)")
		c.Set("status", http.StatusBadRequest)
		return rc, false
	}
	if (status == models.StatusDraft || status == "all") && !canSeeDrafts {
		c.Set("response", "Drafts are only visible to roles that can edit content")
		c.Set("status", http.StatusForbidden)
		return rc, false
	}

	// What readers get by default is published content, including in
	// populated relations; editors opt into drafts with status=draft|all,
	// or by fetching an item by ID.
	if single {
		rc.publishedOnly = !canSeeDrafts || status == models.StatusPublished
	} else {
		rc.publishedOnly = !canSeeDrafts || status == "" || status == models.StatusPublished
	}
	if ct.DraftAndPublish {
		switch status {
		case "", models.StatusPublished:
			rc.listStatus = models.StatusPublished
		case models.StatusDraft:
			rc.listStatus = models.StatusDraft
		}
	}
	return rc, true
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

	// Handle fetching all items or a single item by ID
	if id == "" {
		rc, ok := parseReadContext(c, userRole, ct, false)
		if !ok {
			return
		}
		GetItems(*ct, rc)(c)
		return
	}
	itemID, err := strconv.Atoi(id)
	if err != nil || itemID < 1 {
		c.Set("response", "Invalid ID format")
		c.Set("status", http.StatusBadRequest)
		return
	}
	rc, ok := parseReadContext(c, userRole, ct, true)
	if !ok {
		return
	}
	GetItemByID(*ct, uint(itemID), rc)(c)
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
