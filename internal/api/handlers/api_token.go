package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// maxTokenLifetimeDays bounds expires_in_days (about 10 years).
const maxTokenLifetimeDays = 3650

// CreateAPIToken creates a token and returns its secret once:
// {"name": "website", "role": "viewer", "expires_in_days": 90}.
func CreateAPIToken(c *gin.Context) {
	var body struct {
		Name          string  `json:"name"`
		Description   string  `json:"description"`
		Role          string  `json:"role"`
		ExpiresInDays *int    `json:"expires_in_days"`
		ExpiresAt     *string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		validationError(c, "Invalid JSON body")
		return
	}
	if body.Role == "" {
		body.Role = "viewer"
	}
	var expires *time.Time
	switch {
	case body.ExpiresInDays != nil && body.ExpiresAt != nil:
		validationError(c, "Use expires_in_days or expires_at, not both")
		return
	case body.ExpiresInDays != nil:
		if *body.ExpiresInDays < 1 || *body.ExpiresInDays > maxTokenLifetimeDays {
			validationError(c, "expires_in_days must be between 1 and 3650")
			return
		}
		t := time.Now().Add(time.Duration(*body.ExpiresInDays) * 24 * time.Hour)
		expires = &t
	case body.ExpiresAt != nil:
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			validationError(c, "expires_at must be an RFC 3339 date-time")
			return
		}
		expires = &t
	}

	tok, secret, err := storage.CreateAPIToken(storage.NewAPITokenInput{
		Name: body.Name, Description: body.Description, Role: body.Role,
		ExpiresAt: expires, CreatedBy: c.GetString("username"),
	})
	if err != nil {
		validationError(c, err.Error())
		return
	}
	logger.Log.WithField("token", tok.Name).WithField("role", tok.Role).WithField("by", tok.CreatedBy).Info("API token created")
	out := models.APITokenResponse(*tok)
	out["token"] = secret
	c.Set("response", out)
	c.Set("meta", gin.H{"warning": "Store this token now: it cannot be shown again."})
	c.Set("status", http.StatusCreated)
}

// ListAPITokens lists tokens (never their secrets).
func ListAPITokens(c *gin.Context) {
	list, err := storage.ListAPITokens()
	if err != nil {
		c.Set("response", "Failed to list tokens")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, models.APITokenResponse(t))
	}
	c.Set("response", out)
	c.Set("meta", gin.H{"total": len(out)})
	c.Set("status", http.StatusOK)
}

func tokenID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		validationError(c, "Invalid token ID")
		return 0, false
	}
	return uint(id), true
}

func tokenError(c *gin.Context, err error) {
	if errors.Is(err, storage.ErrTokenNotFound) {
		c.Set("response", "Token not found")
		c.Set("status", http.StatusNotFound)
		return
	}
	c.Set("response", "Token operation failed")
	c.Set("status", http.StatusInternalServerError)
}

// GetAPIToken returns one token.
func GetAPIToken(c *gin.Context) {
	id, ok := tokenID(c)
	if !ok {
		return
	}
	t, err := storage.GetAPIToken(id)
	if err != nil {
		tokenError(c, err)
		return
	}
	c.Set("response", models.APITokenResponse(*t))
	c.Set("status", http.StatusOK)
}

// RevokeAPIToken revokes a token immediately.
func RevokeAPIToken(c *gin.Context) {
	id, ok := tokenID(c)
	if !ok {
		return
	}
	t, err := storage.RevokeAPIToken(id)
	if err != nil {
		tokenError(c, err)
		return
	}
	logger.Log.WithField("token", t.Name).WithField("by", c.GetString("username")).Info("API token revoked")
	c.Set("response", models.APITokenResponse(*t))
	c.Set("status", http.StatusOK)
}
