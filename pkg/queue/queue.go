// Package queue runs GoHead's background jobs (collection events, agent
// runs, webhook deliveries) behind one interface, with two drivers:
//
//   - redis: asynq on Redis, for deployments with several processes or
//     replicas (the default);
//   - database: a jobs table in GoHead's own database, polled by in-process
//     workers. No extra service: suited to a single process (gohead
//     standalone) or a few, on small machines.
package queue

import (
	"context"
	"errors"
	"time"
)

// Driver names.
const (
	DriverRedis    = "redis"
	DriverDatabase = "database"
)

// Defaults shared by the drivers (asynq's).
const (
	DefaultMaxRetry = 25
	DefaultTimeout  = 30 * time.Minute
)

// ErrDuplicateTask is returned when a task with the same TaskID is queued,
// running, or kept for its retention period.
var ErrDuplicateTask = errors.New("a task with this ID already exists")

// SkipRetry, wrapped in a handler's error, fails the task without retrying.
var SkipRetry = errors.New("skip retry")

// Options of an enqueued task.
type Options struct {
	Queue     string
	TaskID    string        // deduplication key; empty: none
	MaxRetry  int           // retries after the first attempt; <0: DefaultMaxRetry
	Timeout   time.Duration // per attempt; 0: DefaultTimeout
	Retention time.Duration // keep the finished task (and its ID) this long
}

// Option sets an Option.
type Option func(*Options)

func Queue(name string) Option         { return func(o *Options) { o.Queue = name } }
func TaskID(id string) Option          { return func(o *Options) { o.TaskID = id } }
func MaxRetry(n int) Option            { return func(o *Options) { o.MaxRetry = n } }
func Timeout(d time.Duration) Option   { return func(o *Options) { o.Timeout = d } }
func Retention(d time.Duration) Option { return func(o *Options) { o.Retention = d } }

// Apply returns the options with defaults.
func Apply(opts []Option) Options {
	o := Options{Queue: "default", MaxRetry: -1}
	for _, f := range opts {
		f(&o)
	}
	if o.MaxRetry < 0 {
		o.MaxRetry = DefaultMaxRetry
	}
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	return o
}

// Client enqueues tasks.
type Client interface {
	// Enqueue queues a task and returns its ID.
	Enqueue(ctx context.Context, taskType string, payload []byte, opts ...Option) (string, error)
	Close() error
}

// HandlerFunc processes a task. Returning an error retries it later (with
// backoff) unless it wraps SkipRetry or the retries are exhausted.
type HandlerFunc func(ctx context.Context, payload []byte) error

// Mux maps task types to handlers.
type Mux struct{ handlers map[string]HandlerFunc }

// NewMux returns an empty Mux.
func NewMux() *Mux { return &Mux{handlers: map[string]HandlerFunc{}} }

// Handle registers the handler of a task type.
func (m *Mux) Handle(taskType string, h HandlerFunc) { m.handlers[taskType] = h }

// Handler returns the handler of a task type.
func (m *Mux) Handler(taskType string) (HandlerFunc, bool) {
	h, ok := m.handlers[taskType]
	return h, ok
}

// Types lists the registered task types.
func (m *Mux) Types() []string {
	out := make([]string, 0, len(m.handlers))
	for t := range m.handlers {
		out = append(out, t)
	}
	return out
}

// ServerConfig configures a Server.
type ServerConfig struct {
	// Concurrency is the number of tasks processed at once.
	Concurrency int
	// Queues to process, with their priority: queues are looked at in a
	// random order weighted by priority, so busy queues do not starve the
	// others.
	Queues map[string]int
	// StrictPriority always empties higher priority queues first.
	StrictPriority bool
}

// Server processes tasks until its context ends.
type Server interface {
	Run(ctx context.Context, mux *Mux) error
}

type taskInfoKey struct{}

type taskInfo struct {
	id    string
	retry int
}

// WithTaskInfo is used by drivers to pass a task's ID and retry count to
// its handler.
func WithTaskInfo(ctx context.Context, id string, retry int) context.Context {
	return context.WithValue(ctx, taskInfoKey{}, taskInfo{id: id, retry: retry})
}

// GetTaskID returns the ID of the task being processed.
func GetTaskID(ctx context.Context) (string, bool) {
	ti, ok := ctx.Value(taskInfoKey{}).(taskInfo)
	return ti.id, ok
}

// GetRetryCount returns how many times the task was retried (0 on the
// first attempt).
func GetRetryCount(ctx context.Context) (int, bool) {
	ti, ok := ctx.Value(taskInfoKey{}).(taskInfo)
	return ti.retry, ok
}
