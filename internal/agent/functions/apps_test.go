package functions

import (
	"context"
	"testing"

	"github.com/gohead-cms/gohead/internal/agent/tools"
	"github.com/gohead-cms/gohead/internal/apps"
	"github.com/gohead-cms/gohead/internal/apps/apptest"
	"github.com/gohead-cms/gohead/internal/models"
	agentModels "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppTools(t *testing.T) {
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.Connection{}, &models.Proposal{}))
	srv := apptest.New()
	defer srv.Close()
	apps.Configure(apps.Config{AllowPrivateNetworks: true})
	defer apps.Configure(apps.Config{})

	conn := &models.Connection{Name: "tracker", Label: "Tracker", URL: srv.URL, AuthType: models.ConnectionAuthBearer, Secret: apptest.Token}
	require.NoError(t, storage.CreateConnection(conn))
	require.NoError(t, apps.Refresh(context.Background(), conn))

	info, ok := tools.Lookup("tracker.search")
	require.True(t, ok, "app tools are looked up like built-in ones")
	assert.Equal(t, "tracker_search", info.Name)
	assert.False(t, info.Writes)
	assert.NoError(t, CheckTools(agentModels.FunctionSpecs{{ImplKey: "tracker.create_issue"}}))
	assert.ErrorContains(t, CheckTools(agentModels.FunctionSpecs{{ImplKey: "tracker.nope"}}), "has no such tool")

	no := false
	r := NewRegistry(agentModels.FunctionSpecs{
		{ImplKey: "tracker.search"},
		{ImplKey: "tracker.create_issue"},
		{ImplKey: "tracker.create_issue", Name: "create_now", Approval: &no},
	})
	search, ok := r.Get("tracker_search")
	require.True(t, ok)
	create, _ := r.Get("tracker_create_issue")
	createNow, _ := r.Get("create_now")
	ctx := storage.WithActor(context.Background(), storage.Actor{Kind: models.ActorAgent, Name: "bot", AgentRunID: 2})

	t.Run("read-only tools are called", func(t *testing.T) {
		out, err := search(ctx, `{"query":"crash"}`)
		require.NoError(t, err)
		assert.Contains(t, out, `"status":"success"`)
		assert.Contains(t, out, `"crash"`)
	})

	t.Run("tools that change things wait for approval", func(t *testing.T) {
		before := len(srv.Calls())
		out, err := create(ctx, map[string]any{"title": "Crash"})
		require.NoError(t, err)
		assert.Contains(t, out, `"pending_approval"`, out)
		assert.Len(t, srv.Calls(), before, "nothing was called")
		list, _, err := storage.ListProposals(storage.ProposalQuery{})
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, models.ProposalCall, list[0].Action)
		assert.Equal(t, "app:tracker", list[0].Collection)
		assert.Equal(t, "create_issue", list[0].Data["tool"])
		require.NotNil(t, list[0].AgentRunID)
	})

	t.Run("unless the agent says not to", func(t *testing.T) {
		out, err := createNow(ctx, map[string]any{"title": "Now"})
		require.NoError(t, err)
		assert.Contains(t, out, "created issue #7", out)

		// An agent that requires approval has every change approved.
		out, _ = createNow(WithWriteMode(ctx, WritePropose), map[string]any{"title": "Later"})
		assert.Contains(t, out, `"pending_approval"`, out)
	})

	t.Run("test runs call nothing that changes things", func(t *testing.T) {
		before := len(srv.Calls())
		dry := WithWriteMode(ctx, WriteDryRun)
		out, _ := createNow(dry, map[string]any{"title": "x"})
		assert.Contains(t, out, `"would_call"`, out)
		assert.Len(t, srv.Calls(), before)
		out, _ = search(dry, map[string]any{"query": "x"})
		assert.Contains(t, out, `"success"`, "reading is fine")
	})

	t.Run("a deleted connection answers with an error", func(t *testing.T) {
		require.NoError(t, storage.DeleteConnection("tracker"))
		out, err := search(ctx, map[string]any{"query": "x"})
		require.NoError(t, err)
		assert.Contains(t, out, `"error"`)
		_, ok := tools.Lookup("tracker.search")
		assert.False(t, ok)
	})
}
