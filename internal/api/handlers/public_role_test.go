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
		code, _ = a.do("admin", "PUT", "/admin/roles/public", map[string]any{"permissions": map[string]any{"media": []string{"read", "create"}}})
		assert.Equal(t, http.StatusBadRequest, code)
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
		} {
			code, out := a.do("", r.method, r.path, map[string]any{"data": map[string]any{"title": "x"}})
			assert.Equal(t, http.StatusUnauthorized, code, "%s %s → %v", r.method, r.path, out)
		}
	})
}

func TestIntrospectionPattern(t *testing.T) {
	assert.True(t, introspection.MatchString(`{ __schema { types { name } } }`))
	assert.True(t, introspection.MatchString(`{ __type(name: "Article") { name } }`))
	assert.False(t, introspection.MatchString(`{ articles { __typename title } }`))
}
