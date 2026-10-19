package graphql

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/contract"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/require"
)

// TestAPIContract keeps api/graphql.json (the introspection result) in sync
// with the GraphQL API.
func TestAPIContract(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, migrations.MigrateDatabase(db))
	t.Cleanup(ResetTypeRegistry)
	require.NoError(t, contract.Seed())
	require.NoError(t, InitializeGraphQLSchema())

	res := graphql.Do(graphql.Params{Schema: GetSchema(), RequestString: contract.IntrospectionQuery})
	require.Empty(t, res.Errors)
	out, err := contract.Encode(map[string]any{"data": res.Data})
	require.NoError(t, err)
	require.NoError(t, contract.Compare("graphql.json", out))
}
