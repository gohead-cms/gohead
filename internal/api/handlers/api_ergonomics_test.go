package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/webhooks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookAdmin(t *testing.T) {
	a := newContentAPI(t)
	admin := a.router.Group("/admin", middleware.AuthMiddleware(), middleware.AdminOnly())
	admin.POST("/webhooks", CreateWebhook)
	admin.GET("/webhooks", ListWebhooks)
	admin.GET("/webhooks/:id", GetWebhook)
	admin.PUT("/webhooks/:id", UpdateWebhook)
	admin.DELETE("/webhooks/:id", DeleteWebhook)
	admin.GET("/webhooks/:id/deliveries", ListWebhookDeliveries)
	admin.POST("/webhooks/:id/test", TestWebhook)
	webhooks.Configure(webhooks.Config{AllowPrivateNetworks: true, Timeout: 2 * time.Second})
	t.Cleanup(func() { webhooks.Configure(webhooks.Config{}) })

	var mu sync.Mutex
	var bodies []string
	var sigs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		sigs = append(sigs, r.Header.Get("X-Gohead-Signature"))
		mu.Unlock()
	}))
	defer srv.Close()

	created := a.ok("admin", "POST", "/admin/webhooks", map[string]any{
		"name": "rebuild", "url": srv.URL, "events": []any{"item:published"}, "collections": []any{"articles"},
		"headers": map[string]any{"X-Api-Key": "s3cret-key"},
	}).(map[string]any)
	secret := created["secret"].(string)
	assert.True(t, strings.HasPrefix(secret, "whsec_"))
	assert.Equal(t, true, created["enabled"])
	assert.Equal(t, "json", created["format"])
	assert.Equal(t, []any{"X-Api-Key"}, created["headers"], "header names only")
	path := fmt.Sprintf("/admin/webhooks/%v", created["id"])

	got := a.ok("admin", "GET", path, nil).(map[string]any)
	assert.NotContains(t, got, "secret", "the secret is shown once")
	assert.NotContains(t, fmt.Sprint(a.ok("admin", "GET", "/admin/webhooks", nil)), "s3cret-key", "header values are never returned")

	res := a.ok("admin", "POST", path+"/test", nil).(map[string]any)
	assert.Equal(t, true, res["delivered"], "%v", res)
	assert.EqualValues(t, 200, res["statusCode"])
	mu.Lock()
	require.Len(t, bodies, 1)
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &body))
	assert.Equal(t, "webhook:test", body["event"])
	assert.NotEmpty(t, sigs[0])
	mu.Unlock()
	deliveries := a.ok("admin", "GET", path+"/deliveries", nil).([]any)
	require.Len(t, deliveries, 1)
	assert.EqualValues(t, 200, deliveries[0].(map[string]any)["statusCode"])

	rotated := a.ok("admin", "PUT", path, map[string]any{"rotate_secret": true, "format": "slack"}).(map[string]any)
	assert.NotEqual(t, secret, rotated["secret"])
	assert.Equal(t, "slack", rotated["format"])
	assert.Equal(t, []any{"item:published"}, rotated["events"], "fields left out are kept")
	unsigned := a.ok("admin", "PUT", path, map[string]any{"signed": false}).(map[string]any)
	assert.Equal(t, false, unsigned["signed"])

	for name, b := range map[string]map[string]any{
		"no name":          {"url": srv.URL},
		"bad url":          {"name": "x", "url": "ftp://x"},
		"unknown event":    {"name": "x", "url": srv.URL, "events": []any{"item:exploded"}},
		"bad format":       {"name": "x", "url": srv.URL, "format": "xml"},
		"reserved header":  {"name": "x", "url": srv.URL, "headers": map[string]any{"X-Gohead-Event": "spoof"}},
		"creds in the URL": {"name": "x", "url": "https://u:p@example.com"},
	} {
		code, _ := a.do("admin", "POST", "/admin/webhooks", b)
		assert.Equal(t, http.StatusBadRequest, code, name)
	}
	code, _ := a.do("editor", "GET", "/admin/webhooks", nil)
	assert.Equal(t, http.StatusForbidden, code)

	a.ok("admin", "DELETE", path, nil)
	code, _ = a.do("admin", "GET", path, nil)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestOpenAPIDocument(t *testing.T) {
	a := newContentAPI(t)
	a.router.GET("/api/openapi.json", middleware.AuthMiddleware(), OpenAPISpec)
	a.router.GET("/api/docs", APIDocs(false))

	a.ok("admin", "POST", "/admin/components", map[string]any{
		"name": "seo", "attributes": map[string]any{"metaTitle": map[string]any{"type": "string", "required": true}},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "authors", "kind": "collection",
		"attributes": map[string]any{"name": map[string]any{"type": "string", "required": true}},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "articles", "kind": "collection", "draft_and_publish": true, "description": "Blog posts",
		"attributes": map[string]any{
			"title":    map[string]any{"type": "string", "required": true},
			"views":    map[string]any{"type": "int", "min": 0},
			"category": map[string]any{"type": "enum", "options": []any{"news", "tutorial"}},
			"author":   map[string]any{"type": "relation", "relation": "oneToOne", "target": "authors"},
			"tags":     map[string]any{"type": "relation", "relation": "manyToMany", "target": "authors"},
			"cover":    map[string]any{"type": "media"},
			"seo":      map[string]any{"type": "component", "component": "seo"},
		},
	})

	code, _ := a.doWith("", "GET", "/api/openapi.json", nil)
	assert.Equal(t, http.StatusUnauthorized, code, "not public by default")

	code, doc := a.do("viewer", "GET", "/api/openapi.json", nil)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "3.1.0", doc["openapi"])
	assert.NotContains(t, doc, "data", "the document is not wrapped")

	paths := doc["paths"].(map[string]any)
	for _, p := range []string{
		"/api/collections/articles", "/api/collections/articles/{id}",
		"/api/collections/articles/{id}/publish", "/api/collections/authors/{id}",
		"/api/media", "/auth/login", "/api/graphql",
	} {
		assert.Contains(t, paths, p)
	}
	assert.NotContains(t, paths, "/api/collections/authors/{id}/publish", "no publish without draft and publish")

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	art := schemas["articles"].(map[string]any)
	props := art["properties"].(map[string]any)
	assert.Equal(t, "Blog posts", art["description"])
	assert.Equal(t, map[string]any{"type": "integer", "minimum": float64(0)}, props["views"])
	assert.Equal(t, []any{"news", "tutorial"}, props["category"].(map[string]any)["enum"])
	assert.Equal(t, map[string]any{"$ref": "#/components/schemas/Component_seo"}, props["seo"])
	assert.Contains(t, fmt.Sprint(props["author"]), "#/components/schemas/authors")
	assert.Equal(t, "array", props["tags"].(map[string]any)["type"])
	assert.ElementsMatch(t, []any{"id", "createdAt", "updatedAt", "status", "publishedAt"}, art["required"])
	assert.Equal(t, []any{"title"}, schemas["articlesInput"].(map[string]any)["required"])
	assert.Contains(t, schemas, "Component_seo")
	assert.Contains(t, schemas, "Media")

	rr := httptest.NewRecorder()
	a.router.ServeHTTP(rr, httptest.NewRequest("GET", "/api/docs", nil))
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "swagger-ui-bundle.js")
	assert.Contains(t, rr.Body.String(), `integrity="sha384-`)
	assert.Contains(t, rr.Header().Get("Content-Security-Policy"), "default-src 'self'")
}
