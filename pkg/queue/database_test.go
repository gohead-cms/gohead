package queue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func init() { logger.InitLogger("error") }

// testDB returns a database with an empty jobs table: a SQLite file, or
// the database of GOHEAD_TEST_DATABASE_URL (PostgreSQL, MySQL) when set.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	var db *gorm.DB
	var err error
	if url := os.Getenv("GOHEAD_TEST_DATABASE_URL"); url != "" {
		db, err = database.InitDatabase(url, gormlogger.Silent)
		require.NoError(t, err)
		require.NoError(t, db.Migrator().DropTable(&Job{}))
	} else {
		path := filepath.Join(t.TempDir(), "q.db")
		db, err = gorm.Open(gormlite.Open(database.SQLiteDSN(path)), &gorm.Config{
			TranslateError: true, Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		require.NoError(t, err)
	}
	require.NoError(t, db.AutoMigrate(&Job{}))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func startServer(t *testing.T, db *gorm.DB, mux *Mux, queues map[string]int) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	srv := NewDatabaseServer(db, DatabaseServerConfig{
		ServerConfig: ServerConfig{Concurrency: 2, Queues: queues},
		PollInterval: 20 * time.Millisecond, ShutdownTimeout: 200 * time.Millisecond,
	})
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx, mux) }()
	var once sync.Once
	stop = func() {
		once.Do(func() {
			cancel()
			require.NoError(t, <-done)
		})
	}
	t.Cleanup(stop)
	return stop
}

func jobs(t *testing.T, db *gorm.DB) []Job {
	var out []Job
	require.NoError(t, db.Order("id").Find(&out).Error)
	return out
}

func TestDatabaseEnqueueAndProcess(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	got := make(chan string, 1)
	mux := NewMux()
	mux.Handle("greet", func(ctx context.Context, p []byte) error {
		id, ok := GetTaskID(ctx)
		require.True(t, ok)
		n, _ := GetRetryCount(ctx)
		got <- fmt.Sprintf("%s/%s/%d", p, id, n)
		return nil
	})
	startServer(t, db, mux, map[string]int{"default": 1})

	id, err := c.Enqueue(context.Background(), "greet", []byte("hi"))
	require.NoError(t, err)
	select {
	case v := <-got:
		assert.Equal(t, "hi/"+id+"/0", v)
	case <-time.After(5 * time.Second):
		t.Fatal("task not processed")
	}
	assert.Eventually(t, func() bool { return len(jobs(t, db)) == 0 }, 2*time.Second, 10*time.Millisecond, "finished task without retention is deleted")
}

func TestDatabaseTaskIDDeduplication(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	ctx := context.Background()

	id, err := c.Enqueue(ctx, "x", nil, TaskID("wh-1"), Retention(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "wh-1", id)
	_, err = c.Enqueue(ctx, "x", nil, TaskID("wh-1"))
	assert.ErrorIs(t, err, ErrDuplicateTask)

	// Done but retained: still a duplicate.
	require.NoError(t, db.Model(&Job{}).Where("task_id = ?", "wh-1").
		Updates(map[string]any{"status": StatusDone, "retention_until": now().Add(time.Hour)}).Error)
	_, err = c.Enqueue(ctx, "x", nil, TaskID("wh-1"))
	assert.ErrorIs(t, err, ErrDuplicateTask)

	// Past its retention: the ID is free again.
	require.NoError(t, db.Model(&Job{}).Where("task_id = ?", "wh-1").Update("retention_until", now().Add(-time.Second)).Error)
	_, err = c.Enqueue(ctx, "x", nil, TaskID("wh-1"))
	require.NoError(t, err)
	assert.Len(t, jobs(t, db), 1)

	// Tasks without an ID never conflict.
	_, err = c.Enqueue(ctx, "x", nil)
	require.NoError(t, err)
	_, err = c.Enqueue(ctx, "x", nil)
	require.NoError(t, err)
}

func TestDatabaseRetryThenSuccess(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	var calls atomic.Int32
	retries := make(chan int, 2)
	mux := NewMux()
	mux.Handle("flaky", func(ctx context.Context, _ []byte) error {
		n, _ := GetRetryCount(ctx)
		retries <- n
		if calls.Add(1) == 1 {
			return errors.New("boom")
		}
		return nil
	})
	startServer(t, db, mux, map[string]int{"default": 1})
	_, err := c.Enqueue(context.Background(), "flaky", nil, Retention(time.Hour))
	require.NoError(t, err)

	assert.Equal(t, 0, <-retries)
	var j Job
	require.Eventually(t, func() bool {
		db.First(&j)
		return j.Status == StatusPending && j.Retried == 1
	}, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, "boom", j.LastError)
	assert.True(t, j.RunAt.After(now().Add(10*time.Second)), "retried with backoff")

	// Make the retry due.
	require.NoError(t, db.Model(&Job{}).Where("id = ?", j.ID).Update("run_at", now()).Error)
	select {
	case n := <-retries:
		assert.Equal(t, 1, n)
	case <-time.After(5 * time.Second):
		t.Fatal("retry not processed")
	}
	require.Eventually(t, func() bool { db.First(&j); return j.Status == StatusDone }, 2*time.Second, 10*time.Millisecond)
	assert.NotNil(t, j.RetentionUntil)
}

func TestDatabaseFailures(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	mux := NewMux()
	mux.Handle("skip", func(context.Context, []byte) error { return fmt.Errorf("bad input: %w", SkipRetry) })
	mux.Handle("noretry", func(context.Context, []byte) error { return errors.New("nope") })
	mux.Handle("panic", func(context.Context, []byte) error { panic("kaboom") })
	startServer(t, db, mux, map[string]int{"default": 1})
	ctx := context.Background()
	_, err := c.Enqueue(ctx, "skip", nil)
	require.NoError(t, err)
	_, err = c.Enqueue(ctx, "noretry", nil, MaxRetry(0))
	require.NoError(t, err)
	_, err = c.Enqueue(ctx, "panic", nil, MaxRetry(0))
	require.NoError(t, err)
	_, err = c.Enqueue(ctx, "unknown", nil, MaxRetry(0))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		for _, j := range jobs(t, db) {
			if j.Status != StatusFailed {
				return false
			}
		}
		return true
	}, 5*time.Second, 10*time.Millisecond)
	js := jobs(t, db)
	require.Len(t, js, 4)
	assert.Contains(t, js[0].LastError, "bad input")
	assert.Equal(t, 0, js[0].Retried, "SkipRetry does not retry")
	assert.Equal(t, "nope", js[1].LastError)
	assert.Contains(t, js[2].LastError, "panic: kaboom")
	assert.Contains(t, js[3].LastError, "handler not found")
	for _, j := range js {
		require.NotNil(t, j.RetentionUntil)
		assert.True(t, j.RetentionUntil.After(now().Add(FailedRetention-time.Minute)))
	}
}

func TestDatabaseQueuePriorityAndFilter(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := c.Enqueue(ctx, "t", []byte("low"), Queue("low"))
		require.NoError(t, err)
	}
	_, err := c.Enqueue(ctx, "t", []byte("high"), Queue("high"))
	require.NoError(t, err)
	_, err = c.Enqueue(ctx, "t", []byte("other"), Queue("other"))
	require.NoError(t, err)

	srv := NewDatabaseServer(db, DatabaseServerConfig{ServerConfig: ServerConfig{Queues: map[string]int{"low": 1, "high": 10}}})
	j, err := srv.claim(ctx, srv.queueOrder())
	require.NoError(t, err)
	require.NotNil(t, j)
	assert.Equal(t, "high", string(j.Payload))
	for i := 0; i < 3; i++ {
		j, err = srv.claim(ctx, srv.queueOrder())
		require.NoError(t, err)
		require.NotNil(t, j)
		assert.Equal(t, "low", string(j.Payload))
	}
	j, err = srv.claim(ctx, srv.queueOrder())
	require.NoError(t, err)
	assert.Nil(t, j, "queues the server does not process are left alone")
}

func TestDatabaseConcurrentClaimsTakeEachTaskOnce(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	ctx := context.Background()
	const n = 30
	for i := 0; i < n; i++ {
		_, err := c.Enqueue(ctx, "t", nil)
		require.NoError(t, err)
	}
	var mu sync.Mutex
	seen := map[uint64]int{}
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		srv := NewDatabaseServer(db, DatabaseServerConfig{ServerConfig: ServerConfig{Queues: map[string]int{"default": 1}}})
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				j, err := srv.claim(ctx, srv.queueOrder())
				if err != nil {
					continue // SQLite busy under contention: try again
				}
				if j == nil {
					return
				}
				mu.Lock()
				seen[j.ID]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Len(t, seen, n)
	for id, k := range seen {
		assert.Equal(t, 1, k, "task %d claimed %d times", id, k)
	}
}

func TestDatabaseStaleTasksAreRecovered(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	ctx := context.Background()
	_, err := c.Enqueue(ctx, "t", nil)
	require.NoError(t, err)
	_, err = c.Enqueue(ctx, "t", nil, MaxRetry(0))
	require.NoError(t, err)
	// Both claimed by a server that died.
	past := now().Add(-time.Minute)
	require.NoError(t, db.Model(&Job{}).Where("1 = 1").Updates(map[string]any{"status": StatusRunning, "locked_by": "dead", "locked_until": past}).Error)
	// A finished task past its retention.
	require.NoError(t, db.Create(&Job{Queue: "default", Type: "t", Status: StatusDone, RunAt: past, RetentionUntil: &past}).Error)

	srv := NewDatabaseServer(db, DatabaseServerConfig{ServerConfig: ServerConfig{Queues: map[string]int{"default": 1}}})
	srv.maintain(ctx, &sync.Map{})
	js := jobs(t, db)
	require.Len(t, js, 2)
	assert.Equal(t, StatusPending, js[0].Status)
	assert.Equal(t, 1, js[0].Retried)
	assert.Equal(t, StatusFailed, js[1].Status)
}

func TestDatabaseLeaseIsExtendedForLongTasks(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	release := make(chan struct{})
	started := make(chan struct{})
	mux := NewMux()
	mux.Handle("long", func(context.Context, []byte) error { close(started); <-release; return nil })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewDatabaseServer(db, DatabaseServerConfig{
		ServerConfig: ServerConfig{Concurrency: 1, Queues: map[string]int{"default": 1}},
		PollInterval: 20 * time.Millisecond,
	})
	srv.lease = 150 * time.Millisecond
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx, mux) }()

	_, err := c.Enqueue(context.Background(), "long", nil)
	require.NoError(t, err)
	<-started
	time.Sleep(500 * time.Millisecond) // several leases
	js := jobs(t, db)
	require.Len(t, js, 1)
	assert.Equal(t, StatusRunning, js[0].Status)
	assert.Equal(t, 0, js[0].Retried, "not taken for a dead worker's task")
	close(release)
	require.Eventually(t, func() bool { return len(jobs(t, db)) == 0 }, 2*time.Second, 10*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

func TestDatabaseShutdownRequeuesInterruptedTasks(t *testing.T) {
	db := testDB(t)
	c := NewDatabaseClient(db)
	started := make(chan struct{})
	mux := NewMux()
	mux.Handle("stuck", func(ctx context.Context, _ []byte) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	stop := startServer(t, db, mux, map[string]int{"default": 1})
	_, err := c.Enqueue(context.Background(), "stuck", nil)
	require.NoError(t, err)
	<-started
	stop()
	js := jobs(t, db)
	require.Len(t, js, 1)
	assert.Equal(t, StatusPending, js[0].Status)
	assert.Equal(t, 0, js[0].Retried)
	assert.Empty(t, js[0].LockedBy)
}

func TestApplyDefaults(t *testing.T) {
	o := Apply(nil)
	assert.Equal(t, "default", o.Queue)
	assert.Equal(t, DefaultMaxRetry, o.MaxRetry)
	assert.Equal(t, DefaultTimeout, o.Timeout)
	o = Apply([]Option{MaxRetry(0), Queue("q")})
	assert.Equal(t, 0, o.MaxRetry)
	assert.Equal(t, "q", o.Queue)
	assert.GreaterOrEqual(t, Backoff(1), 16*time.Second)
}
