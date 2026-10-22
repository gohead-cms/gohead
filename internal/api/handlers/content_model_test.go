package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/ratelimit"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contentAPI drives the real routes as admin, editor or viewer.
type contentAPI struct {
	t      *testing.T
	router *gin.Engine
}

func newContentAPI(t *testing.T) *contentAPI {
	t.Helper()
	router, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, migrations.MigrateDatabase(db))

	router.Use(middleware.ResponseWrapper())
	admin := router.Group("/admin", middleware.AuthMiddleware(), middleware.AdminOnly())
	admin.POST("/collections", CreateCollection)
	admin.PUT("/collections/:name", UpdateCollection)
	admin.POST("/components", CreateComponent)
	admin.DELETE("/components/:name", DeleteComponent)
	admin.POST("/singleton", CreateOrUpdateSingleton)
	admin.PUT("/singleton/:name", CreateOrUpdateSingleton)
	admin.POST("/locales", CreateLocale)
	admin.PUT("/locales/:code", UpdateLocale)
	admin.DELETE("/locales/:code", DeleteLocale)
	admin.GET("/roles", ListRoles)
	admin.POST("/roles", CreateRole)
	admin.PUT("/roles/:name", UpdateRole)
	admin.DELETE("/roles/:name", DeleteRole)
	admin.POST("/users", CreateUser)
	admin.POST("/agents", CreateAgent)
	admin.POST("/agents/:name/test", TestAgent)
	admin.GET("/agent-templates", ListAgentTemplates)
	admin.POST("/agent-templates/:id/render", RenderAgentTemplate)

	// A real limiter, as in production (rate limiting is on by default), so
	// the public-role tests go through it.
	api := router.Group("/api", middleware.PublicOrAuth(middleware.RateLimitAllow(ratelimit.NewMemory(), "public", 10_000, time.Minute)), middleware.DemoGuard())
	signedIn := middleware.SignedInOnly()
	api.Any("/collections/:collection", DynamicCollectionHandler)
	api.Any("/collections/:collection/:id", DynamicCollectionHandler)
	api.POST("/collections/:collection/:id/publish", PublishItem(true))
	api.POST("/collections/:collection/:id/unpublish", PublishItem(false))
	api.GET("/singleton/:name", GetSingleItem)
	api.PUT("/singleton/:name", CreateOrUpdateSingletonItem)
	api.GET("/collections/:collection/:id/revisions", ListItemRevisions)
	api.GET("/collections/:collection/:id/revisions/:version", GetItemRevision)
	api.POST("/collections/:collection/:id/revisions/:version/restore", RestoreItemRevision)
	api.GET("/singleton/:name/revisions", ListSingletonRevisions)
	api.GET("/singleton/:name/revisions/:version", GetSingletonRevision)
	api.POST("/singleton/:name/revisions/:version/restore", RestoreSingletonRevision)
	api.GET("/revisions", ListRevisions)
	api.GET("/proposals", ListProposals)
	api.GET("/proposals/:id", GetProposal)
	api.POST("/proposals/:id/approve", ApproveProposal)
	api.POST("/proposals/:id/reject", RejectProposal)
	api.GET("/locales", ListLocales)
	api.GET("/schema/collections", signedIn, GetCollections)
	api.GET("/schema/collections/:name", signedIn, GetCollection)
	api.GET("/schema/singletons", signedIn, GetSingletons)
	api.GET("/me", Me)
	return &contentAPI{t: t, router: router}
}

// do sends a request and returns the status and the decoded envelope.
func (a *contentAPI) do(role, method, path string, body any) (int, map[string]any) {
	a.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(a.t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if role != "" { // "": an anonymous request
		token, err := auth.GenerateJWT(role+"-user", role)
		require.NoError(a.t, err)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	a.router.ServeHTTP(rr, req)
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	return rr.Code, out
}

// ok asserts a 2xx status and returns "data".
func (a *contentAPI) ok(role, method, path string, body any) any {
	a.t.Helper()
	code, out := a.do(role, method, path, body)
	require.Truef(a.t, code >= 200 && code < 300, "%s %s as %s: %d %v", method, path, role, code, out)
	return out["data"]
}

func (a *contentAPI) errMsg(out map[string]any) string {
	e, _ := out["error"].(map[string]any)
	return fmt.Sprint(e["message"])
}

func ids(list any) []int {
	var out []int
	for _, it := range list.([]any) {
		out = append(out, int(it.(map[string]any)["id"].(float64)))
	}
	sort.Ints(out)
	return out
}

func keys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestOneResponseShape(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "authors", "kind": "collection",
		"attributes": map[string]any{"name": map[string]any{"type": "string", "required": true}},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "posts", "kind": "collection",
		"attributes": map[string]any{
			"title":  map[string]any{"type": "string", "required": true},
			"author": map[string]any{"type": "relation", "relation": "oneToMany", "target": "authors"},
			"tags":   map[string]any{"type": "relation", "relation": "manyToMany", "target": "authors"},
		},
	})
	author := a.ok("editor", "POST", "/api/collections/authors", map[string]any{"data": map[string]any{"name": "Ada"}}).(map[string]any)
	authorID := author["id"]

	created := a.ok("editor", "POST", "/api/collections/posts", map[string]any{
		"data": map[string]any{"title": "Hello", "author": authorID, "tags": []any{authorID}},
	}).(map[string]any)
	id := int(created["id"].(float64))
	got := a.ok("viewer", "GET", fmt.Sprintf("/api/collections/posts/%d", id), nil).(map[string]any)
	list := a.ok("viewer", "GET", "/api/collections/posts", nil).([]any)
	updated := a.ok("editor", "PUT", fmt.Sprintf("/api/collections/posts/%d", id), map[string]any{
		"data": map[string]any{"title": "Hello again", "author": authorID, "tags": []any{authorID}},
	}).(map[string]any)

	want := []string{"author", "createdAt", "id", "tags", "title", "updatedAt"}
	for name, item := range map[string]map[string]any{
		"create": created, "get": got, "list": list[0].(map[string]any), "update": updated,
	} {
		assert.Equal(t, want, keys(item), name)
		// oneToMany relations used to be dropped from single-item responses.
		rel, ok := item["author"].(map[string]any)
		require.True(t, ok, "%s: author populated", name)
		assert.Equal(t, "Ada", rel["name"], name)
		assert.Equal(t, authorID, rel["id"], name)
		assert.NotContains(t, rel, "attributes", name)
		assert.Len(t, item["tags"], 1, name)
	}
	assert.Equal(t, "Hello again", updated["title"])

	// level=0 leaves relation IDs in place.
	raw := a.ok("viewer", "GET", fmt.Sprintf("/api/collections/posts/%d?level=0", id), nil).(map[string]any)
	assert.Equal(t, authorID, raw["author"])

	// A deleted related item is left out rather than failing the read.
	a.ok("editor", "DELETE", fmt.Sprintf("/api/collections/authors/%d", int(authorID.(float64))), nil)
	got = a.ok("viewer", "GET", fmt.Sprintf("/api/collections/posts/%d", id), nil).(map[string]any)
	assert.Nil(t, got["author"])
	assert.Empty(t, got["tags"])

	// List errors use the standard error envelope.
	code, out := a.do("viewer", "GET", "/api/collections/posts?sort=nope", nil)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, a.errMsg(out), "unknown field 'nope'")
}

func TestUniqueValueSurvivesUpdate(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "users_dir", "kind": "collection",
		"attributes": map[string]any{
			"email": map[string]any{"type": "string", "unique": true},
			"bio":   map[string]any{"type": "text"},
		},
	})
	one := a.ok("editor", "POST", "/api/collections/users_dir", map[string]any{"data": map[string]any{"email": "a@x.io"}}).(map[string]any)
	a.ok("editor", "POST", "/api/collections/users_dir", map[string]any{"data": map[string]any{"email": "b@x.io"}})

	path := fmt.Sprintf("/api/collections/users_dir/%d", int(one["id"].(float64)))
	a.ok("editor", "PUT", path, map[string]any{"data": map[string]any{"email": "a@x.io", "bio": "kept my email"}})
	code, out := a.do("editor", "PUT", path, map[string]any{"data": map[string]any{"email": "b@x.io"}})
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, a.errMsg(out), "must be unique")
}

func TestDraftAndPublish(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "articles", "kind": "collection", "draft_and_publish": true,
		"attributes": map[string]any{"title": map[string]any{"type": "string", "required": true}},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "pages", "kind": "collection",
		"attributes": map[string]any{
			"title":    map[string]any{"type": "string"},
			"featured": map[string]any{"type": "relation", "relation": "oneToOne", "target": "articles"},
		},
	})

	draft := a.ok("editor", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{"title": "Draft"}}).(map[string]any)
	assert.Equal(t, "draft", draft["status"])
	assert.Nil(t, draft["publishedAt"])
	draftPath := fmt.Sprintf("/api/collections/articles/%d", int(draft["id"].(float64)))

	live := a.ok("editor", "POST", "/api/collections/articles", map[string]any{
		"data": map[string]any{"title": "Live"}, "status": "published",
	}).(map[string]any)
	assert.Equal(t, "published", live["status"])
	assert.NotNil(t, live["publishedAt"])
	liveID := int(live["id"].(float64))

	t.Run("viewers only see published items", func(t *testing.T) {
		assert.Equal(t, []int{liveID}, ids(a.ok("viewer", "GET", "/api/collections/articles", nil)))
		code, _ := a.do("viewer", "GET", draftPath, nil)
		assert.Equal(t, http.StatusNotFound, code)
		code, out := a.do("viewer", "GET", "/api/collections/articles?status=draft", nil)
		assert.Equal(t, http.StatusForbidden, code)
		assert.Contains(t, a.errMsg(out), "Drafts are only visible")
	})

	t.Run("editors opt into drafts", func(t *testing.T) {
		assert.Equal(t, []int{liveID}, ids(a.ok("editor", "GET", "/api/collections/articles", nil)), "lists default to published")
		assert.Len(t, a.ok("editor", "GET", "/api/collections/articles?status=all", nil), 2)
		drafts := a.ok("editor", "GET", "/api/collections/articles?status=draft", nil).([]any)
		require.Len(t, drafts, 1)
		assert.Equal(t, "Draft", drafts[0].(map[string]any)["title"])
		got := a.ok("editor", "GET", draftPath, nil).(map[string]any)
		assert.Equal(t, "draft", got["status"])
		code, _ := a.do("editor", "GET", draftPath+"?status=published", nil)
		assert.Equal(t, http.StatusNotFound, code)
		code, _ = a.do("editor", "GET", "/api/collections/articles?status=bogus", nil)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("publish and unpublish", func(t *testing.T) {
		code, _ := a.do("viewer", "POST", draftPath+"/publish", nil)
		assert.Equal(t, http.StatusForbidden, code)

		pub := a.ok("editor", "POST", draftPath+"/publish", nil).(map[string]any)
		assert.Equal(t, "published", pub["status"])
		first := pub["publishedAt"]
		again := a.ok("editor", "POST", draftPath+"/publish", nil).(map[string]any)
		assert.Equal(t, first, again["publishedAt"], "publishing twice keeps the date")
		assert.Len(t, a.ok("viewer", "GET", "/api/collections/articles", nil), 2)

		unpub := a.ok("editor", "POST", draftPath+"/unpublish", nil).(map[string]any)
		assert.Equal(t, "draft", unpub["status"])
		assert.Equal(t, []int{liveID}, ids(a.ok("viewer", "GET", "/api/collections/articles", nil)))

		code, _ = a.do("editor", "POST", "/api/collections/articles/9999/publish", nil)
		assert.Equal(t, http.StatusNotFound, code)
	})

	t.Run("updates keep or change the status", func(t *testing.T) {
		livePath := fmt.Sprintf("/api/collections/articles/%d", liveID)
		kept := a.ok("editor", "PUT", livePath, map[string]any{"data": map[string]any{"title": "Live 2"}}).(map[string]any)
		assert.Equal(t, "published", kept["status"])
		back := a.ok("editor", "PUT", livePath, map[string]any{"data": map[string]any{"title": "Live 3"}, "status": "draft"}).(map[string]any)
		assert.Equal(t, "draft", back["status"])
		code, out := a.do("editor", "PUT", livePath, map[string]any{"data": map[string]any{"title": "x"}, "status": "archived"})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, a.errMsg(out), "invalid status")
		a.ok("editor", "PUT", livePath, map[string]any{"data": map[string]any{"title": "Live"}, "status": "published"})
	})

	var pagePath string
	t.Run("drafts are hidden in relations", func(t *testing.T) {
		draftID := draft["id"]
		page := a.ok("editor", "POST", "/api/collections/pages", map[string]any{
			"data": map[string]any{"title": "Home", "featured": draftID},
		}).(map[string]any)
		pagePath = fmt.Sprintf("/api/collections/pages/%d", int(page["id"].(float64)))
		got := a.ok("viewer", "GET", pagePath, nil).(map[string]any)
		assert.Nil(t, got["featured"], "viewer does not see the draft")
		raw := a.ok("viewer", "GET", pagePath+"?level=0", nil).(map[string]any)
		assert.Nil(t, raw["featured"], "not even the draft's ID reaches viewers")
		raw = a.ok("editor", "GET", pagePath+"?level=0", nil).(map[string]any)
		assert.Equal(t, draftID, raw["featured"])
		got = a.ok("editor", "GET", pagePath, nil).(map[string]any)
		require.NotNil(t, got["featured"], "editor fetching by ID sees it")
		assert.Equal(t, "draft", got["featured"].(map[string]any)["status"])
		list := a.ok("editor", "GET", "/api/collections/pages", nil).([]any)
		assert.Nil(t, list[0].(map[string]any)["featured"], "default lists show published content only")
	})

	t.Run("singleton status=all keeps links to drafts for editors", func(t *testing.T) {
		draftID := draft["id"]
		a.ok("admin", "POST", "/admin/singleton", map[string]any{
			"name": "home",
			"attributes": map[string]any{
				"featured": map[string]any{"type": "relation", "relation": "oneToOne", "target": "articles"},
			},
		})
		a.ok("editor", "PUT", "/api/singleton/home", map[string]any{"data": map[string]any{"featured": draftID}})

		got := a.ok("editor", "GET", "/api/singleton/home?level=0", nil).(map[string]any)
		assert.Nil(t, got["featured"], "published only by default")
		got = a.ok("editor", "GET", "/api/singleton/home?level=0&status=all", nil).(map[string]any)
		assert.Equal(t, draftID, got["featured"])

		code, _ := a.do("viewer", "GET", "/api/singleton/home?status=all", nil)
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("editor", "GET", "/api/singleton/home?status=bogus", nil)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("status is rejected without draft and publish", func(t *testing.T) {
		code, out := a.do("editor", "POST", "/api/collections/pages", map[string]any{
			"data": map[string]any{"title": "x"}, "status": "published",
		})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, a.errMsg(out), "does not use draft and publish")
		code, _ = a.do("editor", "POST", pagePath+"/publish", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		page := a.ok("viewer", "GET", pagePath, nil).(map[string]any)
		assert.NotContains(t, page, "status")
		assert.NotContains(t, page, "publishedAt")
	})
}

func TestEnablingDraftAndPublishKeepsItemsVisible(t *testing.T) {
	a := newContentAPI(t)
	def := map[string]any{
		"name": "news", "kind": "collection",
		"attributes": map[string]any{"title": map[string]any{"type": "string"}},
	}
	a.ok("admin", "POST", "/admin/collections", def)
	a.ok("editor", "POST", "/api/collections/news", map[string]any{"data": map[string]any{"title": "Old"}})

	def["draft_and_publish"] = true
	a.ok("admin", "PUT", "/admin/collections/news", def)
	list := a.ok("viewer", "GET", "/api/collections/news", nil).([]any)
	require.Len(t, list, 1, "items that were visible stay visible")
	assert.Equal(t, "published", list[0].(map[string]any)["status"])

	d := a.ok("editor", "POST", "/api/collections/news", map[string]any{"data": map[string]any{"title": "New"}}).(map[string]any)
	assert.Equal(t, "draft", d["status"])

	t.Run("updates that leave the option out keep it", func(t *testing.T) {
		delete(def, "draft_and_publish")
		def["attributes"] = map[string]any{"title": map[string]any{"type": "string"}, "body": map[string]any{"type": "text"}}
		a.ok("admin", "PUT", "/admin/collections/news", def)
		assert.Len(t, a.ok("viewer", "GET", "/api/collections/news", nil), 1, "the draft stays hidden")

		// The shape GET /admin/collections returns is accepted too.
		def["options"] = map[string]any{"draftAndPublish": true}
		a.ok("admin", "PUT", "/admin/collections/news", def)
		assert.Len(t, a.ok("viewer", "GET", "/api/collections/news", nil), 1)
		delete(def, "options")
	})

	t.Run("turning the option off publishes drafts", func(t *testing.T) {
		def["draft_and_publish"] = false
		a.ok("admin", "PUT", "/admin/collections/news", def)
		assert.Len(t, a.ok("viewer", "GET", "/api/collections/news", nil), 2)
		def["draft_and_publish"] = true
		a.ok("admin", "PUT", "/admin/collections/news", def)
		list := a.ok("viewer", "GET", "/api/collections/news", nil).([]any)
		assert.Len(t, list, 2, "what was visible stays visible")
	})

	t.Run("status becomes a reserved name", func(t *testing.T) {
		def["attributes"] = map[string]any{"status": map[string]any{"type": "string"}}
		code, out := a.do("admin", "PUT", "/admin/collections/news", def)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, a.errMsg(out), "reserved")
		code, _ = a.do("admin", "POST", "/admin/collections", map[string]any{
			"name": "bad", "kind": "collection",
			"attributes": map[string]any{"createdAt": map[string]any{"type": "string"}},
		})
		assert.Equal(t, http.StatusBadRequest, code)
	})
}

func TestComponentValues(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/components", map[string]any{
		"name": "seo",
		"attributes": map[string]any{
			"metaTitle": map[string]any{"type": "string", "required": true},
			"keywords":  map[string]any{"type": "string"},
		},
	})
	a.ok("admin", "POST", "/admin/components", map[string]any{
		"name": "link",
		"attributes": map[string]any{
			"label": map[string]any{"type": "string", "required": true},
			"url":   map[string]any{"type": "string", "pattern": "^https://"},
			"seo":   map[string]any{"type": "component", "component": "seo"},
		},
	})

	code, out := a.do("admin", "POST", "/admin/collections", map[string]any{
		"name": "broken", "kind": "collection",
		"attributes": map[string]any{"x": map[string]any{"type": "component", "component": "nope"}},
	})
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, a.errMsg(out), "component 'nope'")

	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "landing", "kind": "collection",
		"attributes": map[string]any{
			"title": map[string]any{"type": "string"},
			"seo":   map[string]any{"type": "component", "component": "seo", "required": true},
			"links": map[string]any{"type": "component", "component": "link", "repeatable": true, "max": 2},
		},
	})

	item := a.ok("editor", "POST", "/api/collections/landing", map[string]any{"data": map[string]any{
		"title": "Home",
		"seo":   map[string]any{"metaTitle": "Welcome"},
		"links": []any{
			map[string]any{"label": "Docs", "url": "https://docs.example", "seo": map[string]any{"metaTitle": "Docs"}},
			map[string]any{"label": "Blog"},
		},
	}}).(map[string]any)
	assert.Equal(t, map[string]any{"metaTitle": "Welcome"}, item["seo"])
	links := item["links"].([]any)
	require.Len(t, links, 2)
	assert.Equal(t, "Docs", links[0].(map[string]any)["seo"].(map[string]any)["metaTitle"])

	got := a.ok("viewer", "GET", fmt.Sprintf("/api/collections/landing/%d", int(item["id"].(float64))), nil).(map[string]any)
	assert.Equal(t, item["links"], got["links"], "values round-trip through storage")

	bad := map[string]map[string]any{
		"missing required field":     {"seo": map[string]any{}},
		"unknown field":              {"seo": map[string]any{"metaTitle": "x", "extra": 1}},
		"not an object":              {"seo": "Welcome"},
		"repeatable needs an array":  {"seo": map[string]any{"metaTitle": "x"}, "links": map[string]any{"label": "x"}},
		"too many":                   {"seo": map[string]any{"metaTitle": "x"}, "links": []any{map[string]any{"label": "1"}, map[string]any{"label": "2"}, map[string]any{"label": "3"}}},
		"nested field fails pattern": {"seo": map[string]any{"metaTitle": "x"}, "links": []any{map[string]any{"label": "1", "url": "http://insecure"}}},
		"required component missing": {"title": "x"},
	}
	for name, data := range bad {
		code, out := a.do("editor", "POST", "/api/collections/landing", map[string]any{"data": data})
		assert.Equal(t, http.StatusBadRequest, code, "%s: %v", name, out)
	}

	t.Run("components in use cannot be deleted", func(t *testing.T) {
		code, out := a.do("admin", "DELETE", "/admin/components/seo", nil)
		assert.Equal(t, http.StatusConflict, code)
		assert.Contains(t, a.errMsg(out), "landing.seo")
		assert.Contains(t, a.errMsg(out), "link.seo")
	})

	t.Run("singletons accept components too", func(t *testing.T) {
		a.ok("admin", "POST", "/admin/singleton", map[string]any{
			"name": "homepage",
			"attributes": map[string]any{
				"headline": map[string]any{"type": "string"},
				"seo":      map[string]any{"type": "component", "component": "seo"},
			},
		})
		code, _ := a.do("viewer", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{"headline": "x"}})
		assert.Equal(t, http.StatusForbidden, code, "viewers cannot write singletons")

		code, out := a.do("editor", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{"seo": map[string]any{}}})
		assert.Equal(t, http.StatusBadRequest, code, "%v", out)

		created := a.ok("editor", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{
			"headline": "Hi", "seo": map[string]any{"metaTitle": "Home"},
		}}).(map[string]any)
		assert.Equal(t, []string{"createdAt", "headline", "id", "seo", "updatedAt"}, keys(created))
		got := a.ok("viewer", "GET", "/api/singleton/homepage", nil).(map[string]any)
		assert.Equal(t, "Home", got["seo"].(map[string]any)["metaTitle"])
	})
}
