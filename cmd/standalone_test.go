package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueueDriver(t *testing.T) {
	for _, tc := range []struct {
		driver     string
		standalone bool
		want       string
	}{
		{"", false, queue.DriverRedis},
		{"", true, queue.DriverDatabase},
		{"redis", true, queue.DriverRedis},
		{" Database ", false, queue.DriverDatabase},
	} {
		got, err := queueDriver(config.Config{Queue: config.QueueConfig{Driver: tc.driver}}, tc.standalone)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, tc)
	}
	_, err := queueDriver(config.Config{Queue: config.QueueConfig{Driver: "kafka"}}, false)
	assert.Error(t, err)
}

func TestStandaloneDataDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	data := filepath.Join(dir, "data")

	// Defaults go to the data directory.
	cfg, err := config.LoadConfig(filepath.Join(dir, "missing.yaml"))
	require.NoError(t, err)
	secretFile, err := applyDataDir(&cfg, data)
	require.NoError(t, err)
	assert.Equal(t, "sqlite://"+filepath.Join(data, "gohead.db"), cfg.DatabaseURL)
	assert.Equal(t, filepath.Join(data, "uploads"), cfg.Media.Local.Path)
	assert.Equal(t, filepath.Join(data, "jwt_secret"), secretFile)
	assert.DirExists(t, data)
	assert.False(t, useRedis(cfg, queue.DriverDatabase), "Redis only when configured")
	assert.True(t, useRedis(cfg, queue.DriverRedis))

	// Explicit settings win.
	t.Setenv("GOHEAD_DATABASE_URL", "postgres://u:p@db:5432/gohead")
	t.Setenv("GOHEAD_REDIS_ADDRESS", "redis:6379")
	cfg, err = config.LoadConfig(filepath.Join(dir, "missing.yaml"))
	require.NoError(t, err)
	_, err = applyDataDir(&cfg, data)
	require.NoError(t, err)
	assert.Equal(t, "postgres://u:p@db:5432/gohead", cfg.DatabaseURL)
	assert.True(t, useRedis(cfg, queue.DriverDatabase))

	// A config file setting too.
	file := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte("media:\n  local:\n    path: /srv/media\n"), 0o600))
	cfg, err = config.LoadConfig(file)
	require.NoError(t, err)
	_, err = applyDataDir(&cfg, data)
	require.NoError(t, err)
	assert.Equal(t, "/srv/media", cfg.Media.Local.Path)
}

func TestJWTSecretIsKeptInTheDataDirectory(t *testing.T) {
	logger.InitLogger("error")
	file := filepath.Join(t.TempDir(), "jwt_secret")
	first := resolveJWTSecret("", file)
	assert.Len(t, first, 64)
	info, err := os.Stat(file)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	assert.Equal(t, first, resolveJWTSecret("", file), "same secret after a restart")
	assert.Equal(t, "a-configured-secret-that-is-long-enough-123", resolveJWTSecret("a-configured-secret-that-is-long-enough-123", file))
	assert.NotEqual(t, first, resolveJWTSecret("", ""), "no file: a new secret each run")
}
