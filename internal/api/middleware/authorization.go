// internal/api/middleware/authorization.go
package middleware

import (
	"net/http"

	"gohead/pkg/logger"

	"github.com/gin-gonic/gin"
)

func AuthorizeRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found"})
			return
		}

		userRole := role.(string)
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user role from the request context
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			logger.Log.Warn("Unauthorized access attempt by non-admin user")
			c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can perform this action"})
			c.Abort()
			return
		}
		c.Next()
	}
}
