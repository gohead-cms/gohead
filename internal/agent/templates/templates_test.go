package templates

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/agent/functions"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T) {
	t.Helper()
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.Collection{}, &models.Attribute{}, &models.Locale{}))
	require.NoError(t, db.Create(&models.Collection{Name: "posts", Kind: "collection", Attributes: []models.Attribute{
		{Name: "title", Type: "string", Localized: true},
		{Name: "body", Type: "richtext", Localized: true},
		{Name: "summary", Type: "text"},
		{Name: "tags", Type: "json"},
		{Name: "flagged", Type: "bool"},
		{Name: "review", Type: "enum", Options: []string{"ok", "flagged", "spam"}},
		{Name: "views", Type: "int"},
	}}).Error)
	require.NoError(t, db.Create(&models.Locale{Code: "en", Name: "English", IsDefault: true}).Error)
	require.NoError(t, db.Create(&models.Locale{Code: "fr", Name: "Français"}).Error)
}

func functionsOf(t *testing.T, def map[string]any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, f := range def["functions"].([]any) {
		spec := f.(map[string]any)
		out[spec["name"].(string)] = spec
		assert.True(t, functions.IsImplementationAvailable(spec["impl_key"].(string)), spec["impl_key"])
	}
	return out
}

func TestRenderTemplates(t *testing.T) {
	fixture(t)
	require.Len(t, List(), 4)

	def, err := Render("summarizer", map[string]any{"collection": "posts", "source": "body", "target": "summary", "words": 25.0})
	require.NoError(t, err)
	assert.Equal(t, "summarizer-posts", def["name"])
	assert.Contains(t, def["system_prompt"], "at most 25 words")
	assert.Equal(t, false, def["require_approval"])
	fns := functionsOf(t, def)
	assert.Equal(t, []string{"posts"}, fns["save_summary"]["collections"], "scoped to the collection")
	assert.Equal(t, "collections.patch_item", fns["save_summary"]["impl_key"])
	trigger := def["trigger"].(map[string]any)["event_trigger"].(map[string]any)
	assert.Equal(t, []string{"item:created", "item:updated"}, trigger["events"])

	def, err = Render("tagger", map[string]any{"collection": "posts", "sources": []any{"title", "body"}, "target": "tags",
		"tags": "news, sport", "name": "my-tagger", "approval": true})
	require.NoError(t, err)
	assert.Equal(t, "my-tagger", def["name"])
	assert.Equal(t, true, def["require_approval"])
	assert.Contains(t, def["system_prompt"], "JSON array")
	assert.Contains(t, def["system_prompt"], "news, sport")

	def, err = Render("moderator", map[string]any{"collection": "posts", "sources": []any{"body"}, "flag": "flagged"})
	require.NoError(t, err)
	assert.Contains(t, def["system_prompt"], `{"flagged": true when`)
	_, err = Render("moderator", map[string]any{"collection": "posts", "sources": []any{"body"}, "flag": "review", "flagged_value": "bad"})
	assert.ErrorContains(t, err, "'bad' is not an option")

	def, err = Render("translator", map[string]any{"collection": "posts", "fields": []any{"title", "body"}, "locales": "fr"})
	require.NoError(t, err)
	assert.Equal(t, true, def["require_approval"], "translations wait for review by default")
	assert.Contains(t, functionsOf(t, def), "list_locales")
	trigger = def["trigger"].(map[string]any)["event_trigger"].(map[string]any)
	assert.Equal(t, []string{"item:created"}, trigger["events"])
}

func TestRenderValidatesParams(t *testing.T) {
	fixture(t)
	for name, tc := range map[string]struct {
		id     string
		params map[string]any
		err    string
	}{
		"unknown collection":   {"summarizer", map[string]any{"collection": "nope", "source": "body", "target": "summary"}, "does not exist"},
		"missing field":        {"summarizer", map[string]any{"collection": "posts", "target": "summary"}, "is required"},
		"unknown field":        {"summarizer", map[string]any{"collection": "posts", "source": "nope", "target": "summary"}, "not an attribute"},
		"wrong type":           {"summarizer", map[string]any{"collection": "posts", "source": "views", "target": "summary"}, "int attribute"},
		"same field":           {"summarizer", map[string]any{"collection": "posts", "source": "summary", "target": "summary"}, "own field"},
		"not localized":        {"translator", map[string]any{"collection": "posts", "fields": []any{"summary"}}, "not localized"},
		"unknown locale":       {"translator", map[string]any{"collection": "posts", "fields": []any{"title"}, "locales": "de"}, "not configured"},
		"tags into the source": {"tagger", map[string]any{"collection": "posts", "sources": []any{"summary"}, "target": "summary"}, "own field"},
		"default locale":       {"translator", map[string]any{"collection": "posts", "fields": []any{"title"}, "locales": "en, fr"}, "default locale"},
		"string bool":          {"summarizer", map[string]any{"collection": "posts", "source": "body", "target": "summary", "approval": "true"}, "approval must be true or false"},
		"string number":        {"summarizer", map[string]any{"collection": "posts", "source": "body", "target": "summary", "words": "30"}, "words must be a number"},
		"string for a list":    {"tagger", map[string]any{"collection": "posts", "sources": "body", "target": "tags"}, "sources must be a list"},
		"same flag values":     {"moderator", map[string]any{"collection": "posts", "sources": []any{"body"}, "flag": "review", "flagged_value": "ok", "ok_value": "ok"}, "must differ"},
		"reason is the flag":   {"moderator", map[string]any{"collection": "posts", "sources": []any{"body"}, "flag": "flagged", "reason": "body"}, "own fields"},
	} {
		_, err := Render(tc.id, tc.params)
		assert.ErrorContains(t, err, tc.err, name)
	}
	_, err := Render("nope", nil)
	assert.ErrorIs(t, err, ErrUnknown)
}
