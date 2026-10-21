package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/apps"
	"github.com/gohead-cms/gohead/internal/apps/apptest"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnections(t *testing.T) {
	a := rolesFixture(t)
	admin := a.router.Group("/admin", middleware.AuthMiddleware(), middleware.AdminOnly())
	admin.GET("/connection-presets", ListConnectionPresets)
	admin.GET("/connections", ListConnections)
	admin.POST("/connections", CreateConnection)
	admin.GET("/connections/:name", GetConnection)
	admin.PUT("/connections/:name", UpdateConnection)
	admin.DELETE("/connections/:name", DeleteConnection)
	admin.POST("/connections/:name/refresh", RefreshConnection)
	admin.GET("/agent-functions", AgentFunctionCatalog)

	srv := apptest.New()
	defer srv.Close()
	apps.Configure(apps.Config{AllowPrivateNetworks: true})
	defer apps.Configure(apps.Config{})

	t.Run("presets", func(t *testing.T) {
		list := a.ok("admin", "GET", "/admin/connection-presets", nil).([]any)
		assert.Equal(t, "github", field(list[0], "id"))
	})

	t.Run("create lists the tools, the token is write-only", func(t *testing.T) {
		out := a.ok("admin", "POST", "/admin/connections", map[string]any{
			"name": "tracker", "label": "Tracker", "url": srv.URL, "authType": "bearer", "token": apptest.Token,
		}).(map[string]any)
		assert.Equal(t, true, out["hasToken"])
		assert.NotContains(t, out, "token")
		assert.Empty(t, out["lastError"])
		assert.Len(t, out["tools"], 2)
		stored, err := storage.GetConnection("tracker")
		require.NoError(t, err)
		assert.NotEmpty(t, stored.Secret)
	})

	t.Run("bad input", func(t *testing.T) {
		for _, bad := range []map[string]any{
			{"name": "content", "url": srv.URL},
			{"name": "Bad Name", "url": srv.URL},
			{"name": "x1", "url": "ftp://example.com"},
			{"name": "x2", "url": srv.URL, "authType": "header", "headerName": "Host"},
			{"preset": "nope"},
		} {
			code, out := a.do("admin", "POST", "/admin/connections", bad)
			assert.Equal(t, http.StatusBadRequest, code, "%v → %v", bad, out)
		}
		code, _ := a.do("admin", "POST", "/admin/connections", map[string]any{"name": "tracker", "url": srv.URL})
		assert.Equal(t, http.StatusConflict, code)
		code, _ = a.do("editor", "GET", "/admin/connections", nil)
		assert.Equal(t, http.StatusForbidden, code)
	})

	t.Run("a wrong token is saved with the error", func(t *testing.T) {
		code, out := a.do("admin", "POST", "/admin/connections", map[string]any{
			"name": "broken", "url": srv.URL, "authType": "bearer", "token": "nope",
		})
		require.Equal(t, http.StatusCreated, code, out)
		assert.NotEmpty(t, field(out["data"], "lastError"))
		assert.NotEmpty(t, field(out["meta"], "warning"))
		a.ok("admin", "DELETE", "/admin/connections/broken", nil)
	})

	t.Run("the token goes nowhere else", func(t *testing.T) {
		code, _ := a.do("admin", "PUT", "/admin/connections/tracker", map[string]any{"url": "https://example.com/mcp"})
		assert.Equal(t, http.StatusBadRequest, code, "changing the URL needs the token again")
		out := a.ok("admin", "PUT", "/admin/connections/tracker", map[string]any{"label": "Issues"}).(map[string]any)
		assert.Equal(t, "Issues", out["label"])
		assert.Equal(t, true, out["hasToken"])
	})

	t.Run("app tools are in the catalog and agents can use them", func(t *testing.T) {
		catalog := a.ok("admin", "GET", "/admin/agent-functions", nil).([]any)
		var found map[string]any
		for _, t := range catalog {
			if field(t, "use") == "tracker.create_issue" {
				found = t.(map[string]any)
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "apps", found["group"])
		assert.Equal(t, "tracker_create_issue", found["name"])
		assert.Equal(t, true, found["writes"])
		assert.Equal(t, "Issues", found["appLabel"])

		agent := map[string]any{
			"name": "triager", "system_prompt": "Triage.", "max_turns": 2,
			"llm_config": map[string]any{"provider": "openai", "model": "m", "api_key": "k"},
			"memory":     map[string]any{"type": "in-memory", "session_scope": "agent"},
			"trigger":    map[string]any{"type": "manual"},
			"tools":      []any{map[string]any{"use": "tracker.create_issue"}, map[string]any{"use": "tracker.search", "approval": true}},
		}
		a.ok("admin", "POST", "/admin/agents", agent)
		agent["name"] = "bad"
		agent["tools"] = []any{map[string]any{"use": "tracker.nope"}}
		code, out := a.do("admin", "POST", "/admin/agents", agent)
		assert.Equal(t, http.StatusBadRequest, code, out)
	})

	t.Run("calls wait for an admin", func(t *testing.T) {
		bot := storage.Actor{Kind: models.ActorAgent, Name: "triager", AgentRunID: 3}
		ok, err := storage.ProposeAppCall(bot, "tracker", "create_issue", map[string]any{"title": "Crash"}, "")
		require.NoError(t, err)
		bad, err := storage.ProposeAppCall(bot, "tracker", "create_issue", map[string]any{"title": "fail"}, "")
		require.NoError(t, err)

		for _, p := range a.ok("editor", "GET", "/api/proposals", nil).([]any) {
			assert.NotEqual(t, "call", field(p, "action"), "editors don't review calls")
		}
		code, _ := a.do("editor", "GET", fmt.Sprintf("/api/proposals/%d", ok.ID), nil)
		assert.Equal(t, http.StatusNotFound, code)

		got := a.ok("admin", "GET", fmt.Sprintf("/api/proposals/%d", ok.ID), nil).(map[string]any)
		assert.Equal(t, "create_issue", field(got["call"], "tool"))
		assert.Equal(t, "Crash", field(field(got["call"], "arguments"), "title"))
		assert.Equal(t, "Issues", field(got["call"], "connectionLabel"))

		out := a.ok("admin", "POST", fmt.Sprintf("/api/proposals/%d/approve", ok.ID), map[string]any{"arguments": map[string]any{"title": "Crash on save"}}).(map[string]any)
		assert.Equal(t, "approved", field(out["proposal"], "status"))
		assert.Equal(t, "created issue #7", out["result"])
		calls := srv.Calls()
		require.NotEmpty(t, calls)
		assert.Equal(t, "Crash on save", calls[len(calls)-1].Arguments["title"], "the reviewer's arguments are used")

		code, body := a.do("admin", "POST", fmt.Sprintf("/api/proposals/%d/approve", bad.ID), nil)
		assert.Equal(t, http.StatusUnprocessableEntity, code, body)
		p, err := storage.GetProposal(bad.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ProposalFailed, p.Status)
		assert.Contains(t, p.Error, "title refused")
	})

	t.Run("deleting a connection rejects its pending calls", func(t *testing.T) {
		p, err := storage.ProposeAppCall(storage.Actor{Kind: models.ActorAgent, Name: "triager"}, "tracker", "create_issue", map[string]any{"title": "x"}, "")
		require.NoError(t, err)
		a.ok("admin", "DELETE", "/admin/connections/tracker", nil)
		p, err = storage.GetProposal(p.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ProposalRejected, p.Status)
		code, _ := a.do("admin", "GET", "/admin/connections/tracker", nil)
		assert.Equal(t, http.StatusNotFound, code)
	})
}
