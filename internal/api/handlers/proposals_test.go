package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalReview(t *testing.T) {
	a := rolesFixture(t)
	page := a.ok("admin", "POST", "/api/collections/pages", map[string]any{"data": map[string]any{"title": "About"}}).(map[string]any)
	art := a.ok("admin", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{"title": "Draft"}}).(map[string]any)
	pages, err := storage.GetCollectionByName("pages")
	require.NoError(t, err)
	articles, err := storage.GetCollectionByName("articles")
	require.NoError(t, err)

	bot := storage.Actor{Kind: models.ActorAgent, Name: "bot", AgentRunID: 4}
	onPage, err := storage.ProposeItemWrite(bot, *pages, models.ProposalUpdate, uint(page["id"].(float64)), "", models.JSONMap{"title": "About us"}, "clearer")
	require.NoError(t, err)
	onArticle, err := storage.ProposeItemWrite(bot, *articles, models.ProposalUpdate, uint(art["id"].(float64)), "", models.JSONMap{"title": "Better"}, "")
	require.NoError(t, err)
	deleteArticle, err := storage.ProposeItemWrite(bot, *articles, models.ProposalDelete, uint(art["id"].(float64)), "", nil, "")
	require.NoError(t, err)

	t.Run("reviewers only see what they may change", func(t *testing.T) {
		assert.Len(t, a.ok("editor", "GET", "/api/proposals", nil), 3)
		list := a.ok("writer", "GET", "/api/proposals", nil).([]any)
		assert.Len(t, list, 2, "the writer can change articles, not pages")
		code, _ := a.do("viewer", "GET", "/api/proposals", nil)
		assert.Equal(t, http.StatusForbidden, code)
		code, _ = a.do("writer", "GET", fmt.Sprintf("/api/proposals/%d", onPage.ID), nil)
		assert.Equal(t, http.StatusNotFound, code)
		code, _ = a.do("writer", "POST", fmt.Sprintf("/api/proposals/%d/approve", deleteArticle.ID), nil)
		assert.Equal(t, http.StatusNotFound, code, "approving a delete needs delete")
		code, _ = a.do("editor", "GET", "/api/proposals?status=nope", nil)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("details show the change", func(t *testing.T) {
		got := a.ok("editor", "GET", fmt.Sprintf("/api/proposals/%d", onPage.ID), nil).(map[string]any)
		assert.Equal(t, false, got["stale"])
		assert.Equal(t, "About", field(got["current"], "title"))
		changes := got["changes"].([]any)
		require.Len(t, changes, 1)
		assert.Equal(t, "About us", field(changes[0], "after"))
	})

	t.Run("approve, edit, reject", func(t *testing.T) {
		out := a.ok("editor", "POST", fmt.Sprintf("/api/proposals/%d/approve", onPage.ID), map[string]any{"data": map[string]any{"title": "About GoHead"}}).(map[string]any)
		assert.Equal(t, "approved", field(out["proposal"], "status"))
		assert.Equal(t, "About GoHead", field(out["item"], "title"))
		code, _ := a.do("editor", "POST", fmt.Sprintf("/api/proposals/%d/approve", onPage.ID), nil)
		assert.Equal(t, http.StatusConflict, code)

		// The writer cannot apply a change to a published article.
		a.ok("admin", "POST", fmt.Sprintf("/api/collections/articles/%d/publish", int(art["id"].(float64))), nil)
		code, _ = a.do("writer", "POST", fmt.Sprintf("/api/proposals/%d/approve", onArticle.ID), map[string]any{"force": true})
		assert.Equal(t, http.StatusForbidden, code)

		// The article changed (published) since: stale until forced.
		code, out2 := a.do("editor", "POST", fmt.Sprintf("/api/proposals/%d/approve", onArticle.ID), nil)
		assert.Equal(t, http.StatusConflict, code, out2)
		a.ok("editor", "POST", fmt.Sprintf("/api/proposals/%d/approve", onArticle.ID), map[string]any{"force": true})

		out = a.ok("editor", "POST", fmt.Sprintf("/api/proposals/%d/reject", deleteArticle.ID), map[string]any{"note": "keep it"}).(map[string]any)
		assert.Equal(t, "rejected", field(out["proposal"], "status"))
		assert.Equal(t, "keep it", field(out["proposal"], "reviewNote"))
		assert.Empty(t, a.ok("editor", "GET", "/api/proposals", nil))
		assert.Len(t, a.ok("editor", "GET", "/api/proposals?status=all&agent=bot", nil), 3)
	})
}

func TestAgentTemplatesAndTestRuns(t *testing.T) {
	a := rolesFixture(t)
	templates := a.ok("admin", "GET", "/admin/agent-templates", nil).([]any)
	require.Len(t, templates, 4)
	code, _ := a.do("editor", "GET", "/admin/agent-templates", nil)
	assert.Equal(t, http.StatusForbidden, code)

	code, out := a.do("admin", "POST", "/admin/agent-templates/summarizer/render", map[string]any{
		"params": map[string]any{"collection": "articles", "source": "title", "target": "title"},
	})
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, a.errMsg(out), "own field")
	code, out = a.do("admin", "POST", "/admin/agent-templates/summarizer/render", map[string]any{"params": map[string]any{"collection": "nope"}})
	assert.Equal(t, http.StatusBadRequest, code, out)
	code, _ = a.do("admin", "POST", "/admin/agent-templates/nope/render", map[string]any{"params": map[string]any{}})
	assert.Equal(t, http.StatusNotFound, code)

	// A rendered definition is a valid agent.
	a.ok("admin", "PUT", "/admin/collections/pages", map[string]any{
		"name": "pages", "kind": "collection",
		"attributes": map[string]any{"title": map[string]any{"type": "string"}, "summary": map[string]any{"type": "text"}},
	})
	rendered := a.ok("admin", "POST", "/admin/agent-templates/summarizer/render", map[string]any{
		"params": map[string]any{"collection": "pages", "source": "title", "target": "summary"},
	})
	created := a.ok("admin", "POST", "/admin/agents", rendered).(map[string]any)
	assert.Equal(t, "summarizer-pages", created["name"])

	// Without Redis there are no runs, test runs included.
	code, _ = a.do("admin", "POST", "/admin/agents/summarizer-pages/test", map[string]any{"input": "hi"})
	assert.Equal(t, http.StatusServiceUnavailable, code)
	code, _ = a.do("admin", "POST", "/admin/agents/summarizer-pages/test", map[string]any{"item_id": 999})
	assert.Equal(t, http.StatusBadRequest, code, "the item must exist")
}
