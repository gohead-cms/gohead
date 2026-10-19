package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func i18nFixture(t *testing.T) (*contentAPI, string) {
	t.Helper()
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/locales", map[string]any{"code": "fr", "name": "Français"})
	a.ok("admin", "POST", "/admin/locales", map[string]any{"code": "de"})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "articles", "kind": "collection",
		"attributes": map[string]any{
			"title":  map[string]any{"type": "string", "required": true, "localized": true},
			"slug":   map[string]any{"type": "string", "unique": true, "localized": true},
			"body":   map[string]any{"type": "text", "localized": true},
			"rating": map[string]any{"type": "int"},
		},
	})
	created := a.ok("editor", "POST", "/api/collections/articles", map[string]any{
		"data": map[string]any{"title": "Hello", "slug": "hello", "body": "World", "rating": 3},
	}).(map[string]any)
	return a, fmt.Sprintf("/api/collections/articles/%d", int(created["id"].(float64)))
}

func TestLocales(t *testing.T) {
	a := newContentAPI(t)
	list := a.ok("viewer", "GET", "/api/locales", nil).([]any)
	require.Len(t, list, 1, "the implicit default locale")
	assert.Equal(t, "en", field(list[0], "code"))
	assert.Equal(t, true, field(list[0], "isDefault"))

	a.ok("admin", "POST", "/admin/locales", map[string]any{"code": "pt-BR", "name": "Português (Brasil)"})
	list = a.ok("viewer", "GET", "/api/locales", nil).([]any)
	require.Len(t, list, 2)
	assert.Equal(t, "en", field(list[0], "code"), "the default comes first")

	code, _ := a.do("admin", "POST", "/admin/locales", map[string]any{"code": "pt-BR"})
	assert.Equal(t, http.StatusConflict, code)
	code, _ = a.do("admin", "POST", "/admin/locales", map[string]any{"code": "Not a code"})
	assert.Equal(t, http.StatusBadRequest, code)
	code, _ = a.do("editor", "POST", "/admin/locales", map[string]any{"code": "it"})
	assert.Equal(t, http.StatusForbidden, code)
	code, _ = a.do("admin", "DELETE", "/admin/locales/en", nil)
	assert.Equal(t, http.StatusConflict, code, "the default locale stays")

	// No content in pt-BR yet: it can become the default (a relabelling).
	l := a.ok("admin", "PUT", "/admin/locales/pt-BR", map[string]any{"isDefault": true, "name": "Português"}).(map[string]any)
	assert.Equal(t, true, l["isDefault"])
	assert.Equal(t, "Português", l["name"])
	a.ok("admin", "DELETE", "/admin/locales/en", nil)
	assert.Len(t, a.ok("viewer", "GET", "/api/locales", nil), 1)
}

func TestLocalizedContent(t *testing.T) {
	a, path := i18nFixture(t)

	t.Run("the default locale is the entry itself", func(t *testing.T) {
		got := a.ok("viewer", "GET", path, nil).(map[string]any)
		assert.Equal(t, "Hello", got["title"])
		assert.Equal(t, "en", got["locale"])
		assert.Equal(t, []any{"en"}, got["locales"])
	})

	t.Run("a translation holds localized attributes only", func(t *testing.T) {
		code, out := a.do("editor", "PUT", path+"?locale=fr", map[string]any{"data": map[string]any{"rating": 5}})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, a.errMsg(out), "'rating' is not localized")

		got := a.ok("editor", "PUT", path+"?locale=fr", map[string]any{
			"data": map[string]any{"title": "Bonjour", "slug": "bonjour"},
		}).(map[string]any)
		assert.Equal(t, "Bonjour", got["title"])
		assert.Equal(t, "fr", got["locale"])
		assert.Equal(t, []any{"en", "fr"}, got["locales"])
	})

	t.Run("reads fall back to the default locale", func(t *testing.T) {
		fr := a.ok("viewer", "GET", path+"?locale=fr", nil).(map[string]any)
		assert.Equal(t, "Bonjour", fr["title"])
		assert.Equal(t, "World", fr["body"], "not translated: the default value")
		assert.EqualValues(t, 3, fr["rating"], "shared attributes are the same in every locale")

		strict := a.ok("viewer", "GET", path+"?locale=fr&fallback=false", nil).(map[string]any)
		assert.NotContains(t, strict, "body")

		de := a.ok("viewer", "GET", path+"?locale=de", nil).(map[string]any)
		assert.Equal(t, "Hello", de["title"])
		assert.Equal(t, "de", de["locale"])

		code, _ := a.do("viewer", "GET", path+"?locale=xx", nil)
		assert.Equal(t, http.StatusBadRequest, code)
		code, _ = a.do("viewer", "GET", path+"?fallback=maybe", nil)
		assert.Equal(t, http.StatusBadRequest, code)

		// The default locale is unchanged.
		assert.Equal(t, "Hello", field(a.ok("viewer", "GET", path, nil), "title"))
	})

	t.Run("lists filter and sort in the locale", func(t *testing.T) {
		a.ok("editor", "POST", "/api/collections/articles", map[string]any{
			"data": map[string]any{"title": "Zebra", "slug": "zebra", "rating": 1},
		})
		list := a.ok("viewer", "GET", "/api/collections/articles?locale=fr&filters[title]=Bonjour", nil).([]any)
		require.Len(t, list, 1)
		assert.Equal(t, "Bonjour", field(list[0], "title"))
		assert.Empty(t, a.ok("viewer", "GET", "/api/collections/articles?locale=fr&filters[title]=Hello", nil))
		// Untranslated entries fall back: "Zebra" sorts after "Bonjour".
		sorted := a.ok("viewer", "GET", "/api/collections/articles?locale=fr&sort=title:desc", nil).([]any)
		assert.Equal(t, "Zebra", field(sorted[0], "title"))
		strict := a.ok("viewer", "GET", "/api/collections/articles?locale=fr&fallback=false&filters[title]=Zebra", nil).([]any)
		assert.Empty(t, strict, "without fallback only translated values match")
	})

	t.Run("unique values are unique per locale", func(t *testing.T) {
		other := a.ok("editor", "POST", "/api/collections/articles", map[string]any{
			"data": map[string]any{"title": "Other", "slug": "other"},
		}).(map[string]any)
		otherPath := fmt.Sprintf("/api/collections/articles/%d", int(other["id"].(float64)))
		code, out := a.do("editor", "PUT", otherPath+"?locale=fr", map[string]any{"data": map[string]any{"slug": "bonjour"}})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, a.errMsg(out), "unique in locale 'fr'")
		a.ok("editor", "PUT", otherPath+"?locale=de", map[string]any{"data": map[string]any{"slug": "bonjour"}})
	})

	t.Run("entries are created in the default locale", func(t *testing.T) {
		code, _ := a.do("editor", "POST", "/api/collections/articles?locale=fr", map[string]any{"data": map[string]any{"title": "x"}})
		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("translations are in the history", func(t *testing.T) {
		list := a.ok("editor", "GET", path+"/revisions", nil).([]any)
		require.Len(t, list, 2)
		rev := a.ok("editor", "GET", path+"/revisions/2", nil).(map[string]any)
		assert.Equal(t, "Bonjour", field(field(rev["translations"], "fr"), "title"))
		changes := rev["changes"].([]any)
		var names []string
		for _, ch := range changes {
			names = append(names, field(ch, "field").(string))
		}
		assert.Equal(t, []string{"slug@fr", "title@fr"}, names)

		// Restoring version 1 removes the translation.
		a.ok("editor", "POST", path+"/revisions/1/restore", nil)
		got := a.ok("viewer", "GET", path+"?locale=fr", nil).(map[string]any)
		assert.Equal(t, "Hello", got["title"])
		assert.Equal(t, []any{"en"}, got["locales"])
	})

	t.Run("an empty translation removes it; a null removes one value", func(t *testing.T) {
		a.ok("editor", "PUT", path+"?locale=fr", map[string]any{"data": map[string]any{"title": "Salut", "body": "Monde"}})
		a.ok("editor", "PUT", path+"?locale=fr", map[string]any{"data": map[string]any{"title": "Salut", "body": nil}})
		got := a.ok("viewer", "GET", path+"?locale=fr", nil).(map[string]any)
		assert.Equal(t, "World", got["body"])
		a.ok("editor", "PUT", path+"?locale=fr", map[string]any{"data": map[string]any{}})
		assert.Equal(t, []any{"en"}, field(a.ok("viewer", "GET", path, nil), "locales"))
	})

	t.Run("a locale with content is only deleted with purge", func(t *testing.T) {
		a.ok("editor", "PUT", path+"?locale=fr", map[string]any{"data": map[string]any{"title": "Salut"}})
		code, out := a.do("admin", "DELETE", "/admin/locales/fr", nil)
		require.Equal(t, http.StatusConflict, code, "%v", out)
		code, _ = a.do("admin", "PUT", "/admin/locales/fr", map[string]any{"isDefault": true})
		assert.Equal(t, http.StatusConflict, code, "it has content: making it the default would hide it")
		code, out = a.do("admin", "DELETE", "/admin/locales/fr?purge=true", nil)
		require.Equal(t, http.StatusOK, code, "%v", out)
		assert.EqualValues(t, 1, field(out["meta"], "purgedEntries"))
		code, _ = a.do("viewer", "GET", path+"?locale=fr", nil)
		assert.Equal(t, http.StatusBadRequest, code)
	})
}

func TestLocalizedSchema(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "authors", "kind": "collection",
		"attributes": map[string]any{"name": map[string]any{"type": "string"}},
	})
	code, out := a.do("admin", "POST", "/admin/collections", map[string]any{
		"name": "books", "kind": "collection",
		"attributes": map[string]any{
			"title":  map[string]any{"type": "string", "localized": true},
			"author": map[string]any{"type": "relation", "relation": "oneToOne", "target": "authors", "localized": true},
		},
	})
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, a.errMsg(out), "relations cannot be localized")
	code, out = a.do("admin", "POST", "/admin/collections", map[string]any{
		"name": "books", "kind": "collection",
		"attributes": map[string]any{
			"title":  map[string]any{"type": "string", "localized": true},
			"locale": map[string]any{"type": "string"},
		},
	})
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, a.errMsg(out), "reserved")
}

func TestLocalizedSingleton(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/locales", map[string]any{"code": "fr"})
	a.ok("admin", "POST", "/admin/singleton", map[string]any{
		"name": "homepage",
		"attributes": map[string]any{
			"headline": map[string]any{"type": "string", "localized": true},
			"theme":    map[string]any{"type": "string"},
		},
	})
	code, _ := a.do("editor", "PUT", "/api/singleton/homepage?locale=fr", map[string]any{"data": map[string]any{"headline": "Bienvenue"}})
	assert.Equal(t, http.StatusNotFound, code, "content is created in the default locale first")

	a.ok("editor", "PUT", "/api/singleton/homepage", map[string]any{"data": map[string]any{"headline": "Welcome", "theme": "dark"}})
	fr := a.ok("editor", "PUT", "/api/singleton/homepage?locale=fr", map[string]any{"data": map[string]any{"headline": "Bienvenue"}}).(map[string]any)
	assert.Equal(t, "Bienvenue", fr["headline"])
	assert.Equal(t, "dark", fr["theme"])

	assert.Equal(t, "Welcome", field(a.ok("viewer", "GET", "/api/singleton/homepage", nil), "headline"))
	assert.Equal(t, "Bienvenue", field(a.ok("viewer", "GET", "/api/singleton/homepage?locale=fr", nil), "headline"))
	code, _ = a.do("editor", "PUT", "/api/singleton/homepage?locale=fr", map[string]any{"data": map[string]any{"theme": "light"}})
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Len(t, a.ok("editor", "GET", "/api/singleton/homepage/revisions", nil), 2)
}

func TestLocalizedRelationsAndRestores(t *testing.T) {
	a := newContentAPI(t)
	a.ok("admin", "POST", "/admin/locales", map[string]any{"code": "fr"})
	a.ok("admin", "POST", "/admin/locales", map[string]any{"code": "de"})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "articles", "kind": "collection",
		"attributes": map[string]any{
			"title": map[string]any{"type": "string", "localized": true},
			"note":  map[string]any{"type": "string", "localized": true},
		},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "menus", "kind": "collection",
		"attributes": map[string]any{
			"label":   map[string]any{"type": "string"},
			"article": map[string]any{"type": "relation", "relation": "oneToOne", "target": "articles"},
		},
	})
	art := a.ok("editor", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{"title": "Hello"}}).(map[string]any)
	artPath := fmt.Sprintf("/api/collections/articles/%d", int(art["id"].(float64)))
	a.ok("editor", "PUT", artPath+"?locale=fr", map[string]any{"data": map[string]any{"title": "Bonjour"}})
	menu := a.ok("editor", "POST", "/api/collections/menus", map[string]any{"data": map[string]any{"label": "Main", "article": art["id"]}}).(map[string]any)
	menuPath := fmt.Sprintf("/api/collections/menus/%d", int(menu["id"].(float64)))

	t.Run("related entries are read in the same locale", func(t *testing.T) {
		got := a.ok("viewer", "GET", menuPath+"?locale=fr", nil).(map[string]any)
		related := got["article"].(map[string]any)
		assert.Equal(t, "Bonjour", related["title"])
		assert.Equal(t, "fr", related["locale"])
		got = a.ok("viewer", "GET", menuPath, nil).(map[string]any)
		assert.Equal(t, "Hello", field(got["article"], "title"))
	})

	t.Run("an unchanged translation records nothing", func(t *testing.T) {
		before := len(a.ok("editor", "GET", artPath+"/revisions", nil).([]any))
		a.ok("editor", "PUT", artPath+"?locale=de", map[string]any{"data": map[string]any{}})
		a.ok("editor", "PUT", artPath+"?locale=fr", map[string]any{"data": map[string]any{"title": "Bonjour"}})
		assert.Len(t, a.ok("editor", "GET", artPath+"/revisions", nil), before)
	})

	t.Run("restores skip deleted locales", func(t *testing.T) {
		a.ok("editor", "PUT", artPath+"?locale=de", map[string]any{"data": map[string]any{"title": "Hallo"}})
		versions := a.ok("editor", "GET", artPath+"/revisions", nil).([]any)
		withDe := int(field(versions[0], "version").(float64))
		a.ok("admin", "DELETE", "/admin/locales/de?purge=true", nil)
		code, out := a.do("editor", "POST", fmt.Sprintf("%s/revisions/%d/restore", artPath, withDe), nil)
		require.Equal(t, http.StatusOK, code, "%v", out)
		assert.Equal(t, []any{"en", "fr"}, field(out["data"], "locales"))
	})

	t.Run("values of attributes no longer localized do not block locales", func(t *testing.T) {
		a.ok("admin", "POST", "/admin/locales", map[string]any{"code": "it"})
		a.ok("editor", "PUT", artPath+"?locale=it", map[string]any{"data": map[string]any{"note": "nota"}})
		a.ok("admin", "PUT", "/admin/collections/articles", map[string]any{
			"name": "articles", "kind": "collection",
			"attributes": map[string]any{
				"title": map[string]any{"type": "string", "localized": true},
				"note":  map[string]any{"type": "string"},
			},
		})
		got := a.ok("viewer", "GET", artPath, nil).(map[string]any)
		assert.Equal(t, []any{"en", "fr"}, got["locales"], "the Italian note is not served any more")
		a.ok("admin", "DELETE", "/admin/locales/it", nil)
	})
}
