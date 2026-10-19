package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// field returns m[key] for a JSON object.
func field(v any, key string) any { return v.(map[string]any)[key] }

func actions(list any) []string {
	var out []string
	for _, r := range list.([]any) {
		out = append(out, field(r, "action").(string))
	}
	return out
}

func TestContentHistory(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "articles", "kind": "collection", "draft_and_publish": true,
		"attributes": map[string]any{
			"title": map[string]any{"type": "string", "required": true},
			"body":  map[string]any{"type": "text"},
		},
	})
	created := a.ok("editor", "POST", "/api/collections/articles", map[string]any{
		"data": map[string]any{"title": "First", "body": "Hello"},
	}).(map[string]any)
	id := int(created["id"].(float64))
	path := fmt.Sprintf("/api/collections/articles/%d", id)
	history := path + "/revisions"

	a.ok("editor", "PUT", path, map[string]any{"data": map[string]any{"title": "Second", "body": "Hello"}})
	a.ok("editor", "PUT", path, map[string]any{"data": map[string]any{"title": "Second", "body": "Hello"}})
	a.ok("editor", "POST", path+"/publish", nil)

	t.Run("every change is a revision, newest first", func(t *testing.T) {
		code, out := a.do("editor", "GET", history, nil)
		require.Equal(t, http.StatusOK, code, "%v", out)
		list := out["data"].([]any)
		assert.Equal(t, []string{"publish", "update", "create"}, actions(list), "a save that changes nothing adds no revision")
		latest := list[0].(map[string]any)
		assert.EqualValues(t, 3, latest["version"])
		assert.Equal(t, "user", latest["actorKind"])
		assert.Equal(t, "editor-user", latest["actorName"])
		assert.Equal(t, "published", latest["status"])
		assert.NotContains(t, latest, "data", "lists leave the data out")
		assert.EqualValues(t, 3, field(field(out["meta"], "pagination"), "total"))
	})

	t.Run("a revision shows its data and what changed", func(t *testing.T) {
		rev := a.ok("editor", "GET", history+"/2", nil).(map[string]any)
		assert.Equal(t, "Second", field(rev["data"], "title"))
		assert.EqualValues(t, 1, rev["comparedTo"])
		changes := rev["changes"].([]any)
		require.Len(t, changes, 1)
		assert.Equal(t, map[string]any{"field": "title", "change": "changed", "before": "First", "after": "Second"}, changes[0])

		first := a.ok("editor", "GET", history+"/1", nil).(map[string]any)
		assert.Nil(t, first["comparedTo"])
		assert.Len(t, first["changes"], 2, "the first version is all additions")

		vsCurrent := a.ok("editor", "GET", history+"/1?compare=current", nil).(map[string]any)
		assert.EqualValues(t, 3, vsCurrent["comparedTo"])
		assert.Len(t, vsCurrent["changes"], 1)

		code, _ := a.do("editor", "GET", history+"/9", nil)
		assert.Equal(t, http.StatusNotFound, code)
		code, _ = a.do("editor", "GET", history+"/1?compare=nope", nil)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("history needs the editor role", func(t *testing.T) {
		code, _ := a.do("viewer", "GET", history, nil)
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("viewer", "POST", history+"/1/restore", nil)
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("viewer", "GET", "/api/revisions", nil)
		assert.Equal(t, http.StatusForbidden, code)
	})

	t.Run("restoring makes an old version current, as a new revision", func(t *testing.T) {
		code, out := a.do("editor", "POST", history+"/1/restore", nil)
		require.Equal(t, http.StatusOK, code, "%v", out)
		assert.Equal(t, "First", field(out["data"], "title"))
		assert.Equal(t, "published", field(out["data"], "status"), "the publication state is kept")
		rev := field(out["meta"], "revision")
		assert.Equal(t, "restore", field(rev, "action"))
		assert.EqualValues(t, 4, field(rev, "version"))
		assert.EqualValues(t, 1, field(rev, "restoredFrom"))
		assert.Equal(t, false, field(out["meta"], "undeleted"))
		assert.Equal(t, "First", field(a.ok("viewer", "GET", path, nil), "title"))
	})

	t.Run("a deleted item can be brought back", func(t *testing.T) {
		a.ok("editor", "DELETE", path, nil)
		code, _ := a.do("editor", "GET", path, nil)
		assert.Equal(t, http.StatusNotFound, code)

		list := a.ok("editor", "GET", history, nil)
		assert.Equal(t, "delete", actions(list)[0], "the history survives the delete")
		deleted := a.ok("editor", "GET", "/api/revisions?collection=articles&action=delete", nil).([]any)
		require.Len(t, deleted, 1)
		assert.EqualValues(t, id, field(deleted[0], "entityId"))

		code, out := a.do("editor", "POST", history+"/2/restore", nil)
		require.Equal(t, http.StatusOK, code, "%v", out)
		assert.Equal(t, true, field(out["meta"], "undeleted"))
		assert.Equal(t, "draft", field(out["data"], "status"), "a restored item starts as a draft")
		got := a.ok("editor", "GET", path, nil)
		assert.Equal(t, "Second", field(got, "title"))
	})

	t.Run("attributes the collection lost are dropped", func(t *testing.T) {
		a.ok("admin", "PUT", "/admin/collections/articles", map[string]any{
			"name": "articles", "kind": "collection", "draft_and_publish": true,
			"attributes": map[string]any{"title": map[string]any{"type": "string", "required": true}},
		})
		a.ok("editor", "PUT", path, map[string]any{"data": map[string]any{"title": "Third"}})
		code, out := a.do("editor", "POST", history+"/1/restore", nil)
		require.Equal(t, http.StatusOK, code, "%v", out)
		assert.Equal(t, []any{"body"}, field(out["meta"], "dropped"))
		assert.Equal(t, "First", field(out["data"], "title"))
		assert.NotContains(t, out["data"], "body")
	})

	t.Run("the change feed filters", func(t *testing.T) {
		list := a.ok("editor", "GET", "/api/revisions?collection=articles&actorKind=user&pageSize=2", nil).([]any)
		assert.Len(t, list, 2)
		code, _ := a.do("editor", "GET", "/api/revisions?action=nope", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		code, _ = a.do("editor", "GET", "/api/revisions?collection=a&singleton=b", nil)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("unknown entries", func(t *testing.T) {
		code, _ := a.do("editor", "GET", "/api/collections/articles/9999/revisions", nil)
		assert.Equal(t, http.StatusNotFound, code)
		code, _ = a.do("editor", "GET", "/api/collections/nope/1/revisions", nil)
		assert.Equal(t, http.StatusNotFound, code)
		code, _ = a.do("editor", "POST", "/api/collections/articles/9999/revisions/1/restore", nil)
		assert.Equal(t, http.StatusNotFound, code)
	})
}

func TestSingletonHistory(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/singleton", map[string]any{
		"name":       "homepage",
		"attributes": map[string]any{"headline": map[string]any{"type": "string"}},
	})
	a.ok("editor", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{"headline": "One"}})
	a.ok("editor", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{"headline": "Two"}})

	list := a.ok("editor", "GET", "/api/singleton/homepage/revisions", nil)
	assert.Equal(t, []string{"update", "create"}, actions(list))
	rev := a.ok("editor", "GET", "/api/singleton/homepage/revisions/2", nil).(map[string]any)
	assert.Len(t, rev["changes"], 1)

	restored := a.ok("editor", "POST", "/api/singleton/homepage/revisions/1/restore", nil)
	assert.Equal(t, "One", field(restored, "headline"))
	assert.Equal(t, "One", field(a.ok("viewer", "GET", "/api/singleton/homepage", nil), "headline"))
	assert.Len(t, a.ok("editor", "GET", "/api/revisions?singleton=homepage", nil), 3)

	code, _ := a.do("editor", "GET", "/api/singleton/nope/revisions", nil)
	assert.Equal(t, http.StatusNotFound, code)
}

func TestHistoryHidesPasswordsAndValidatesPaging(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "members", "kind": "collection",
		"attributes": map[string]any{
			"name":   map[string]any{"type": "string"},
			"secret": map[string]any{"type": "password"},
		},
	})
	m := a.ok("editor", "POST", "/api/collections/members", map[string]any{
		"data": map[string]any{"name": "Ada", "secret": "hunter2hunter2"},
	}).(map[string]any)
	history := fmt.Sprintf("/api/collections/members/%d/revisions", int(m["id"].(float64)))

	rev := a.ok("editor", "GET", history+"/1", nil).(map[string]any)
	assert.Equal(t, "********", field(rev["data"], "secret"))
	for _, ch := range rev["changes"].([]any) {
		if field(ch, "field") == "secret" {
			assert.Equal(t, "********", field(ch, "after"))
		}
	}

	code, _ := a.do("editor", "GET", history+"?pageSize=500", nil)
	assert.Equal(t, http.StatusBadRequest, code)
	code, _ = a.do("editor", "GET", "/api/revisions?page=0", nil)
	assert.Equal(t, http.StatusBadRequest, code)

	code, out := a.do("editor", "POST", history+"/1/restore", nil)
	require.Equal(t, http.StatusOK, code, "%v", out)
	assert.Equal(t, true, field(out["meta"], "unchanged"), "restoring the current content records nothing")
	assert.Len(t, a.ok("editor", "GET", history, nil), 1)
}
