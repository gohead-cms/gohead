package handlers

import (
	"net/http"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/storage"

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

// allowNested answers 403 and returns false when data would look up or
// create items (relations given as objects) in a collection the caller
// cannot read and create in.
func allowNested(c *gin.Context, ct models.Collection, data map[string]any) bool {
	targets, err := storage.NestedWriteTargets(ct, data)
	if err != nil {
		return true // validation reports unknown targets
	}
	for _, name := range targets {
		if !can(c, auth.ActionCreate, name) {
			c.Set("response", "Access denied")
			c.Set("details", "relation values given as objects create items in "+name+", which your role cannot do; link existing items by ID instead")
			c.Set("status", http.StatusForbidden)
			return false
		}
	}
	return true
}

// allowLiveEdit answers 403 and returns false when the item is published
// in a draft-and-publish collection and the caller cannot publish there:
// editing it would change live content without review.
func allowLiveEdit(c *gin.Context, ct models.Collection, itemID uint) bool {
	if !ct.DraftAndPublish || can(c, auth.ActionPublish, ct.Name) {
		return true
	}
	item, err := storage.GetItemByID(ct.ID, itemID)
	if err != nil || item.PublishedAt == nil {
		return true // not found is reported by the update; drafts are fine
	}
	c.Set("response", "Access denied")
	c.Set("details", "this item is published: changing it changes live content, which needs the publish permission; ask someone who can publish, or unpublish it first")
	c.Set("status", http.StatusForbidden)
	return false
}
