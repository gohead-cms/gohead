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
			if err := conn.Transaction(func(tx *gorm.DB) error {
				if err := m.Up(tx); err != nil {
					return err
				}
				return tx.Create(&record{ID: m.ID, Description: m.Description, AppliedAt: time.Now().UTC()}).Error
			}); err != nil {
				return fmt.Errorf("migration %s failed: %w", m.ID, err)
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
		list := sorted()
		for i := len(list) - 1; i >= 0 && len(reverted) < steps; i-- {
			m := list[i]
			if _, ok := done[m.ID]; !ok {
				continue
			}
			if m.Down == nil {
				return fmt.Errorf("migration %s cannot be undone; restore a backup instead", m.ID)
			}
			if err := conn.Transaction(func(tx *gorm.DB) error {
				if err := m.Down(tx); err != nil {
					return err
				}
				return tx.Delete(&record{ID: m.ID}).Error
			}); err != nil {
				return fmt.Errorf("reverting %s failed: %w", m.ID, err)
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
	if err := db.AutoMigrate(&record{}); err != nil {
		return nil, err
	}
	done, err := appliedIDs(db)
	if err != nil {
		return nil, err
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

// withLock runs fn on a single connection holding the migration lock.
func withLock(db *gorm.DB, fn func(conn *gorm.DB) error) error {
	if err := db.AutoMigrate(&record{}); err != nil {
		return fmt.Errorf("could not create schema_migrations: %w", err)
	}
	switch db.Dialector.Name() {
	case "postgres":
		return db.Connection(func(conn *gorm.DB) error {
			if err := conn.Exec("SELECT pg_advisory_lock(?)", lockKey).Error; err != nil {
				return fmt.Errorf("could not take the migration lock: %w", err)
			}
			defer conn.Exec("SELECT pg_advisory_unlock(?)", lockKey)
			return fn(conn)
		})
	case "mysql":
		return db.Connection(func(conn *gorm.DB) error {
			var got *int
			if err := conn.Raw("SELECT GET_LOCK(?, ?)", lockName, int(lockTimeout.Seconds())).Scan(&got).Error; err != nil {
				return fmt.Errorf("could not take the migration lock: %w", err)
			}
			if got == nil || *got != 1 {
				return fmt.Errorf("timed out waiting for the migration lock (another `gohead migrate` running?)")
			}
			defer conn.Exec("SELECT RELEASE_LOCK(?)", lockName)
			return fn(conn)
		})
	default:
		// SQLite: one process owns the file; writes are serialized anyway.
		return fn(db)
	}
}
