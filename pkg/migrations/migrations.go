// Package migrations versions the database schema.
//
// Each Migration runs once per database, in ID order, and is recorded in the
// schema_migrations table. A database lock (PostgreSQL advisory lock, MySQL
// GET_LOCK) serializes runners, so API replicas, init containers and
// `gohead migrate` can start at the same time safely.
//
// Adding a migration: append to All with the next ID. Write it so it also
// works on a fresh database where 0001_baseline already created the latest
// models (check with tx.Migrator().HasColumn/HasTable before changing), and
// give it a Down when it can be undone. Never edit a released migration.
package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/gohead-cms/gohead/internal/models"
	agents "github.com/gohead-cms/gohead/internal/models/agents"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// Migration is one versioned schema change.
type Migration struct {
	ID          string // sortable, e.g. "0002_item_locale"
	Description string
	Up          func(tx *gorm.DB) error
	Down        func(tx *gorm.DB) error // nil: cannot be undone
}

// All lists the migrations in order.
var All = []Migration{
	{
		ID:          "0001_baseline",
		Description: "Tables of GoHead 0.5 (creates them, or brings databases from earlier versions up to date)",
		Up:          baseline,
	},
	{
		ID:          "0002_revisions",
		Description: "Content revisions (history of every entry change)",
		Up:          func(tx *gorm.DB) error { return tx.AutoMigrate(&revision0002{}) },
		Down:        func(tx *gorm.DB) error { return tx.Migrator().DropTable(&revision0002{}) },
	},
	{
		ID:          "0003_i18n",
		Description: "Locales, localized attributes and translations",
		Up:          i18nUp,
		Down:        i18nDown,
	},
	{
		ID:          "0004_agent_reviews",
		Description: "Agent proposals awaiting approval, approval setting, test runs",
		Up:          agentsV2Up,
		Down:        agentsV2Down,
	},
	{
		ID:          "0005_public_role",
		Description: "Rename a custom role named public, now the role of anonymous visitors",
		Up:          publicRoleUp,
		Down:        publicRoleDown,
	},
	{
		ID:          "0006_jobs",
		Description: "Jobs table of the database job queue (GoHead without Redis)",
		Up:          jobsUp,
		Down:        jobsDown,
	},
}

// baseline creates or updates every table from the models, as GoHead did at
// each start before versioned migrations.
func baseline(tx *gorm.DB) error {
	return tx.AutoMigrate(
		&models.UserRole{},
		&models.Collection{},
		&models.Attribute{},
		&models.Component{},
		&models.ComponentAttribute{},
		&models.Singleton{},
		&models.SingleItem{},
		&models.Item{},
		&models.Media{},
		&models.APIToken{},
		&models.Webhook{},
		&models.WebhookDelivery{},
		&models.User{},
		&agents.Agent{},
		&agents.AgentMessage{},
		&agents.AgentRun{},
		&agents.AgentRunStep{},
	)
}

// record is a row of schema_migrations.
type record struct {
	ID          string `gorm:"primaryKey;size:191"`
	Description string `gorm:"size:255"`
	AppliedAt   time.Time
}

func (record) TableName() string { return "schema_migrations" }

// State is a migration and whether it ran.
type State struct {
	ID          string
	Description string
	AppliedAt   *time.Time // nil: pending
	Reversible  bool
}

// ErrPending is returned by Check when migrations have not run.
var ErrPending = errors.New("database schema is not up to date")

// MigrateDatabase applies every pending migration. Kept for callers (and
// tests) that only need an up-to-date schema.
func MigrateDatabase(db *gorm.DB) error {
	_, err := Up(db)
	return err
}

// Up applies the pending migrations and returns their IDs.
func Up(db *gorm.DB) ([]string, error) {
	var applied []string
	err := withLock(db, func(conn *gorm.DB) error {
		done, err := appliedIDs(conn)
		if err != nil {
			return err
		}
		for _, m := range sorted() {
			if _, ok := done[m.ID]; ok {
				continue
			}
			start := time.Now()
			// Not wrapped in a transaction: MySQL commits DDL implicitly,
			// and GORM's migrator does not run reliably inside one. A
			// migration that needs atomic data changes opens its own.
			if err := m.Up(conn); err != nil {
				return fmt.Errorf("migration %s failed: %w", m.ID, err)
			}
			if err := conn.Create(&record{ID: m.ID, Description: m.Description, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return fmt.Errorf("migration %s ran but could not be recorded: %w", m.ID, err)
			}
			logger.Log.WithField("migration", m.ID).WithField("ms", time.Since(start).Milliseconds()).Info("Applied database migration")
			applied = append(applied, m.ID)
		}
		return nil
	})
	return applied, err
}

// Down reverts the last `steps` applied migrations, newest first.
func Down(db *gorm.DB, steps int) ([]string, error) {
	var reverted []string
	err := withLock(db, func(conn *gorm.DB) error {
		done, err := appliedIDs(conn)
		if err != nil {
			return err
		}
		if unknown := unknownIDs(done); len(unknown) > 0 {
			return fmt.Errorf("the database has migrations %v from a newer GoHead version: revert them with that version first", unknown)
		}
		list := sorted()
		for i := len(list) - 1; i >= 0 && len(reverted) < steps; i-- {
			m := list[i]
			if _, ok := done[m.ID]; !ok {
				continue
			}
			if m.Down == nil {
				return fmt.Errorf("migration %s cannot be undone; restore a backup instead", m.ID)
			}
			if err := m.Down(conn); err != nil {
				return fmt.Errorf("reverting %s failed: %w", m.ID, err)
			}
			if err := conn.Delete(&record{ID: m.ID}).Error; err != nil {
				return fmt.Errorf("%s was reverted but is still recorded: %w", m.ID, err)
			}
			logger.Log.WithField("migration", m.ID).Info("Reverted database migration")
			reverted = append(reverted, m.ID)
		}
		return nil
	})
	return reverted, err
}

// Status lists every migration with when it was applied.
func Status(db *gorm.DB) ([]State, error) {
	done := map[string]record{}
	// Read-only: a database never migrated has no table, and everything is
	// pending.
	if db.Migrator().HasTable(&record{}) {
		var err error
		if done, err = appliedIDs(db); err != nil {
			return nil, err
		}
	}
	if unknown := unknownIDs(done); len(unknown) > 0 {
		logger.Log.WithField("migrations", unknown).Warn("The database has migrations this GoHead version does not know: it was migrated by a newer version")
	}
	var out []State
	for _, m := range sorted() {
		s := State{ID: m.ID, Description: m.Description, Reversible: m.Down != nil}
		if r, ok := done[m.ID]; ok {
			t := r.AppliedAt
			s.AppliedAt = &t
		}
		out = append(out, s)
	}
	return out, nil
}

// Check returns ErrPending (with the pending IDs) when migrations have not
// run, for processes that must not change the schema themselves.
func Check(db *gorm.DB) error {
	states, err := Status(db)
	if err != nil {
		return err
	}
	var pending []string
	for _, s := range states {
		if s.AppliedAt == nil {
			pending = append(pending, s.ID)
		}
	}
	if len(pending) > 0 {
		return fmt.Errorf("%w: pending %v (run `gohead migrate`)", ErrPending, pending)
	}
	return nil
}

// unknownIDs lists applied migrations this binary does not define.
func unknownIDs(done map[string]record) []string {
	known := map[string]bool{}
	for _, m := range All {
		known[m.ID] = true
	}
	var out []string
	for id := range done {
		if !known[id] {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func sorted() []Migration {
	list := append([]Migration(nil), All...)
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list
}

func appliedIDs(db *gorm.DB) (map[string]record, error) {
	var rows []record
	if err := db.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("could not read schema_migrations: %w", err)
	}
	out := make(map[string]record, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out, nil
}

// lockName identifies the migration lock (MySQL) and lockKey its PostgreSQL
// advisory-lock counterpart.
const (
	lockName    = "gohead_schema_migrations"
	lockKey     = int64(4_711_202_610)
	lockTimeout = 10 * time.Minute
)

// withLock runs fn while holding the migration lock. The lock lives on a
// dedicated connection (locks are per session), and fn uses the normal pool.
func withLock(db *gorm.DB, fn func(db *gorm.DB) error) error {
	// schema_migrations is created under the lock, so runners starting
	// together on an empty database don't race to create it.
	run := func() error {
		if err := db.AutoMigrate(&record{}); err != nil {
			return fmt.Errorf("could not create schema_migrations: %w", err)
		}
		return fn(db)
	}
	dialect := db.Dialector.Name()
	if dialect != "postgres" && dialect != "mysql" {
		// SQLite: no lock. Run migrations from one process (the API, or
		// `gohead migrate` while the API is stopped).
		return run()
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("could not open a connection for the migration lock: %w", err)
	}
	defer conn.Close()

	if dialect == "postgres" {
		// Blocks until the lock is free (or the context times out).
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
			return fmt.Errorf("could not take the migration lock: %w", err)
		}
		defer func() { _, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey) }()
	} else {
		var got sql.NullInt64
		if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", lockName, int(lockTimeout.Seconds())).Scan(&got); err != nil {
			return fmt.Errorf("could not take the migration lock: %w", err)
		}
		if !got.Valid || got.Int64 != 1 {
			return fmt.Errorf("timed out waiting for the migration lock (another `gohead migrate` running?)")
		}
		defer func() { _, _ = conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(?)", lockName) }()
	}
	return run()
}
