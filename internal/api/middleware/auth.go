// internal/api/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortWithError(c, http.StatusUnauthorized, "", "Authorization header required")
			return
		}

		// Parse Bearer token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			abortWithError(c, http.StatusUnauthorized, "", "Bearer token required")
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
			abortWithError(c, http.StatusForbidden, "", "Invalid role")
			return
		}

		// Attach user details to the context
		c.Set("username", claims.Username)
		c.Set("role", role.Name)
		c.Next()
	}
}

// abortWithError aborts with the same error envelope ResponseWrapper uses, so
// clients can always read error.message. An empty name derives it from status.
func abortWithError(c *gin.Context, status int, name, message string) {
	if name == "" {
		name = getErrorName(status)
	}
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"status":  status,
			"name":    name,
			"message": message,
			"details": nil,
		},
	})
}
