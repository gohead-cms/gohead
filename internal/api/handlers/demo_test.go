package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/demo"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemoMode(t *testing.T) {
	a := rolesFixture(t)
	snapshot := filepath.Join(t.TempDir(), "demo.snapshot")
	require.NoError(t, demo.Configure(demo.Config{
		Enabled: true, Username: "demo-user", Password: "try-gohead-demo", ResetAt: "03:00", Snapshot: snapshot,
	}))
	t.Cleanup(func() { _ = demo.Configure(demo.Config{}) })
	require.NoError(t, demo.EnsureAccount())
	require.NoError(t, demo.EnsureAccount(), "runs at every start")

	a.router.GET("/auth/me", middleware.AuthMiddleware(), Me)
	a.router.GET("/auth/demo", DemoInfo)
	admin := a.router.Group("/admin", middleware.AuthMiddleware(), middleware.AdminOnly())
	admin.GET("/collections", GetCollections)
	admin.GET("/demo", DemoStatus)
	admin.POST("/demo/snapshot", TakeDemoSnapshot)
	admin.POST("/demo/reset", ResetDemo)

	t.Run("the login page gets the account", func(t *testing.T) {
		info := a.ok("", "GET", "/auth/demo", nil).(map[string]any)
		assert.Equal(t, "demo-user", info["username"])
		assert.Equal(t, "try-gohead-demo", info["password"])
		assert.NotEmpty(t, info["nextReset"])
		me := a.ok("demo", "GET", "/auth/me", nil).(map[string]any)
		assert.Equal(t, true, me["demo"])
	})

	t.Run("the admin is read-only", func(t *testing.T) {
		a.ok("demo", "GET", "/admin/collections", nil)
		for _, req := range [][2]string{
			{"POST", "/admin/collections"}, {"PUT", "/admin/collections/pages"},
			{"POST", "/admin/demo/snapshot"}, {"POST", "/admin/demo/reset"}, {"POST", "/admin/agents"},
		} {
			code, _ := a.do("demo", req[0], req[1], map[string]any{"name": "x"})
			assert.Equal(t, http.StatusForbidden, code, req)
		}
	})

	t.Run("drafts only", func(t *testing.T) {
		art := a.ok("demo", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{"title": "From a visitor"}}).(map[string]any)
		id := int(art["id"].(float64))
		code, _ := a.do("demo", "POST", fmt.Sprintf("/api/collections/articles/%d/publish", id), nil)
		assert.Equal(t, http.StatusForbidden, code, "no publishing")
		code, _ = a.do("demo", "DELETE", fmt.Sprintf("/api/collections/articles/%d", id), nil)
		assert.Equal(t, http.StatusForbidden, code, "no deleting")
	})

	t.Run("calls are shown, not decided", func(t *testing.T) {
		p, err := storage.ProposeAppCall(storage.Actor{Kind: models.ActorAgent, Name: "bot"}, "tracker", "create_issue", map[string]any{"title": "x"}, "")
		require.NoError(t, err)
		found := false
		for _, item := range a.ok("demo", "GET", "/api/proposals", nil).([]any) {
			if field(item, "id") == float64(p.ID) {
				found = true
			}
		}
		assert.True(t, found, "visitors see calls")
		a.ok("demo", "GET", fmt.Sprintf("/api/proposals/%d", p.ID), nil)
		for _, action := range []string{"approve", "reject"} {
			code, _ := a.do("demo", "POST", fmt.Sprintf("/api/proposals/%d/%s", p.ID, action), nil)
			assert.Equal(t, http.StatusForbidden, code, action)
		}
	})

	t.Run("snapshot and reset", func(t *testing.T) {
		if database.DB.Dialector.Name() != "sqlite" {
			t.Skip("demo snapshots are SQLite only")
		}
		a.ok("admin", "POST", "/api/collections/pages", map[string]any{"data": map[string]any{"title": "Kept"}})
		snap := a.ok("admin", "POST", "/admin/demo/snapshot", nil).(map[string]any)
		assert.NotEmpty(t, snap["snapshotAt"])
		a.ok("demo", "POST", "/api/collections/pages", map[string]any{"data": map[string]any{"title": "Gone after the reset"}})
		require.NoError(t, demo.Reset(context.Background(), AfterDemoReset))

		pages := a.ok("admin", "GET", "/api/collections/pages", nil).([]any)
		var titles []any
		for _, p := range pages {
			titles = append(titles, field(p, "title"))
		}
		assert.Contains(t, titles, "Kept")
		assert.NotContains(t, titles, "Gone after the reset")
		// The demo account still signs in.
		a.ok("demo", "GET", "/auth/me", nil)
	})
}
