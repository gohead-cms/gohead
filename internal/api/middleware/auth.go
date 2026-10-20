// internal/api/middleware/auth.go
package middleware

import (
	"net/http"
	"strconv"
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

// AnonymousKey is set in the context of requests without credentials that
// act with the public role.
const AnonymousKey = "anonymous"

// PublicOrAuth authenticates like AuthMiddleware, except that a request
// without an Authorization header acts with the public role when that role
// grants anything. Anonymous requests go through limit (per client IP), and
// a 403 they get becomes a 401 so clients know signing in may help.
func PublicOrAuth(limit gin.HandlerFunc) gin.HandlerFunc {
	authenticated := AuthMiddleware()
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != "" || !storage.PublicAccessEnabled() {
			authenticated(c)
			return
		}
		if limit != nil {
			limit(c)
			if c.IsAborted() {
				return
			}
		}
		c.Set("username", "")
		c.Set("role", auth.RolePublic)
		c.Set(AnonymousKey, true)
		c.Writer.Header().Add("Vary", "Authorization")
		capLevel(c)
		c.Next()
		status := c.GetInt("status")
		// Forbidden, or a type that does not exist: the same answer, so
		// visitors cannot list the content model.
		unknownType := status == http.StatusNotFound && typeNotFound[c.GetString("response")]
		if status == http.StatusForbidden || unknownType {
			c.Set("status", http.StatusUnauthorized)
			c.Set("response", "Authentication required")
			c.Set("details", "anonymous visitors cannot do this: send a token")
		}
	}
}

// SignedInOnly refuses anonymous requests (public role) with 401, for
// routes only accounts and tokens may use whatever the public role grants.
func SignedInOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsAnonymous(c) {
			abortWithError(c, http.StatusUnauthorized, "", "Authorization header required")
			return
		}
		c.Next()
	}
}

// MaxAnonymousLevel is the deepest relation population anonymous requests
// get: each level multiplies the work.
const MaxAnonymousLevel = 2

// typeNotFound are the answers for unknown collections and singletons.
var typeNotFound = map[string]bool{"Collection not found": true, "Singleton not found": true, "Single type not found": true}

// capLevel lowers ?level= to MaxAnonymousLevel.
func capLevel(c *gin.Context) {
	q := c.Request.URL.Query()
	v := q.Get("level")
	if v == "" {
		return
	}
	if n, err := strconv.Atoi(v); err == nil && n > MaxAnonymousLevel {
		q.Set("level", strconv.Itoa(MaxAnonymousLevel))
		c.Request.URL.RawQuery = q.Encode()
	}
}

// IsAnonymous reports whether the request acts with the public role.
func IsAnonymous(c *gin.Context) bool { return c.GetBool(AnonymousKey) }

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
			if _, err := storage.AssignableRole(tok.Role); err != nil {
				abortWithError(c, http.StatusForbidden, "", "Invalid role")
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

		// The account's current role, so a role change applies to existing
		// sessions at once (the token's claim is used for accounts the
		// database does not know).
		roleName := claims.Role
		if user, err := storage.GetUserByUsername(claims.Username); err == nil && user.Role.Name != "" {
			roleName = user.Role.Name
		}
		role, err := storage.AssignableRole(roleName)
		if err != nil {
			logger.Log.Warnf("Role '%s' not found", roleName)
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
