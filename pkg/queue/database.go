package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	mrand "math/rand/v2"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// Job statuses of the database driver.
const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// FailedRetention is how long the database driver keeps tasks that failed
// for good (asynq's archive keeps them 90 days; a small machine keeps less).
var FailedRetention = 7 * 24 * time.Hour

// Job is a task of the database driver (table jobs).
type Job struct {
	ID             uint64  `gorm:"primaryKey"`
	TaskID         *string `gorm:"size:191;uniqueIndex"`
	Queue          string  `gorm:"size:64;not null;index:idx_jobs_claim,priority:2"`
	Type           string  `gorm:"size:128;not null"`
	Payload        []byte
	Status         string     `gorm:"size:16;not null;index:idx_jobs_claim,priority:1"`
	RunAt          time.Time  `gorm:"not null;index:idx_jobs_claim,priority:3"`
	Retried        int        `gorm:"not null;default:0"`
	MaxRetry       int        `gorm:"not null;default:0"`
	TimeoutMS      int64      `gorm:"column:timeout_ms;not null;default:0"`
	RetentionMS    int64      `gorm:"column:retention_ms;not null;default:0"`
	LockedBy       string     `gorm:"size:64"`
	LockedUntil    *time.Time `gorm:"index"`
	RetentionUntil *time.Time `gorm:"index"`
	LastError      string     `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TableName implements gorm's Tabler.
func (Job) TableName() string { return "jobs" }

// notifier wakes the database servers of this process when a task is
// queued, so they do not wait for their next poll.
var notifier = struct {
	sync.Mutex
	subs map[chan struct{}]struct{}
}{subs: map[chan struct{}]struct{}{}}

func subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	notifier.Lock()
	notifier.subs[ch] = struct{}{}
	notifier.Unlock()
	return ch
}

func unsubscribe(ch chan struct{}) {
	notifier.Lock()
	delete(notifier.subs, ch)
	notifier.Unlock()
}

func notify() {
	notifier.Lock()
	defer notifier.Unlock()
	for ch := range notifier.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func now() time.Time { return time.Now().UTC() }

// DatabaseClient queues tasks in the jobs table.
type DatabaseClient struct{ db *gorm.DB }

// NewDatabaseClient returns a client of the jobs table of db.
func NewDatabaseClient(db *gorm.DB) *DatabaseClient { return &DatabaseClient{db: db} }

// Enqueue implements Client.
func (c *DatabaseClient) Enqueue(ctx context.Context, taskType string, payload []byte, opts ...Option) (string, error) {
	o := Apply(opts)
	t := now()
	job := Job{
		Queue: o.Queue, Type: taskType, Payload: payload, Status: StatusPending, RunAt: t,
		MaxRetry: o.MaxRetry, TimeoutMS: o.Timeout.Milliseconds(), RetentionMS: o.Retention.Milliseconds(),
	}
	if o.TaskID != "" {
		id := o.TaskID
		job.TaskID = &id
	}
	db := c.db.WithContext(ctx)
	err := db.Create(&job).Error
	if err != nil && job.TaskID != nil && isDuplicate(err) {
		// A finished task past its retention no longer holds its ID, even
		// when no janitor ran since.
		res := db.Where("task_id = ? AND status IN ? AND retention_until < ?", *job.TaskID, []string{StatusDone, StatusFailed}, t).Delete(&Job{})
		if res.Error == nil && res.RowsAffected > 0 {
			job.ID = 0
			err = db.Create(&job).Error
		}
		if err != nil && isDuplicate(err) {
			return "", ErrDuplicateTask
		}
	}
	if err != nil {
		return "", err
	}
	notify()
	return job.id(), nil
}

// Close implements Client.
func (c *DatabaseClient) Close() error { return nil }

func (j *Job) id() string {
	if j.TaskID != nil {
		return *j.TaskID
	}
	return strconv.FormatUint(j.ID, 10)
}

func isDuplicate(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// Drivers whose errors gorm does not translate.
	msg := err.Error()
	for _, s := range []string{"UNIQUE constraint failed", "Duplicate entry", "duplicate key value"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// DatabaseServerConfig tunes the database driver.
type DatabaseServerConfig struct {
	ServerConfig
	// PollInterval is how often other processes' tasks and retries are
	// looked for (tasks queued by this process start at once). Default 1s.
	PollInterval time.Duration
	// ShutdownTimeout is how long running tasks may finish on shutdown
	// before they are cancelled and queued again. Default 8s.
	ShutdownTimeout time.Duration
}

// DatabaseServer processes tasks of the jobs table.
type DatabaseServer struct {
	db  *gorm.DB
	cfg DatabaseServerConfig
	id  string

	lease time.Duration // how long a claim lasts without heartbeat
}

// NewDatabaseServer returns a server of the jobs table of db.
func NewDatabaseServer(db *gorm.DB, cfg DatabaseServerConfig) *DatabaseServer {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 8 * time.Second
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return &DatabaseServer{db: db, cfg: cfg, id: hex.EncodeToString(b), lease: time.Minute}
}

// Run implements Server.
func (s *DatabaseServer) Run(ctx context.Context, mux *Mux) error {
	queues := s.queueOrder()
	if len(queues) == 0 {
		return errors.New("queue: no queues to process")
	}
	wake := subscribe()
	defer unsubscribe(wake)

	// Handlers run under their own context, cancelled ShutdownTimeout
	// after ctx ends.
	runCtx, cancelRun := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelRun()
	stopping := make(chan struct{})

	slots := make(chan struct{}, s.cfg.Concurrency)
	var wg sync.WaitGroup
	var inflight sync.Map // job ID → struct{}

	poll := time.NewTicker(s.cfg.PollInterval)
	defer poll.Stop()

	// Maintenance runs on its own, so leases stay fresh while every slot
	// is busy with long tasks.
	maintDone := make(chan struct{})
	go func() {
		defer close(maintDone)
		tick := time.NewTicker(s.lease / 3)
		defer tick.Stop()
		for {
			s.maintain(ctx, &inflight)
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()

	for {
		// Wait for a free slot.
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			goto shutdown
		}
		job, err := s.claim(ctx, queues)
		if err != nil && ctx.Err() == nil {
			logger.Log.WithError(err).Warn("Job queue: could not look for tasks")
		}
		if job == nil {
			<-slots
			select {
			case <-ctx.Done():
				goto shutdown
			case <-wake:
			case <-poll.C:
			}
			continue
		}
		inflight.Store(job.ID, struct{}{})
		wg.Add(1)
		go func(job *Job) {
			defer wg.Done()
			defer func() { <-slots }()
			defer inflight.Delete(job.ID)
			s.process(runCtx, stopping, mux, job)
		}(job)
	}

shutdown:
	close(stopping)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(s.cfg.ShutdownTimeout):
		cancelRun()
		<-done
	}
	<-maintDone
	return nil
}

// queueOrder returns the queues, highest priority first.
func (s *DatabaseServer) queueOrder() []string {
	qs := make([]string, 0, len(s.cfg.Queues))
	for q := range s.cfg.Queues {
		qs = append(qs, q)
	}
	sort.Slice(qs, func(i, j int) bool {
		pi, pj := s.cfg.Queues[qs[i]], s.cfg.Queues[qs[j]]
		if pi != pj {
			return pi > pj
		}
		return qs[i] < qs[j]
	})
	return qs
}

// claim takes the next due task, highest priority queue first.
func (s *DatabaseServer) claim(ctx context.Context, queues []string) (*Job, error) {
	db := s.db.WithContext(ctx)
	for attempt := 0; attempt < 5; attempt++ {
		t := now()
		var cand Job
		found := false
		for _, q := range queues {
			res := db.Select("id").Where("status = ? AND queue = ? AND run_at <= ?", StatusPending, q, t).
				Order("run_at, id").Limit(1).Find(&cand)
			if res.Error != nil {
				return nil, res.Error
			}
			if res.RowsAffected > 0 {
				found = true
				break
			}
		}
		if !found {
			return nil, nil
		}
		until := t.Add(s.lease)
		res := db.Model(&Job{}).Where("id = ? AND status = ?", cand.ID, StatusPending).
			Updates(map[string]any{"status": StatusRunning, "locked_by": s.id, "locked_until": until, "updated_at": t})
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			continue // another worker took it
		}
		var job Job
		if err := db.First(&job, cand.ID).Error; err != nil {
			return nil, err
		}
		return &job, nil
	}
	return nil, nil
}

func (s *DatabaseServer) process(ctx context.Context, stopping <-chan struct{}, mux *Mux, job *Job) {
	log := logger.Log.WithField("task_id", job.id()).WithField("type", job.Type)
	h, ok := mux.Handler(job.Type)
	var err error
	if !ok {
		err = fmt.Errorf("handler not found for task %q", job.Type)
	} else {
		timeout := time.Duration(job.TimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = DefaultTimeout
		}
		hctx, cancel := context.WithTimeout(WithTaskInfo(ctx, job.id(), job.Retried), timeout)
		err = safeCall(hctx, h, job.Payload)
		cancel()
	}

	t := now()
	db := s.db.WithContext(context.Background())
	mine := db.Model(&Job{}).Where("id = ? AND locked_by = ? AND status = ?", job.ID, s.id, StatusRunning)

	if err == nil {
		if job.RetentionMS > 0 {
			until := t.Add(time.Duration(job.RetentionMS) * time.Millisecond)
			err = mine.Updates(map[string]any{"status": StatusDone, "locked_by": "", "locked_until": nil, "retention_until": until, "last_error": "", "updated_at": t}).Error
		} else {
			err = db.Where("id = ? AND locked_by = ?", job.ID, s.id).Delete(&Job{}).Error
		}
		if err != nil {
			log.WithError(err).Error("Job queue: could not record a finished task")
		}
		return
	}

	select {
	case <-stopping:
		if ctx.Err() != nil {
			// Cancelled by shutdown: run it again later, as if never started.
			_ = mine.Updates(map[string]any{"status": StatusPending, "locked_by": "", "locked_until": nil, "updated_at": t}).Error
			log.Info("Job queue: task interrupted by shutdown, queued again")
			return
		}
	default:
	}

	msg := err.Error()
	if errors.Is(err, SkipRetry) || job.Retried >= job.MaxRetry {
		retention := time.Duration(job.RetentionMS) * time.Millisecond
		if retention < FailedRetention {
			retention = FailedRetention
		}
		if e := mine.Updates(map[string]any{"status": StatusFailed, "locked_by": "", "locked_until": nil, "last_error": msg, "retention_until": t.Add(retention), "updated_at": t}).Error; e != nil {
			log.WithError(e).Error("Job queue: could not record a failed task")
		}
		log.WithError(err).Warn("Job queue: task failed")
		return
	}
	next := t.Add(Backoff(job.Retried + 1))
	if e := mine.Updates(map[string]any{"status": StatusPending, "retried": job.Retried + 1, "run_at": next, "locked_by": "", "locked_until": nil, "last_error": msg, "updated_at": t}).Error; e != nil {
		log.WithError(e).Error("Job queue: could not schedule a retry")
	}
	log.WithError(err).WithField("retry_at", next).Info("Job queue: task failed, will retry")
}

func safeCall(ctx context.Context, h HandlerFunc, payload []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()
	return h(ctx, payload)
}

// Backoff is the delay before retry n (1-based), asynq's default:
// n⁴ + 15 + rand(0..30)·(n+1) seconds.
func Backoff(n int) time.Duration {
	sec := math.Pow(float64(n), 4) + 15 + float64(mrand.IntN(30)*(n+1))
	return time.Duration(sec) * time.Second
}

// maintain extends the leases of this server's running tasks, queues again
// the tasks of servers that stopped (lease expired), and deletes finished
// tasks past their retention.
func (s *DatabaseServer) maintain(ctx context.Context, inflight *sync.Map) {
	db := s.db.WithContext(ctx)
	t := now()
	var ids []uint64
	inflight.Range(func(k, _ any) bool { ids = append(ids, k.(uint64)); return true })
	if len(ids) > 0 {
		db.Model(&Job{}).Where("id IN ? AND locked_by = ? AND status = ?", ids, s.id, StatusRunning).
			Updates(map[string]any{"locked_until": t.Add(s.lease)})
	}
	// A task whose server died counts as a failed attempt, as with asynq.
	var stale []Job
	db.Where("status = ? AND locked_until < ?", StatusRunning, t).Limit(100).Find(&stale)
	for _, j := range stale {
		upd := map[string]any{"locked_by": "", "locked_until": nil, "updated_at": t, "last_error": "the worker running the task stopped"}
		if j.Retried >= j.MaxRetry {
			upd["status"] = StatusFailed
			upd["retention_until"] = t.Add(FailedRetention)
		} else {
			upd["status"] = StatusPending
			upd["retried"] = j.Retried + 1
			upd["run_at"] = t
		}
		db.Model(&Job{}).Where("id = ? AND status = ? AND locked_until < ?", j.ID, StatusRunning, t).Updates(upd)
	}
	db.Where("status IN ? AND retention_until < ?", []string{StatusDone, StatusFailed}, t).Delete(&Job{})
}

// Counts returns the number of tasks by status (for health and tests).
func Counts(ctx context.Context, db *gorm.DB) (map[string]int64, error) {
	var rows []struct {
		Status string
		N      int64
	}
	err := db.WithContext(ctx).Model(&Job{}).Select("status, count(*) AS n").Group("status").Scan(&rows).Error
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Status] = r.N
	}
	return out, err
}
