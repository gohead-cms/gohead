package ratelimit

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logger.InitLogger("error")
	logger.Log.SetOutput(&bytes.Buffer{})
}

func exercise(t *testing.T, c Counter, advance func(time.Duration)) {
	t.Helper()
	ctx := context.Background()

	for i := int64(1); i <= 3; i++ {
		n, reset, err := c.Hit(ctx, "k", time.Minute)
		require.NoError(t, err)
		assert.Equal(t, i, n)
		assert.Greater(t, reset, 50*time.Second)
	}
	n, _, err := c.Count(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, int64(3), n, "Count does not increment")

	n, _, _ = c.Count(ctx, "other")
	assert.Equal(t, int64(0), n)

	advance(61 * time.Second)
	n, _, _ = c.Hit(ctx, "k", time.Minute)
	assert.Equal(t, int64(1), n, "a new window starts after expiry")

	require.NoError(t, c.Reset(ctx, "k"))
	n, _, _ = c.Count(ctx, "k")
	assert.Equal(t, int64(0), n)
}

func TestMemory(t *testing.T) {
	m := NewMemory()
	now := time.Now()
	m.now = func() time.Time { return now }
	exercise(t, m, func(d time.Duration) { now = now.Add(d) })
}

func TestRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	exercise(t, NewRedis(rdb, "test:"), mr.FastForward)
}

// Two replicas sharing Redis share the budget.
func TestRedisSharedAcrossReplicas(t *testing.T) {
	mr := miniredis.RunT(t)
	a := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	b := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer a.Close()
	defer b.Close()
	ca, cb := NewRedis(a, "p:"), NewRedis(b, "p:")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _, _, _ = ca.Hit(context.Background(), "ip", time.Minute) }()
		go func() { defer wg.Done(); _, _, _ = cb.Hit(context.Background(), "ip", time.Minute) }()
	}
	wg.Wait()
	n, reset, err := ca.Count(context.Background(), "ip")
	require.NoError(t, err)
	assert.Equal(t, int64(100), n, "hits from both replicas are counted atomically")
	assert.Greater(t, reset, time.Duration(0), "the key always has an expiry")
}

func TestFallbackWhenRedisIsDown(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	defer rdb.Close()
	c := New(rdb, "p:")

	n, _, err := c.Hit(context.Background(), "k", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	mr.Close() // Redis goes away
	for i := int64(1); i <= 3; i++ {
		n, _, err = c.Hit(context.Background(), "k", time.Minute)
		require.NoError(t, err, "errors are absorbed by the fallback")
		assert.Equal(t, i, n, "limits keep working per replica")
	}
}

func TestNewWithoutRedis(t *testing.T) {
	_, ok := New(nil, "p:").(*Memory)
	assert.True(t, ok)
}
