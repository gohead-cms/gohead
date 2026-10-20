package queue

import (
	"context"
	"errors"
	"fmt"

	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/hibiken/asynq"
)

// RedisClient enqueues tasks on Redis with asynq.
type RedisClient struct{ c *asynq.Client }

// NewRedisClient returns a client of the Redis described by opt.
func NewRedisClient(opt asynq.RedisConnOpt) *RedisClient {
	return &RedisClient{c: asynq.NewClient(opt)}
}

// Enqueue implements Client.
func (r *RedisClient) Enqueue(ctx context.Context, taskType string, payload []byte, opts ...Option) (string, error) {
	o := Apply(opts)
	aopts := []asynq.Option{asynq.Queue(o.Queue), asynq.MaxRetry(o.MaxRetry), asynq.Timeout(o.Timeout)}
	if o.TaskID != "" {
		aopts = append(aopts, asynq.TaskID(o.TaskID))
	}
	if o.Retention > 0 {
		aopts = append(aopts, asynq.Retention(o.Retention))
	}
	info, err := r.c.EnqueueContext(ctx, asynq.NewTask(taskType, payload), aopts...)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return "", ErrDuplicateTask
	}
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// Close implements Client.
func (r *RedisClient) Close() error { return r.c.Close() }

// RedisServer processes tasks from Redis with asynq.
type RedisServer struct {
	opt asynq.RedisConnOpt
	cfg ServerConfig
}

// NewRedisServer returns a server of the Redis described by opt.
func NewRedisServer(opt asynq.RedisConnOpt, cfg ServerConfig) *RedisServer {
	return &RedisServer{opt: opt, cfg: cfg}
}

// Run implements Server.
func (r *RedisServer) Run(ctx context.Context, mux *Mux) error {
	srv := asynq.NewServer(r.opt, asynq.Config{
		Concurrency: r.cfg.Concurrency,
		Queues:      r.cfg.Queues,
		Logger:      &logger.AsynqLoggerAdapter{},
	})
	amux := asynq.NewServeMux()
	for _, t := range mux.Types() {
		h, _ := mux.Handler(t)
		amux.HandleFunc(t, func(ctx context.Context, task *asynq.Task) error {
			id, _ := asynq.GetTaskID(ctx)
			retry, _ := asynq.GetRetryCount(ctx)
			err := h(WithTaskInfo(ctx, id, retry), task.Payload())
			if err != nil && errors.Is(err, SkipRetry) {
				return fmt.Errorf("%w: %w", err, asynq.SkipRetry)
			}
			return err
		})
	}
	if err := srv.Start(amux); err != nil {
		return err
	}
	<-ctx.Done()
	srv.Shutdown()
	return nil
}
