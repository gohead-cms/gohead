// internal/api/middleware/authorization.go
package middleware

import (
	"net/http"

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
		if !exists || role != "admin" {
			logger.Log.Warn("Unauthorized access attempt by non-admin user")
			abortWithError(c, http.StatusForbidden, "", "Only admins can perform this action")
			return
		}
		c.Next()
	}
}
