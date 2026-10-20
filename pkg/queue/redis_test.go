package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisDriver(t *testing.T) {
	mr := miniredis.RunT(t)
	opt := asynq.RedisClientOpt{Addr: mr.Addr()}
	c := NewRedisClient(opt)
	defer c.Close()
	ctx := context.Background()

	id, err := c.Enqueue(ctx, "greet", []byte("hi"), Queue("q"), TaskID("t-1"), Retention(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "t-1", id)
	_, err = c.Enqueue(ctx, "greet", []byte("hi"), Queue("q"), TaskID("t-1"))
	assert.ErrorIs(t, err, ErrDuplicateTask)

	got := make(chan string, 4)
	mux := NewMux()
	mux.Handle("greet", func(ctx context.Context, p []byte) error {
		id, _ := GetTaskID(ctx)
		n, _ := GetRetryCount(ctx)
		got <- fmt.Sprintf("%s/%s/%d", p, id, n)
		return nil
	})
	mux.Handle("skip", func(context.Context, []byte) error { return fmt.Errorf("bad: %w", SkipRetry) })
	_, err = c.Enqueue(ctx, "skip", nil, Queue("q"), TaskID("t-2"))
	require.NoError(t, err)

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		done <- NewRedisServer(opt, ServerConfig{Concurrency: 1, Queues: map[string]int{"q": 1}}).Run(runCtx, mux)
	}()
	select {
	case v := <-got:
		assert.Equal(t, "hi/t-1/0", v)
	case <-time.After(10 * time.Second):
		t.Fatal("task not processed")
	}
	// SkipRetry archives the task instead of retrying it.
	inspector := asynq.NewInspector(opt)
	defer inspector.Close()
	require.Eventually(t, func() bool {
		info, err := inspector.GetTaskInfo("q", "t-2")
		return err == nil && info.State == asynq.TaskStateArchived
	}, 10*time.Second, 50*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}
