package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTokenRoutes adds the token admin routes the way the server does.
func withTokenRoutes(a *contentAPI) {
	admin := a.router.Group("/admin", middleware.AuthMiddleware(), middleware.AdminOnly())
	creds := admin.Group("", middleware.UserSessionOnly())
	creds.POST("/tokens", CreateAPIToken)
	creds.GET("/tokens", ListAPITokens)
	creds.GET("/tokens/:id", GetAPIToken)
	creds.DELETE("/tokens/:id", RevokeAPIToken)
	creds.GET("/users", GetAllUsers)
}

// doWith sends a request with a raw bearer token.
func (a *contentAPI) doWith(token, method, path string, body any) (int, map[string]any) {
	a.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(a.t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	a.router.ServeHTTP(rr, req)
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	return rr.Code, out
}

func TestAPITokens(t *testing.T) {
	a := newContentAPI(t)
	withTokenRoutes(a)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "notes", "kind": "collection",
		"attributes": map[string]any{"title": map[string]any{"type": "string"}},
	})
	a.ok("editor", "POST", "/api/collections/notes", map[string]any{"data": map[string]any{"title": "hello"}})

	created := a.ok("admin", "POST", "/admin/tokens", map[string]any{"name": "website", "expires_in_days": 30}).(map[string]any)
	readToken := created["token"].(string)
	assert.Regexp(t, `^ght_[0-9a-f]{64}$`, readToken)
	assert.Equal(t, "viewer", created["role"], "tokens are read-only by default")
	assert.Equal(t, readToken[:10], created["prefix"])
	assert.NotNil(t, created["expiresAt"])
	assert.Equal(t, "admin-user", created["createdBy"])

	var stored models.APIToken
	require.NoError(t, database.DB.First(&stored, created["id"]).Error)
	assert.NotContains(t, stored.Hash, readToken, "only a hash is stored")

	t.Run("a read token reads but cannot write", func(t *testing.T) {
		code, out := a.doWith(readToken, "GET", "/api/collections/notes", nil)
		assert.Equal(t, http.StatusOK, code, "%v", out)
		code, _ = a.doWith(readToken, "POST", "/api/collections/notes", map[string]any{"data": map[string]any{"title": "x"}})
		assert.Equal(t, http.StatusForbidden, code)
	})

	writeToken := a.ok("admin", "POST", "/admin/tokens", map[string]any{"name": "ci", "role": "editor"}).(map[string]any)["token"].(string)
	t.Run("an editor token writes", func(t *testing.T) {
		code, out := a.doWith(writeToken, "POST", "/api/collections/notes", map[string]any{"data": map[string]any{"title": "from CI"}})
		assert.Equal(t, http.StatusCreated, code, "%v", out)
	})

	t.Run("tokens cannot manage credentials", func(t *testing.T) {
		adminToken := a.ok("admin", "POST", "/admin/tokens", map[string]any{"name": "ops", "role": "admin"}).(map[string]any)["token"].(string)
		code, out := a.doWith(adminToken, "POST", "/admin/tokens", map[string]any{"name": "sneaky", "role": "admin"})
		assert.Equal(t, http.StatusForbidden, code)
		assert.Contains(t, a.errMsg(out), "user session")
		code, _ = a.doWith(adminToken, "GET", "/admin/users", nil)
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.doWith(adminToken, "POST", "/admin/collections", map[string]any{
			"name": "fromtoken", "kind": "collection", "attributes": map[string]any{"x": map[string]any{"type": "string"}},
		})
		assert.Equal(t, http.StatusCreated, code, "admin tokens can still manage the schema (CI)")
	})

	t.Run("listing never shows secrets", func(t *testing.T) {
		list := a.ok("admin", "GET", "/admin/tokens", nil).([]any)
		require.GreaterOrEqual(t, len(list), 3)
		for _, it := range list {
			m := it.(map[string]any)
			assert.NotContains(t, m, "token")
			assert.NotContains(t, fmt.Sprint(m), readToken[10:])
		}
		got := a.ok("admin", "GET", fmt.Sprintf("/admin/tokens/%v", created["id"]), nil).(map[string]any)
		assert.NotNil(t, got["lastUsedAt"], "use is recorded")
		assert.Equal(t, "active", got["status"])
	})

	t.Run("revoked and expired tokens are refused", func(t *testing.T) {
		revoked := a.ok("admin", "DELETE", fmt.Sprintf("/admin/tokens/%v", created["id"]), nil).(map[string]any)
		assert.Equal(t, "revoked", revoked["status"])
		code, out := a.doWith(readToken, "GET", "/api/collections/notes", nil)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Contains(t, a.errMsg(out), "revoked")

		short := a.ok("admin", "POST", "/admin/tokens", map[string]any{"name": "short", "expires_at": time.Now().Add(time.Hour).Format(time.RFC3339)}).(map[string]any)
		require.NoError(t, database.DB.Model(&models.APIToken{}).Where("id = ?", short["id"]).
			Update("expires_at", time.Now().Add(-time.Minute)).Error)
		code, _ = a.doWith(short["token"].(string), "GET", "/api/collections/notes", nil)
		assert.Equal(t, http.StatusUnauthorized, code)

		code, _ = a.doWith("ght_"+strings.Repeat("0", 64), "GET", "/api/collections/notes", nil)
		assert.Equal(t, http.StatusUnauthorized, code, "unknown tokens")
	})

	t.Run("validation", func(t *testing.T) {
		for name, body := range map[string]map[string]any{
			"no name":        {"role": "viewer"},
			"unknown role":   {"name": "x", "role": "root"},
			"both expiries":  {"name": "x", "expires_in_days": 1, "expires_at": "2030-01-01T00:00:00Z"},
			"expiry in past": {"name": "x", "expires_at": "2001-01-01T00:00:00Z"},
			"too long":       {"name": "x", "expires_in_days": 99999},
			"bad expires_at": {"name": "x", "expires_at": "tomorrow"},
		} {
			code, _ := a.do("admin", "POST", "/admin/tokens", body)
			assert.Equal(t, http.StatusBadRequest, code, name)
		}
		code, _ := a.do("editor", "GET", "/admin/tokens", nil)
		assert.Equal(t, http.StatusForbidden, code, "admins only")
	})
}
