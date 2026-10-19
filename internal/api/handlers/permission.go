package handlers

import (
	"net/http"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"

	"github.com/gin-gonic/gin"
)

// hasPermission reports whether role may perform action on at least one
// content type. Content handlers check one type with can.
func hasPermission(role, action string) bool {
	return auth.HasPermission(role, action)
}

// can reports whether the caller may perform action on resource (a
// collection name, auth.SingletonResource(name) or auth.ResourceMedia).
func can(c *gin.Context, action, resource string) bool {
	return auth.Can(c.GetString("role"), action, resource)
}

// allow answers 403 and returns false when the caller may not perform
// action on resource.
func allow(c *gin.Context, action, resource string) bool {
	if can(c, action, resource) {
		return true
	}
	c.Set("response", "Access denied")
	c.Set("details", "your role cannot "+action+" "+describeResource(resource))
	c.Set("status", http.StatusForbidden)
	return false
}

func describeResource(resource string) string {
	if resource == auth.ResourceMedia {
		return "media"
	}
	if name, ok := auth.IsSingletonResource(resource); ok {
		return "the " + name + " singleton"
	}
	return "items of " + resource
}

// readableCollections keeps the collections the caller can read.
func readableCollections(c *gin.Context, collections []models.Collection) []models.Collection {
	out := make([]models.Collection, 0, len(collections))
	for _, ct := range collections {
		if can(c, auth.ActionRead, ct.Name) {
			out = append(out, ct)
		}
	}
	return out
}
