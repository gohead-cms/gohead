package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rolesFixture(t *testing.T) *contentAPI {
	t.Helper()
	a := newContentAPI(t)
	a.router.GET("/api/media", middleware.AuthMiddleware(), ListMedia)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "pages", "kind": "collection",
		"attributes": map[string]any{"title": map[string]any{"type": "string"}},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "articles", "kind": "collection", "draft_and_publish": true,
		"attributes": map[string]any{
			"title": map[string]any{"type": "string"},
			"page":  map[string]any{"type": "relation", "relation": "oneToOne", "target": "pages"},
		},
	})
	a.ok("admin", "POST", "/admin/singleton", map[string]any{
		"name": "homepage", "attributes": map[string]any{"headline": map[string]any{"type": "string"}},
	})
	a.ok("admin", "POST", "/admin/roles", map[string]any{
		"name": "writer", "description": "Writes articles, publishes nothing",
		"permissions": map[string]any{
			"content": map[string]any{"articles": []string{"create", "update"}, "singleton:homepage": []string{"read"}},
			"media":   []string{"read", "create"},
		},
	})
	a.ok("admin", "POST", "/admin/roles", map[string]any{
		"name": "reviewer",
		"permissions": map[string]any{
			"content": map[string]any{"*": []string{"read"}, "articles": []string{"publish"}},
		},
	})
	return a
}

func TestCustomRolesAdmin(t *testing.T) {
	a := rolesFixture(t)

	roles := a.ok("admin", "GET", "/admin/roles", nil).([]any)
	require.Len(t, roles, 6, "built-in, public and custom roles")
	assert.Equal(t, "admin", field(roles[0], "name"), "built-in roles first")
	assert.Equal(t, true, field(roles[0], "builtin"))
	var writer map[string]any
	for _, r := range roles {
		if field(r, "name") == "writer" {
			writer = r.(map[string]any)
		}
	}
	require.NotNil(t, writer)
	content := field(writer["permissions"], "content").(map[string]any)
	assert.Equal(t, []any{"read", "create", "update"}, content["articles"], "read is implied, actions are ordered")

	for _, bad := range []map[string]any{
		{"name": "x1", "permissions": map[string]any{"content": map[string]any{"nope": []string{"read"}}}},
		{"name": "x2", "permissions": map[string]any{"content": map[string]any{"pages": []string{"fly"}}}},
		{"name": "x3", "permissions": map[string]any{"content": map[string]any{"singleton:homepage": []string{"delete"}}}},
		{"name": "Bad Name"},
	} {
		code, out := a.do("admin", "POST", "/admin/roles", bad)
		assert.Equal(t, http.StatusBadRequest, code, "%v → %v", bad, out)
	}
	code, _ := a.do("admin", "POST", "/admin/roles", map[string]any{"name": "editor"})
	assert.Equal(t, http.StatusConflict, code)
	code, _ = a.do("admin", "PUT", "/admin/roles/viewer", map[string]any{"permissions": map[string]any{}})
	assert.Equal(t, http.StatusConflict, code, "built-in roles are fixed")
	code, _ = a.do("editor", "POST", "/admin/roles", map[string]any{"name": "x4"})
	assert.Equal(t, http.StatusForbidden, code)

	// A role in use cannot be deleted.
	a.ok("admin", "POST", "/admin/users", map[string]any{
		"username": "wanda", "email": "wanda@example.com", "password": "correct-horse-battery", "role_name": "writer",
	})
	code, out := a.do("admin", "DELETE", "/admin/roles/writer", nil)
	assert.Equal(t, http.StatusConflict, code)
	assert.Contains(t, a.errMsg(out), "1 user(s)")
	a.ok("admin", "DELETE", "/admin/roles/reviewer", nil)
}

func TestCustomRolesEnforced(t *testing.T) {
	a := rolesFixture(t)
	page := a.ok("admin", "POST", "/api/collections/pages", map[string]any{"data": map[string]any{"title": "About"}}).(map[string]any)

	t.Run("a writer drafts articles but cannot publish or delete", func(t *testing.T) {
		art := a.ok("writer", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{"title": "Hi", "page": page["id"]}}).(map[string]any)
		path := fmt.Sprintf("/api/collections/articles/%d", int(art["id"].(float64)))
		a.ok("writer", "PUT", path, map[string]any{"data": map[string]any{"title": "Hello", "page": page["id"]}})

		code, _ := a.do("writer", "POST", path+"/publish", nil)
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("writer", "PUT", path, map[string]any{"data": map[string]any{"title": "x"}, "status": "published"})
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("writer", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{"title": "x"}, "status": "published"})
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("writer", "DELETE", path, nil)
		assert.Equal(t, http.StatusForbidden, code)

		got := a.ok("writer", "GET", path, nil).(map[string]any)
		assert.Equal(t, "draft", got["status"], "editing gives access to drafts")
		assert.Nil(t, got["page"], "related entries of types the role cannot read are hidden")

		assert.Len(t, a.ok("writer", "GET", path+"/revisions", nil), 2)

		// The reviewer reads everything and publishes articles, nothing else.
		a.ok("reviewer", "POST", path+"/publish", nil)
		code, _ = a.do("reviewer", "PUT", path, map[string]any{"data": map[string]any{"title": "x"}})
		assert.Equal(t, http.StatusForbidden, code)
		reviewed := a.ok("reviewer", "GET", path+"?status=all", nil).(map[string]any)
		assert.Equal(t, "About", field(reviewed["page"], "title"))
	})

	t.Run("other types are out of reach", func(t *testing.T) {
		code, _ := a.do("writer", "GET", "/api/collections/pages", nil)
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("writer", "POST", "/api/collections/pages", map[string]any{"data": map[string]any{"title": "x"}})
		assert.Equal(t, http.StatusForbidden, code)
		code, out := a.do("writer", "POST", "/api/collections/articles", map[string]any{
			"data": map[string]any{"title": "x", "page": map[string]any{"title": "Sneaky"}},
		})
		assert.Equal(t, http.StatusForbidden, code, "relation objects would create a page: %v", out)
		a.ok("writer", "POST", "/api/collections/articles", map[string]any{
			"data": map[string]any{"title": "x", "page": map[string]any{"id": page["id"]}},
		})

		schemas := a.ok("writer", "GET", "/api/schema/collections", nil).([]any)
		require.Len(t, schemas, 1)
		assert.Equal(t, "articles", field(field(schemas[0], "schema"), "collectionName"))
		code, _ = a.do("writer", "GET", "/api/schema/collections/pages", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Len(t, a.ok("writer", "GET", "/api/schema/singletons", nil), 1)

		// The change feed only shows types the role edits.
		for _, r := range a.ok("writer", "GET", "/api/revisions", nil).([]any) {
			assert.Equal(t, "articles", field(r, "typeName"))
		}
		code, _ = a.do("writer", "GET", fmt.Sprintf("/api/collections/pages/%d/revisions", int(page["id"].(float64))), nil)
		assert.Equal(t, http.StatusForbidden, code)
	})

	t.Run("singletons and media", func(t *testing.T) {
		a.ok("admin", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{"headline": "Hi"}})
		a.ok("writer", "GET", "/api/singleton/homepage", nil)
		code, _ := a.do("writer", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{"headline": "x"}})
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("reviewer", "GET", "/api/media", nil)
		assert.Equal(t, http.StatusForbidden, code, "no media permission")
	})

	t.Run("permission changes apply at once", func(t *testing.T) {
		a.ok("admin", "PUT", "/admin/roles/writer", map[string]any{
			"permissions": map[string]any{"content": map[string]any{"articles": []string{"create", "update"}, "pages": []string{"read"}}},
		})
		assert.Len(t, a.ok("writer", "GET", "/api/collections/pages", nil), 1)
		a.ok("admin", "POST", "/admin/users", map[string]any{
			"username": "writer-user", "email": "w@example.com", "password": "correct-horse-battery", "role_name": "writer",
		})
		me := a.ok("writer", "GET", "/api/me", nil).(map[string]any)
		assert.Equal(t, []any{"read"}, field(field(me["permissions"], "content"), "pages"))
	})

	t.Run("built-in roles keep their rights", func(t *testing.T) {
		a.ok("editor", "POST", "/api/collections/pages", map[string]any{"data": map[string]any{"title": "x"}})
		code, _ := a.do("viewer", "POST", "/api/collections/pages", map[string]any{"data": map[string]any{"title": "x"}})
		assert.Equal(t, http.StatusForbidden, code)
		assert.NotEmpty(t, a.ok("viewer", "GET", "/api/collections/pages", nil))
	})
}

func TestCustomRolesReviewRules(t *testing.T) {
	a := newContentAPI(t)
	a.router.PUT("/admin/users/:id", middleware.AuthMiddleware(), middleware.AdminOnly(), UpdateUser)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "notes", "kind": "collection", "draft_and_publish": true,
		"attributes": map[string]any{"text": map[string]any{"type": "string"}},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "posts", "kind": "collection", "draft_and_publish": true,
		"attributes": map[string]any{
			"title": map[string]any{"type": "string"},
			"note":  map[string]any{"type": "relation", "relation": "oneToOne", "target": "notes"},
		},
	})
	a.ok("admin", "POST", "/admin/roles", map[string]any{
		"name": "author",
		"permissions": map[string]any{"content": map[string]any{
			"posts": []string{"create", "update", "delete"}, "notes": []string{"read"},
		}},
	})
	draftNote := a.ok("admin", "POST", "/api/collections/notes", map[string]any{"data": map[string]any{"text": "secret draft"}}).(map[string]any)

	t.Run("drafts of other types stay hidden through relations", func(t *testing.T) {
		post := a.ok("author", "POST", "/api/collections/posts?level=1", map[string]any{"data": map[string]any{"title": "x", "note": draftNote["id"]}}).(map[string]any)
		assert.Nil(t, post["note"], "the author cannot see drafts of notes")
		path := fmt.Sprintf("/api/collections/posts/%d", int(post["id"].(float64)))
		raw := a.ok("author", "GET", path+"?level=0&status=all", nil).(map[string]any)
		assert.Nil(t, raw["note"], "not even its ID")
		admin := a.ok("admin", "GET", path+"?level=0", nil).(map[string]any)
		assert.Equal(t, draftNote["id"], admin["note"])
	})

	t.Run("published content needs publish to change", func(t *testing.T) {
		live := a.ok("admin", "POST", "/api/collections/posts", map[string]any{"data": map[string]any{"title": "Live"}, "status": "published"}).(map[string]any)
		path := fmt.Sprintf("/api/collections/posts/%d", int(live["id"].(float64)))
		code, out := a.do("author", "PUT", path, map[string]any{"data": map[string]any{"title": "Defaced"}})
		assert.Equal(t, http.StatusForbidden, code, "%v", out)

		a.ok("author", "DELETE", path, nil)
		code, _ = a.do("author", "POST", path+"/revisions/1/restore", nil)
		assert.Equal(t, http.StatusForbidden, code, "restoring would publish it again")
		a.ok("admin", "POST", path+"/revisions/1/restore", nil)
	})

	t.Run("roles follow renamed and deleted types", func(t *testing.T) {
		a.ok("admin", "PUT", "/admin/collections/notes", map[string]any{
			"name": "memos", "kind": "collection", "draft_and_publish": true,
			"attributes": map[string]any{"text": map[string]any{"type": "string"}},
		})
		roles := a.ok("admin", "GET", "/admin/roles", nil).([]any)
		for _, r := range roles {
			if field(r, "name") == "author" {
				content := field(field(r, "permissions"), "content").(map[string]any)
				assert.Contains(t, content, "memos")
				assert.NotContains(t, content, "notes")
			}
		}
	})

	t.Run("a new role applies to existing sessions", func(t *testing.T) {
		a.ok("admin", "POST", "/admin/users", map[string]any{
			"username": "author-user", "email": "a@example.com", "password": "correct-horse-battery", "role_name": "author",
		})
		user, err := storage.GetUserByUsername("author-user")
		require.NoError(t, err)
		a.ok("author", "POST", "/api/collections/posts", map[string]any{"data": map[string]any{"title": "ok"}})
		a.ok("admin", "PUT", fmt.Sprintf("/admin/users/%d", user.ID), map[string]any{"role_name": "viewer"})
		code, _ := a.do("author", "POST", "/api/collections/posts", map[string]any{"data": map[string]any{"title": "no"}})
		assert.Equal(t, http.StatusForbidden, code, "the token still says author, the account is now a viewer")
	})
}
