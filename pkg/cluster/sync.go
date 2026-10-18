// Package cluster keeps state that lives in memory consistent across API
// replicas. Today that is the generated GraphQL schema: when one replica
// changes a collection, every other replica must rebuild its schema.
package cluster

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/redis/go-redis/v9"
)

const (
	// DefaultChannel carries "something changed" notifications.
	DefaultChannel = "gohead:schema:changed"
	// DefaultVersionKey is a counter bumped on every change. Replicas poll it
	// so that a notification missed during a Redis disconnect is not lost.
	DefaultVersionKey = "gohead:schema:version"
	// DefaultPollInterval bounds how stale a replica can get when pub/sub
	// messages are lost.
	DefaultPollInterval = 30 * time.Second
)

// SchemaSync broadcasts schema changes between replicas. A nil Redis client
// makes it a no-op, which is correct for a single replica.
type SchemaSync struct {
	rdb          *redis.Client
	channel      string
	versionKey   string
	pollInterval time.Duration
	instanceID   string

	mu      sync.Mutex
	version int64 // last version this replica has applied
	pending bool  // a local change could not be announced yet
}

// Option customizes a SchemaSync (mainly for tests).
type Option func(*SchemaSync)

// WithPollInterval changes how often the version key is checked.
func WithPollInterval(d time.Duration) Option { return func(s *SchemaSync) { s.pollInterval = d } }

// WithKeys changes the channel and version key names.
func WithKeys(channel, versionKey string) Option {
	return func(s *SchemaSync) { s.channel, s.versionKey = channel, versionKey }
}

// NewSchemaSync returns a SchemaSync using rdb (which may be nil).
func NewSchemaSync(rdb *redis.Client, opts ...Option) *SchemaSync {
	id := make([]byte, 8)
	_, _ = rand.Read(id)
	s := &SchemaSync{
		rdb:          rdb,
		channel:      DefaultChannel,
		versionKey:   DefaultVersionKey,
		pollInterval: DefaultPollInterval,
		instanceID:   hex.EncodeToString(id),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// InstanceID identifies this replica in notifications.
func (s *SchemaSync) InstanceID() string { return s.instanceID }

// Notify tells the other replicas that the schema changed. The caller is
// expected to have rebuilt its own schema already. If Redis is unreachable,
// the announcement is retried by Run until it succeeds, so a change made
// during an outage still reaches the other replicas.
func (s *SchemaSync) Notify(ctx context.Context) error {
	if s.rdb == nil {
		return nil
	}
	err := s.announce(ctx)
	s.mu.Lock()
	s.pending = err != nil
	s.mu.Unlock()
	return err
}

func (s *SchemaSync) announce(ctx context.Context) error {
	v, err := s.rdb.Incr(ctx, s.versionKey).Result()
	if err != nil {
		return err
	}
	s.setApplied(v)
	return s.rdb.Publish(ctx, s.channel, s.instanceID).Err()
}

func (s *SchemaSync) setApplied(v int64) {
	s.mu.Lock()
	s.version = v
	s.mu.Unlock()
}

func (s *SchemaSync) retryPending(ctx context.Context) {
	s.mu.Lock()
	pending := s.pending
	s.mu.Unlock()
	if !pending {
		return
	}
	if err := s.announce(ctx); err != nil {
		return
	}
	s.mu.Lock()
	s.pending = false
	s.mu.Unlock()
	logger.Log.Info("Schema sync: announced a change made while Redis was unavailable")
}

// Run listens for changes made by other replicas and calls reload for each,
// until ctx is cancelled. It also polls the version key, so a replica that
// missed a message (Redis restart, network blip) catches up within one poll
// interval. Run returns immediately when there is no Redis client.
func (s *SchemaSync) Run(ctx context.Context, reload func() error) {
	if s.rdb == nil {
		return
	}

	// Start from the current version: the schema was just built at startup.
	if v, err := s.currentVersion(ctx); err == nil {
		s.setApplied(v)
	} else {
		logger.Log.WithError(err).Warn("Schema sync: cannot read version; will retry")
	}

	sub := s.rdb.Subscribe(ctx, s.channel)
	defer sub.Close()
	msgs := sub.Channel()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	apply := func(reason string) {
		v, err := s.currentVersion(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Log.WithError(err).Warn("Schema sync: cannot read version")
			}
			return
		}
		// Any difference means another replica changed something. "Not
		// equal" rather than "greater" also covers a Redis that lost its
		// data and restarted the counter from zero.
		s.mu.Lock()
		stale := v != s.version
		s.mu.Unlock()
		if !stale {
			return
		}
		if err := reload(); err != nil {
			logger.Log.WithError(err).Error("Schema sync: reload failed")
			return
		}
		s.setApplied(v)
		logger.Log.WithField("version", v).WithField("reason", reason).Info("Schema reloaded from another replica's change")
	}

	logger.Log.WithField("instance", s.instanceID).Info("Schema sync started")
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-msgs:
			if !ok {
				return
			}
			if m.Payload == s.instanceID {
				continue // our own change, already applied
			}
			apply("notification")
		case <-ticker.C:
			s.retryPending(ctx)
			apply("poll")
		}
	}
}

func (s *SchemaSync) currentVersion(ctx context.Context) (int64, error) {
	v, err := s.rdb.Get(ctx, s.versionKey).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return v, err
}
