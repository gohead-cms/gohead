package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicRole(t *testing.T) {
	a := rolesFixture(t)
	page := a.ok("admin", "POST", "/api/collections/pages", map[string]any{"data": map[string]any{"title": "About"}}).(map[string]any)
	draft := a.ok("admin", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{"title": "Secret draft"}}).(map[string]any)
	live := a.ok("admin", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{"title": "Live", "page": page["id"]}, "status": "published"}).(map[string]any)
	a.ok("admin", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{"headline": "Welcome"}})

	t.Run("anonymous requests are refused until the public role grants something", func(t *testing.T) {
		code, out := a.do("", "GET", "/api/collections/articles", nil)
		assert.Equal(t, http.StatusUnauthorized, code, out)
		assert.Equal(t, "Authorization header required", a.errMsg(out))
	})

	t.Run("the public role can only read and stays", func(t *testing.T) {
		roles := a.ok("admin", "GET", "/admin/roles", nil).([]any)
		assert.Equal(t, "public", field(roles[3], "name"), "listed after the built-in roles")
		assert.Equal(t, true, field(roles[3], "public"))

		code, out := a.do("admin", "PUT", "/admin/roles/public", map[string]any{"permissions": map[string]any{"content": map[string]any{"articles": []string{"read", "create"}}}})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, a.errMsg(out), "can only read")
		for _, refused := range []map[string]any{
			{"media": []string{"read"}},
			{"content": map[string]any{"*": []string{"read"}}},
		} {
			code, out = a.do("admin", "PUT", "/admin/roles/public", map[string]any{"permissions": refused})
			assert.Equal(t, http.StatusBadRequest, code, "%v → %v", refused, out)
		}
		a.ok("admin", "POST", "/admin/collections", map[string]any{
			"name": "accounts", "kind": "collection",
			"attributes": map[string]any{"login": map[string]any{"type": "string"}, "secret": map[string]any{"type": "password"}},
		})
		code, out = a.do("admin", "PUT", "/admin/roles/public", map[string]any{"permissions": map[string]any{"content": map[string]any{"accounts": []string{"read"}}}})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, a.errMsg(out), "password")
		code, _ = a.do("admin", "DELETE", "/admin/roles/public", nil)
		assert.Equal(t, http.StatusConflict, code)
		code, _ = a.do("admin", "POST", "/admin/roles", map[string]any{"name": "public"})
		assert.Equal(t, http.StatusConflict, code)
		code, out = a.do("admin", "POST", "/admin/users", map[string]any{
			"username": "anon", "email": "anon@example.com", "password": "correct-horse-battery", "role_name": "public",
		})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, fmt.Sprint(field(out["error"], "details")), "anonymous")
		_, _, err := storage.CreateAPIToken(storage.NewAPITokenInput{Name: "anon", Role: "public"})
		assert.ErrorIs(t, err, storage.ErrPublicRole)

		// A session claiming the public role is not anonymous access.
		code, _ = a.do("public", "GET", "/api/collections/articles", nil)
		assert.Equal(t, http.StatusForbidden, code)

		a.ok("admin", "PUT", "/admin/roles/public", map[string]any{
			"description": "Website visitors",
			"permissions": map[string]any{"content": map[string]any{"articles": []string{"read"}, "singleton:homepage": []string{"read"}}},
		})
	})

	t.Run("visitors read published content they are granted", func(t *testing.T) {
		list := a.ok("", "GET", "/api/collections/articles", nil).([]any)
		require.Len(t, list, 1, "drafts are never public")
		assert.Equal(t, "Live", field(list[0], "title"))
		assert.Nil(t, field(list[0], "page"), "relations to types visitors cannot read are hidden")

		code, _ := a.do("", "GET", "/api/collections/articles?status=draft", nil)
		assert.Equal(t, http.StatusUnauthorized, code)
		code, _ = a.do("", "GET", fmt.Sprintf("/api/collections/articles/%v", draft["id"]), nil)
		assert.Contains(t, []int{http.StatusNotFound, http.StatusUnauthorized}, code)
		a.ok("", "GET", fmt.Sprintf("/api/collections/articles/%v", live["id"]), nil)
		assert.Equal(t, "Welcome", field(a.ok("", "GET", "/api/singleton/homepage", nil), "headline"))
		a.ok("", "GET", "/api/locales", nil)

		// Deep population is capped for visitors.
		deep := a.ok("", "GET", fmt.Sprintf("/api/collections/articles/%v?level=5", live["id"]), nil)
		assert.NotNil(t, deep)
	})

	t.Run("a type that gains a password attribute is closed again", func(t *testing.T) {
		a.ok("admin", "PUT", "/admin/singleton/homepage", map[string]any{
			"name": "homepage", "attributes": map[string]any{"headline": map[string]any{"type": "string"}, "pin": map[string]any{"type": "password"}},
		})
		code, _ := a.do("", "GET", "/api/singleton/homepage", nil)
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("everything else needs a token", func(t *testing.T) {
		for _, r := range []struct{ method, path string }{
			{"GET", "/api/collections/pages"},
			{"POST", "/api/collections/articles"},
			{"PUT", fmt.Sprintf("/api/collections/articles/%v", live["id"])},
			{"DELETE", fmt.Sprintf("/api/collections/articles/%v", live["id"])},
			{"PUT", "/api/singleton/homepage"},
			{"GET", "/api/schema/collections"},
			{"GET", "/api/revisions"},
			{"GET", "/admin/roles"},
			{"GET", "/api/collections/does-not-exist"},
			{"GET", "/api/singleton/does-not-exist"},
		} {
			code, out := a.do("", r.method, r.path, map[string]any{"data": map[string]any{"title": "x"}})
			assert.Equal(t, http.StatusUnauthorized, code, "%s %s → %v", r.method, r.path, out)
		}
	})
}

func TestAnonymousQueryShape(t *testing.T) {
	depth, fields, queries, err := queryShape(`query { articles { title author { name avatar { url } } } }`)
	require.NoError(t, err)
	assert.Equal(t, []any{4, 6, true}, []any{depth, fields, queries})
	depth, _, _, err = queryShape(`{ a { ...F } } fragment F on A { b { ...G } } fragment G on B { c { d } }`)
	require.NoError(t, err)
	assert.Equal(t, 4, depth, "fragments count")
	_, _, _, err = queryShape(`{ a { ...F } } fragment F on A { b { ...F } }`)
	require.NoError(t, err, "fragment cycles do not loop")
	_, _, queries, _ = queryShape(`mutation { createarticles(data: {}) { id } }`)
	assert.False(t, queries)
}

func TestIntrospectionPattern(t *testing.T) {
	assert.True(t, introspection.MatchString(`{ __schema { types { name } } }`))
	assert.True(t, introspection.MatchString(`{ __type(name: "Article") { name } }`))
	assert.False(t, introspection.MatchString(`{ articles { __typename title } }`))
}
