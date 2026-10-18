package cluster

import (
	"bytes"
	"context"
	"sync/atomic"
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

func newReplica(t *testing.T, mr *miniredis.Miniredis, poll time.Duration) (*SchemaSync, *atomic.Int32) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	s := NewSchemaSync(rdb, WithPollInterval(poll))
	reloads := &atomic.Int32{}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go s.Run(ctx, func() error { reloads.Add(1); return nil })
	return s, reloads
}

func TestSchemaSyncBroadcast(t *testing.T) {
	mr := miniredis.RunT(t)
	a, aReloads := newReplica(t, mr, time.Hour)
	_, bReloads := newReplica(t, mr, time.Hour)
	_, cReloads := newReplica(t, mr, time.Hour)

	// Let the subscriptions register before publishing.
	require.Eventually(t, func() bool {
		return len(mr.PubSubNumSub(DefaultChannel)) > 0 &&
			mr.PubSubNumSub(DefaultChannel)[DefaultChannel] == 3
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, a.Notify(context.Background()))

	assert.Eventually(t, func() bool { return bReloads.Load() == 1 && cReloads.Load() == 1 },
		2*time.Second, 10*time.Millisecond, "other replicas reload once")
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(0), aReloads.Load(), "the notifying replica does not reload its own change")
	assert.Equal(t, int32(1), bReloads.Load(), "no duplicate reloads")
}

func TestSchemaSyncCatchesUpByPolling(t *testing.T) {
	mr := miniredis.RunT(t)
	_, reloads := newReplica(t, mr, 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond) // replica started and read version 0

	// A change whose notification was lost (e.g. during a disconnect):
	// only the version key moved.
	_, err := mr.Incr(DefaultVersionKey, 1)
	require.NoError(t, err)

	assert.Eventually(t, func() bool { return reloads.Load() == 1 }, 2*time.Second, 20*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), reloads.Load(), "a version is applied once")
}

func TestSchemaSyncStartsAtCurrentVersion(t *testing.T) {
	mr := miniredis.RunT(t)
	// Changes made before this replica started are already in its schema.
	_, err := mr.Incr(DefaultVersionKey, 7)
	require.NoError(t, err)

	_, reloads := newReplica(t, mr, 30*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(0), reloads.Load())
}

func TestSchemaSyncWithoutRedisIsNoop(t *testing.T) {
	s := NewSchemaSync(nil)
	assert.NoError(t, s.Notify(context.Background()))

	done := make(chan struct{})
	go func() { s.Run(context.Background(), func() error { return nil }); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run should return immediately without Redis")
	}
}

// A change made while Redis is down must still reach the other replicas
// once Redis is back.
func TestSchemaSyncRetriesAnnouncementAfterOutage(t *testing.T) {
	mr := miniredis.RunT(t)
	a, _ := newReplica(t, mr, 50*time.Millisecond)
	_, bReloads := newReplica(t, mr, 50*time.Millisecond)
	time.Sleep(150 * time.Millisecond)

	mr.Close()
	assert.Error(t, a.Notify(context.Background()), "Redis is down")
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(0), bReloads.Load())

	require.NoError(t, mr.Restart())
	assert.Eventually(t, func() bool { return bReloads.Load() >= 1 }, 3*time.Second, 20*time.Millisecond,
		"the pending announcement is retried and picked up")
}

// If Redis loses its data, the version counter restarts at zero; replicas
// must still notice later changes.
func TestSchemaSyncSurvivesCounterReset(t *testing.T) {
	mr := miniredis.RunT(t)
	a, _ := newReplica(t, mr, 50*time.Millisecond)
	_, bReloads := newReplica(t, mr, 50*time.Millisecond)
	time.Sleep(150 * time.Millisecond)

	for i := 0; i < 3; i++ {
		require.NoError(t, a.Notify(context.Background()))
	}
	require.Eventually(t, func() bool { return bReloads.Load() >= 1 }, 2*time.Second, 20*time.Millisecond)
	before := bReloads.Load()

	mr.FlushAll() // counter back to zero
	time.Sleep(150 * time.Millisecond)
	settled := bReloads.Load() // a reset itself may trigger one harmless reload

	require.NoError(t, a.Notify(context.Background())) // version 1 again
	assert.Eventually(t, func() bool { return bReloads.Load() > settled }, 2*time.Second, 20*time.Millisecond,
		"a change after the reset is still applied")
	assert.GreaterOrEqual(t, settled, before)
}
