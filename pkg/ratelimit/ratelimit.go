// Package ratelimit provides fixed-window counters shared by all API
// replicas through Redis, with an in-memory fallback when Redis is not
// configured or temporarily unreachable.
package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// Counter counts events per key in fixed time windows.
type Counter interface {
	// Hit records one event for key and returns the count in the current
	// window and the time until the window resets.
	Hit(ctx context.Context, key string, window time.Duration) (count int64, reset time.Duration, err error)
	// Count returns the current count without recording an event.
	Count(ctx context.Context, key string) (count int64, reset time.Duration, err error)
	// Reset clears key.
	Reset(ctx context.Context, key string) error
}

// ---------------------------------------------------------------- Redis

// hitScript increments the counter and starts the window on the first hit,
// atomically, so concurrent replicas never leave a key without expiry.
var hitScript = redis.NewScript(`
local c = redis.call('INCR', KEYS[1])
if c == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
local t = redis.call('PTTL', KEYS[1])
if t < 0 then redis.call('PEXPIRE', KEYS[1], ARGV[1]); t = tonumber(ARGV[1]) end
return {c, t}
`)

// Redis is a Counter shared by every replica using the same Redis.
type Redis struct {
	rdb    *redis.Client
	prefix string
}

// NewRedis returns a Redis counter whose keys start with prefix.
func NewRedis(rdb *redis.Client, prefix string) *Redis { return &Redis{rdb: rdb, prefix: prefix} }

func (r *Redis) Hit(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	res, err := hitScript.Run(ctx, r.rdb, []string{r.prefix + key}, window.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, 0, err
	}
	return res[0], time.Duration(res[1]) * time.Millisecond, nil
}

func (r *Redis) Count(ctx context.Context, key string) (int64, time.Duration, error) {
	pipe := r.rdb.Pipeline()
	get := pipe.Get(ctx, r.prefix+key)
	ttl := pipe.PTTL(ctx, r.prefix+key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return 0, 0, err
	}
	n, err := get.Int64()
	if errors.Is(err, redis.Nil) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return n, max(ttl.Val(), 0), nil
}

func (r *Redis) Reset(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, r.prefix+key).Err()
}

// ---------------------------------------------------------------- memory

type entry struct {
	count   int64
	expires time.Time
}

// Memory is a per-process Counter. It is correct for a single replica and
// is used as a fallback when Redis fails.
type Memory struct {
	mu      sync.Mutex
	entries map[string]*entry
	now     func() time.Time
	hits    int
}

// NewMemory returns an empty in-memory counter.
func NewMemory() *Memory { return &Memory{entries: map[string]*entry{}, now: time.Now} }

func (m *Memory) live(key string) *entry {
	e, ok := m.entries[key]
	if ok && m.now().After(e.expires) {
		delete(m.entries, key)
		return nil
	}
	return e
}

func (m *Memory) Hit(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hits++; m.hits%1024 == 0 {
		m.sweep()
	}
	e := m.live(key)
	if e == nil {
		e = &entry{expires: m.now().Add(window)}
		m.entries[key] = e
	}
	e.count++
	return e.count, e.expires.Sub(m.now()), nil
}

func (m *Memory) Count(_ context.Context, key string) (int64, time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.live(key); e != nil {
		return e.count, e.expires.Sub(m.now()), nil
	}
	return 0, 0, nil
}

func (m *Memory) Reset(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.entries, key)
	m.mu.Unlock()
	return nil
}

// sweep drops expired entries so the map cannot grow without bound.
func (m *Memory) sweep() {
	now := m.now()
	for k, e := range m.entries {
		if now.After(e.expires) {
			delete(m.entries, k)
		}
	}
}

// ---------------------------------------------------------------- fallback

// Fallback uses primary and switches to secondary for any call where
// primary fails, so a Redis outage degrades limits to per-replica instead
// of disabling them or blocking logins.
type Fallback struct {
	Primary, Secondary Counter
	mu                 sync.Mutex
	warned             time.Time
}

func (f *Fallback) warn(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if time.Since(f.warned) > time.Minute {
		f.warned = time.Now()
		logger.Log.WithError(err).Warn("Rate limiter: Redis unavailable, using per-replica limits")
	}
}

func (f *Fallback) Hit(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	n, t, err := f.Primary.Hit(ctx, key, window)
	if err != nil {
		f.warn(err)
		return f.Secondary.Hit(ctx, key, window)
	}
	return n, t, nil
}

func (f *Fallback) Count(ctx context.Context, key string) (int64, time.Duration, error) {
	n, t, err := f.Primary.Count(ctx, key)
	if err != nil {
		f.warn(err)
		return f.Secondary.Count(ctx, key)
	}
	return n, t, nil
}

func (f *Fallback) Reset(ctx context.Context, key string) error {
	err := f.Primary.Reset(ctx, key)
	_ = f.Secondary.Reset(ctx, key)
	if err != nil {
		f.warn(err)
	}
	return nil
}

// New returns the counter to use: Redis with in-memory fallback, or memory
// only when rdb is nil.
func New(rdb *redis.Client, prefix string) Counter {
	if rdb == nil {
		return NewMemory()
	}
	return &Fallback{Primary: NewRedis(rdb, prefix), Secondary: NewMemory()}
}
