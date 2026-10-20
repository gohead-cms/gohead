package migrations

import (
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacyPublicRoleIsRenamed(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()
	require.NoError(t, db.AutoMigrate(&models.UserRole{}, &models.APIToken{}))
	legacy := models.UserRole{Name: "public", Permissions: models.JSONMap{"content": map[string]any{"*": []any{"read", "update"}}}}
	require.NoError(t, db.Create(&legacy).Error)
	require.NoError(t, db.Create(&models.UserRole{Name: "public-legacy"}).Error)
	require.NoError(t, db.Create(&models.APIToken{Name: "old", Role: "public", Hash: "h1", Prefix: "ght_x"}).Error)

	require.NoError(t, publicRoleUp(db))

	var renamed models.UserRole
	require.NoError(t, db.First(&renamed, legacy.ID).Error)
	assert.Equal(t, "public-legacy-2", renamed.Name, "a free name is picked")
	assert.Equal(t, legacy.Permissions, renamed.Permissions, "its permissions are kept")
	var tok models.APIToken
	require.NoError(t, db.Where("name = ?", "old").First(&tok).Error)
	assert.Equal(t, "public-legacy-2", tok.Role, "tokens follow the role")
	var n int64
	db.Model(&models.UserRole{}).Where("name = ?", "public").Count(&n)
	assert.Zero(t, n, "the public role is seeded afresh, granting nothing")

	require.NoError(t, publicRoleUp(db), "nothing to do the second time")
}
