package apps_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gohead-cms/gohead/internal/apps"
	"github.com/gohead-cms/gohead/internal/apps/apptest"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/secrets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAndCall(t *testing.T) {
	srv := apptest.New()
	defer srv.Close()
	ctx := context.Background()
	ep := apps.Endpoint{URL: srv.URL, AuthType: models.ConnectionAuthBearer, Token: apptest.Token}

	t.Run("private addresses are refused by default", func(t *testing.T) {
		apps.Configure(apps.Config{})
		_, err := apps.ListTools(ctx, ep)
		require.Error(t, err)
		assert.ErrorIs(t, err, apps.ErrBlockedAddress)
	})

	apps.Configure(apps.Config{AllowPrivateNetworks: true})
	defer apps.Configure(apps.Config{})

	t.Run("lists tools with their hints", func(t *testing.T) {
		list, err := apps.ListTools(ctx, ep)
		require.NoError(t, err)
		require.Len(t, list, 2)
		byName := map[string]models.ConnectionTool{}
		for _, tool := range list {
			byName[tool.Name] = tool
		}
		search := byName["search"]
		assert.True(t, search.ReadOnly)
		assert.Equal(t, "Search issues", search.Title)
		assert.Equal(t, "object", search.InputSchema["type"])
		create := byName["create_issue"]
		assert.False(t, create.ReadOnly, "tools that say nothing may change things")
		assert.True(t, create.Destructive)
	})

	t.Run("a wrong token is refused", func(t *testing.T) {
		_, err := apps.ListTools(ctx, apps.Endpoint{URL: srv.URL, AuthType: models.ConnectionAuthBearer, Token: "nope"})
		require.Error(t, err)
	})

	t.Run("calls return JSON for the model", func(t *testing.T) {
		res, err := apps.Call(ctx, ep, "search", map[string]any{"query": "bug"})
		require.NoError(t, err)
		var out map[string]any
		require.NoError(t, json.Unmarshal([]byte(res.JSON()), &out))
		assert.Equal(t, "success", out["status"])
		assert.Equal(t, "bug", out["result"].(map[string]any)["query"], "JSON text is given as JSON")

		res, err = apps.Call(ctx, ep, "create_issue", map[string]any{"title": "fail"})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.JSON(), `"status":"error"`)
		assert.Len(t, srv.Calls(), 2)
	})
}

func TestSecrets(t *testing.T) {
	defer func() { _ = secrets.SetKey("") }()

	require.NoError(t, secrets.SetKey(""))
	plain, err := secrets.Seal("tok")
	require.NoError(t, err)
	assert.Equal(t, "tok", plain, "without a key, values are kept as they are")

	require.NoError(t, secrets.SetKey("a-long-jwt-secret-of-at-least-32-chars"))
	sealed, err := secrets.Seal("tok")
	require.NoError(t, err)
	assert.NotContains(t, sealed, "tok")
	got, err := secrets.Open(sealed)
	require.NoError(t, err)
	assert.Equal(t, "tok", got)
	got, err = secrets.Open(plain)
	require.NoError(t, err)
	assert.Equal(t, "tok", got, "values stored without a key still read")

	require.NoError(t, secrets.SetKey("another-secret-another-secret-another"))
	_, err = secrets.Open(sealed)
	assert.ErrorIs(t, err, secrets.ErrUndecryptable)
}
