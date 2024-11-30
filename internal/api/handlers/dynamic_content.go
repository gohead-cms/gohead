// internal/api/handlers/dynamic_content.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

func DynamicContentHandler(c *gin.Context) {
	contentTypeName := c.Param("contentType")
	id := c.Param("id")

	// Retrieve the ContentType
	ct, exists := storage.GetContentType(contentTypeName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Content type not found"})
		return
	}

	switch c.Request.Method {
	case http.MethodPost:
		CreateContentItem(ct)(c)
	case http.MethodGet:
		if id == "" {
			GetContentItems(ct)(c)
		} else {
			GetContentItemByID(ct)(c)
		}
	case http.MethodPut:
		if id != "" {
			UpdateContentItem(ct)(c)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required for update"})
		}
	case http.MethodDelete:
		if id != "" {
			DeleteContentItem(ct)(c)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required for deletion"})
		}
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}
