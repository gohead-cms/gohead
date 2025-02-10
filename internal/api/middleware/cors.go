package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"gohead/pkg/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware applies dynamic CORS settings based on config
func CORSMiddleware(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		// Check if the origin is allowed
		allowed := false
		for _, o := range cfg.CORS.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if allowed {

			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.CORS.AllowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.CORS.AllowedHeaders, ", "))

			if cfg.CORS.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			c.Header("Access-Control-Max-Age", strconv.Itoa(cfg.CORS.MaxAge))
		}

		// Handle preflight OPTIONS request
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
