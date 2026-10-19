package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/accounts"
	"github.com/gohead-cms/gohead/internal/api/middleware"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUIEndpoints(t *testing.T) {
	a := newContentAPI(t)
	a.router.GET("/auth/me", middleware.AuthMiddleware(), Me)
	admin := a.router.Group("/admin", middleware.AuthMiddleware(), middleware.AdminOnly())
	admin.GET("/overview", GetOverview)
	admin.GET("/runs", ListAllAgentRuns)
	admin.GET("/agent-functions", AgentFunctionCatalog)
	admin.POST("/agents", CreateAgent)
	admin.POST("/agents/:name/run", RunAgentNow)

	_, err := accounts.CreateAdmin("admin-user", "admin@example.com", "a-long-admin-password")
	require.NoError(t, err)
	me := a.ok("admin", "GET", "/auth/me", nil).(map[string]any)
	assert.Equal(t, "admin-user", me["username"])
	assert.Equal(t, "admin", me["role"])
	assert.Equal(t, "user", me["kind"])
	assert.NotContains(t, me, "password")

	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "posts", "kind": "collection",
		"attributes": map[string]any{"title": map[string]any{"type": "string"}},
	})
	a.ok("editor", "POST", "/api/collections/posts", map[string]any{"data": map[string]any{"title": "Hello"}})
	a.ok("admin", "POST", "/admin/agents", map[string]any{
		"name": "tagger", "system_prompt": "Tag.", "max_turns": 2,
		"llm_config": map[string]any{"provider": "openai", "model": "m", "api_key": "k"},
		"memory":     map[string]any{"type": "in-memory", "session_scope": "agent"},
		"trigger":    map[string]any{"type": "manual"},
	})
	agent, err := storage.GetAgentByName("tagger")
	require.NoError(t, err)
	for _, status := range []string{agents.RunSucceeded, agents.RunFailed} {
		require.NoError(t, storage.CreateAgentRun(&agents.AgentRun{AgentID: agent.ID, AgentName: "tagger",
			Status: status, StartedAt: time.Now().UTC(), InputTokens: 10, OutputTokens: 5}))
	}

	ov := a.ok("admin", "GET", "/admin/overview", nil).(map[string]any)["overview"].(map[string]any)
	counts := ov["counts"].(map[string]any)
	assert.EqualValues(t, 1, counts["collections"])
	assert.EqualValues(t, 1, counts["items"])
	assert.EqualValues(t, 1, counts["agents"])
	runs := ov["runs24h"].(map[string]any)
	assert.EqualValues(t, 2, runs["total"])
	assert.EqualValues(t, 1, runs["failed"])
	assert.EqualValues(t, 30, runs["inputTokens"].(float64)+runs["outputTokens"].(float64))
	recent := ov["recentItems"].([]any)
	require.Len(t, recent, 1)
	assert.Equal(t, "Hello", recent[0].(map[string]any)["title"])
	assert.Equal(t, "posts", recent[0].(map[string]any)["collection"])

	failed := a.ok("admin", "GET", "/admin/runs?status=failed", nil).([]any)
	require.Len(t, failed, 1)
	assert.Equal(t, "tagger", failed[0].(map[string]any)["agentName"])
	code, _ := a.do("admin", "GET", "/admin/runs?status=nope", nil)
	assert.Equal(t, http.StatusBadRequest, code)

	catalog := a.ok("admin", "GET", "/admin/agent-functions", nil).([]any)
	assert.NotEmpty(t, catalog)
	first := catalog[0].(map[string]any)
	assert.NotEmpty(t, first["description"])
	assert.Equal(t, "object", first["parameters"].(map[string]any)["type"])

	// Without Redis there is no queue: the API says so instead of failing.
	code, _ = a.do("admin", "POST", "/admin/agents/tagger/run", map[string]any{"input": "go"})
	assert.Equal(t, http.StatusServiceUnavailable, code)
	code, _ = a.do("admin", "POST", "/admin/agents/nobody/run", nil)
	assert.Equal(t, http.StatusNotFound, code)

	code, _ = a.do("editor", "GET", "/admin/overview", nil)
	assert.Equal(t, http.StatusForbidden, code)

	// Every signed-in user can read the content model; only admins change it.
	a.router.GET("/api/schema/collections", middleware.AuthMiddleware(), GetCollections)
	a.router.GET("/api/schema/collections/:name", middleware.AuthMiddleware(), GetCollection)
	for _, role := range []string{"editor", "viewer"} {
		list := a.ok(role, "GET", "/api/schema/collections", nil).([]any)
		assert.Len(t, list, 1, role)
		one := a.ok(role, "GET", "/api/schema/collections/posts", nil).(map[string]any)
		assert.Contains(t, one, "schema")
	}
}

// A schema read from the API and sent back unchanged keeps every setting
// (the admin UI edits collections this way: PUT replaces the definition).
func TestCollectionSchemaRoundTrip(t *testing.T) {
	a := newContentAPI(t)
	a.router.GET("/admin/collections/:name", middleware.AuthMiddleware(), middleware.AdminOnly(), GetCollection)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "authors", "kind": "collection",
		"attributes": map[string]any{"name": map[string]any{"type": "string"}},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "posts", "kind": "collection", "description": "Blog posts", "draft_and_publish": true,
		"attributes": map[string]any{
			"title":    map[string]any{"type": "string", "required": true, "unique": true, "pattern": "^[A-Z]"},
			"views":    map[string]any{"type": "int", "min": 0, "max": 10},
			"category": map[string]any{"type": "enum", "options": []string{"news", "tips"}},
			"author":   map[string]any{"type": "relation", "relation": "oneToOne", "target": "authors"},
		},
	})

	read := func() map[string]any {
		return a.ok("admin", "GET", "/admin/collections/posts", nil).(map[string]any)["schema"].(map[string]any)
	}
	before := read()
	a.ok("admin", "PUT", "/admin/collections/posts", map[string]any{
		"name": "posts", "kind": "collection",
		"description": before["description"], "options": before["options"], "attributes": before["attributes"],
	})
	after := read()
	assert.Equal(t, before, after, "nothing lost or changed by the round trip")

	attrs := after["attributes"].(map[string]any)
	title := attrs["title"].(map[string]any)
	assert.Equal(t, true, title["unique"])
	assert.Equal(t, "^[A-Z]", title["pattern"])
	assert.EqualValues(t, 10, attrs["views"].(map[string]any)["max"])
	assert.Len(t, attrs["category"].(map[string]any)["options"], 2)
	assert.Equal(t, "Blog posts", after["description"])

	// The relation still points at the collection (stored without the api:: prefix).
	a.ok("editor", "POST", "/api/collections/authors", map[string]any{"data": map[string]any{"name": "Ada"}})
	a.ok("editor", "POST", "/api/collections/posts", map[string]any{"data": map[string]any{"title": "Hello", "author": 1}})
}

func TestAdminsCannotLockThemselvesOut(t *testing.T) {
	a := newContentAPI(t)
	users := a.router.Group("/admin/users", middleware.AuthMiddleware(), middleware.AdminOnly())
	users.POST("", CreateUser)
	users.PUT("/:id", UpdateUser)
	users.DELETE("/:id", DeleteUser)

	me, err := accounts.CreateAdmin("admin-user", "admin@example.com", "a-long-admin-password")
	require.NoError(t, err)
	path := func(id any) string { return "/admin/users/" + fmt.Sprint(id) }

	code, _ := a.do("admin", "PUT", path(me.ID), map[string]any{"role_name": "editor"})
	assert.Equal(t, http.StatusConflict, code, "no self-demotion")

	a.ok("admin", "POST", "/admin/users", map[string]any{"username": "alice", "email": "alice@example.com", "password": "alice-password", "role_name": "admin"})
	alice, err := storage.GetUserByUsername("alice")
	require.NoError(t, err)
	a.ok("admin", "PUT", path(alice.ID), map[string]any{"role_name": "editor"}) // two admins: fine

	// Another admin session cannot remove the last admin either.
	asOther := func(method, target string, body any) (int, string) {
		var buf bytes.Buffer
		if body != nil {
			require.NoError(t, json.NewEncoder(&buf).Encode(body))
		}
		req := httptest.NewRequest(method, target, &buf)
		req.Header.Set("Content-Type", "application/json")
		token, err := auth.GenerateJWT("other-admin", "admin")
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, req)
		return rr.Code, rr.Body.String()
	}
	code, body := asOther("PUT", path(me.ID), map[string]any{"role_name": "viewer"})
	assert.Equal(t, http.StatusConflict, code)
	assert.Contains(t, body, "last admin")
	code, body = asOther("DELETE", path(me.ID), nil)
	assert.Equal(t, http.StatusConflict, code)
	assert.Contains(t, body, "last admin")
}
