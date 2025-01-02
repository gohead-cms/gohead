// internal/api/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// Parse Bearer token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			return
		}

		// Parse JWT and extract claims
		claims, err := auth.ParseJWT(tokenString)
		if err != nil {
			abortWithError(c, http.StatusUnauthorized, "InvalidTokenError", "Invalid token")
			return
		}

		// Retrieve the role using the storage abstraction
		role, err := storage.GetRoleByName(claims.Role)
		if err != nil {
			logger.Log.Warnf("Role '%s' not found", claims.Role)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid role"})
			return
		}

		// Attach user details to the context
		c.Set("username", claims.Username)
		c.Set("role", role.Name)
		c.Next()
	}
}

// Helper function to abort with a standardized error
func abortWithError(c *gin.Context, status int, name, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"status":  status,
			"name":    name,
			"message": message,
			"details": nil,
		},
	})
}
