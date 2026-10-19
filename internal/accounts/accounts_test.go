package accounts

import (
	"errors"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/seed"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func setup(t *testing.T) {
	t.Helper()
	_, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.UserRole{}, &models.User{}))
	seed.SeedRoles()
}

func TestBootstrapAdmin(t *testing.T) {
	setup(t)

	created, err := BootstrapAdmin("", "", "")
	require.NoError(t, err)
	assert.False(t, created, "nothing configured, nothing created")

	_, err = BootstrapAdmin("root", "root@example.com", "short")
	assert.True(t, errors.Is(err, ErrInvalid), "admin passwords need 12 characters")

	created, err = BootstrapAdmin("root", "root@example.com", "a-long-enough-password")
	require.NoError(t, err)
	assert.True(t, created)
	u, err := storage.GetUserByUsername("root")
	require.NoError(t, err)
	assert.Equal(t, "admin", u.Role.Name)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("a-long-enough-password")), "stored hashed")

	created, err = BootstrapAdmin("other", "other@example.com", "another-long-password")
	require.NoError(t, err)
	assert.False(t, created, "only on an empty instance")
	_, err = storage.GetUserByUsername("other")
	assert.Error(t, err)
}

func TestBootstrapAdminNeedsPassword(t *testing.T) {
	setup(t)
	_, err := BootstrapAdmin("root", "", "")
	assert.True(t, errors.Is(err, ErrInvalid))
}

func TestSetPassword(t *testing.T) {
	setup(t)
	_, err := CreateAdmin("alice", "alice@example.com", "first-password-123")
	require.NoError(t, err)

	assert.True(t, errors.Is(SetPassword("alice", "too-short"), ErrInvalid))
	require.NoError(t, SetPassword("alice", "second-password-456"))
	u, err := storage.GetUserByUsername("alice")
	require.NoError(t, err)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("second-password-456")))
	assert.Error(t, SetPassword("nobody", "whatever-long-password"))
}
