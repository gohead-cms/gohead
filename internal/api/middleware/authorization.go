// internal/api/middleware/authorization.go
package middleware

import (
	"net/http"

	"github.com/gohead-cms/gohead/internal/demo"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
)

func AuthorizeRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			abortWithError(c, http.StatusForbidden, "", "insufficient permissions")
			return
		}

		userRole := role.(string)
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		abortWithError(c, http.StatusForbidden, "", "insufficient permissions")
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user role from the request context
		role, exists := c.Get("role")
		if r, _ := role.(string); demo.IsDemo(r) {
			if demoAllowed(c) {
				c.Next()
				return
			}
			abortWithError(c, http.StatusForbidden, "DemoError", "This is a demo: you can look at everything, edit drafts and try agents with test runs, but not change settings.")
			return
		}
		if !exists || role != "admin" {
			logger.Log.Warn("Unauthorized access attempt by non-admin user")
			abortWithError(c, http.StatusForbidden, "", "Only admins can perform this action")
			return
		}
		c.Next()
	}
}

// demoAllowed reports whether the demo account may make an admin request:
// any read, and test runs of agents.
func demoAllowed(c *gin.Context) bool {
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return c.Request.Method == http.MethodPost && c.FullPath() == "/admin/agents/:name/test"
}
