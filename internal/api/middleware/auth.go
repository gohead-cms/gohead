// internal/api/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// APITokenIDKey is set in the context when a request authenticates with an
// API token rather than a user session.
const APITokenIDKey = "api_token_id"

// UserSessionOnly refuses requests made with an API token. It protects
// routes that would let a leaked token create more credentials (tokens,
// users).
func UserSessionOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, isToken := c.Get(APITokenIDKey); isToken {
			abortWithError(c, http.StatusForbidden, "", "This endpoint needs a user session; API tokens cannot manage credentials")
			return
		}
		c.Next()
	}
}

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

		// Long-lived API tokens (ght_...) act with their role.
		if strings.HasPrefix(tokenString, models.APITokenPrefix) {
			tok, err := storage.AuthenticateAPIToken(tokenString)
			if err != nil {
				abortWithError(c, http.StatusUnauthorized, "InvalidTokenError", "Invalid or revoked API token")
				return
			}
			c.Set("username", "token:"+tok.Name)
			c.Set("role", tok.Role)
			c.Set(APITokenIDKey, tok.ID)
			c.Next()
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
