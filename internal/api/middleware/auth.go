// internal/api/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			return
		}

		claims, err := auth.ParseJWT(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		logger.Log.WithFields(logrus.Fields{
			"username": claims.Username,
			"role":     claims.Role,
			"path":     c.Request.URL.Path,
			"method":   c.Request.Method,
		}).Info("Authenticated request")

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}
