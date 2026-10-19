package openapi

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/contract"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/require"
)

// TestAPIContract keeps api/openapi.json in sync with the REST API.
func TestAPIContract(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, migrations.MigrateDatabase(db))
	require.NoError(t, contract.Seed())

	doc, err := Build("contract")
	require.NoError(t, err)
	out, err := contract.Encode(doc)
	require.NoError(t, err)
	require.NoError(t, contract.Compare("openapi.json", out))
}
