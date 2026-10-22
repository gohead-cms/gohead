// Package demo runs GoHead as a public demo: visitors sign in with a shared
// account that can look at everything in the admin and edit content
// drafts, but not change settings, and the database goes back to a
// snapshot every night.
package demo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gohead-cms/gohead/internal/accounts"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// Role is the role of the demo account.
const Role = "demo"

// Config is the demo settings.
type Config struct {
	Enabled  bool
	Username string
	Password string
	// ResetAt is the daily reset time, "HH:MM" UTC; "" never.
	ResetAt string
	// Snapshot is the file resets restore.
	Snapshot string
}

var (
	mu  sync.RWMutex
	cfg Config
)

// Configure sets the demo settings (call at startup).
func Configure(c Config) error {
	if c.Enabled {
		if c.Username == "" || c.Password == "" {
			return errors.New("demo.username and demo.password are required in demo mode")
		}
		if c.ResetAt != "" {
			if _, _, err := parseClock(c.ResetAt); err != nil {
				return err
			}
		}
	}
	mu.Lock()
	cfg = c
	mu.Unlock()
	return nil
}

func current() Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// Enabled reports whether the instance is a demo.
func Enabled() bool { return current().Enabled }

// IsDemo reports whether role is the demo account's, in demo mode.
func IsDemo(role string) bool { return role == Role && current().Enabled }

// Info is what the login page shows: how to sign in, when it resets.
type Info struct {
	Username string     `json:"username"`
	Password string     `json:"password"`
	ResetAt  string     `json:"resetAt,omitempty"`
	NextAt   *time.Time `json:"nextReset,omitempty"`
}

// PublicInfo returns the demo's sign-in details.
func PublicInfo(now time.Time) Info {
	c := current()
	info := Info{Username: c.Username, Password: c.Password, ResetAt: c.ResetAt}
	if next, ok := NextReset(now); ok {
		info.NextAt = &next
	}
	return info
}

// EnsureAccount creates or updates the demo role and account: content
// read, create and update everywhere (no publish, no delete), media read.
func EnsureAccount() error {
	c := current()
	perms := auth.Permissions{
		Content: map[string][]string{auth.AnyType: {auth.ActionRead, auth.ActionCreate, auth.ActionUpdate}},
		Media:   []string{auth.ActionRead},
	}
	const desc = "Visitors of the public demo: read everything, edit drafts"
	if _, err := storage.CreateCustomRole(Role, desc, perms); err != nil {
		if !errors.Is(err, storage.ErrRoleExists) {
			return fmt.Errorf("demo role: %w", err)
		}
		if _, err := storage.UpdateCustomRole(Role, desc, perms); err != nil {
			return fmt.Errorf("demo role: %w", err)
		}
	}
	user, err := storage.GetUserByUsername(c.Username)
	if err == nil {
		if user.Role.Name != Role {
			return fmt.Errorf("demo.username %q is an existing account with the %s role: choose another name", c.Username, user.Role.Name)
		}
		return accounts.SetPassword(c.Username, c.Password)
	}
	role, err := storage.GetRoleByName(Role)
	if err != nil {
		return err
	}
	u, err := accounts.Build(c.Username, "demo@example.com", c.Password, role)
	if err != nil {
		return fmt.Errorf("demo account: %w", err)
	}
	return storage.CreateUser(u)
}

// ---- Snapshot and reset (SQLite)

// ErrNotSQLite is returned on databases other than SQLite: the demo reset
// copies SQLite tables.
var ErrNotSQLite = errors.New("demo snapshots need a SQLite database")

// ErrNoSnapshot is returned when no snapshot was taken yet.
var ErrNoSnapshot = errors.New("no demo snapshot yet: take one with POST /admin/demo/snapshot")

func checkSQLite() error {
	if database.DB == nil || database.DB.Dialector.Name() != "sqlite" {
		return ErrNotSQLite
	}
	return nil
}

func sqlString(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// Snapshot saves the database as it is now: resets go back to it.
func Snapshot(ctx context.Context) (time.Time, error) {
	if err := checkSQLite(); err != nil {
		return time.Time{}, err
	}
	path := current().Snapshot
	if path == "" {
		return time.Time{}, errors.New("demo.snapshot is not set")
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	if err := database.DB.WithContext(ctx).Exec("VACUUM INTO " + sqlString(tmp)).Error; err != nil {
		return time.Time{}, fmt.Errorf("could not save the snapshot: %w", err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return time.Time{}, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	logger.Log.WithField("file", path).Info("Saved the demo snapshot")
	return info.ModTime(), nil
}

// SnapshotTime returns when the snapshot was taken.
func SnapshotTime() (time.Time, bool) {
	info, err := os.Stat(current().Snapshot)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

// kept are the tables a reset leaves alone: the job queue (jobs in flight)
// and the migrations applied.
var kept = map[string]bool{"jobs": true, "schema_migrations": true}

// Restore puts every table back as in the snapshot, in one transaction.
// Columns added since the snapshot keep their defaults.
func Restore(ctx context.Context) error {
	if err := checkSQLite(); err != nil {
		return err
	}
	path := current().Snapshot
	if _, err := os.Stat(path); err != nil {
		return ErrNoSnapshot
	}
	sqlDB, err := database.DB.DB()
	if err != nil {
		return err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "ATTACH DATABASE "+sqlString(path)+" AS snap"); err != nil {
		return fmt.Errorf("could not open the snapshot: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), "DETACH DATABASE snap") }()

	tables, err := tableNames(ctx, conn, "snap")
	if err != nil {
		return err
	}
	existing, err := tableNames(ctx, conn, "main")
	if err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, t := range tables {
		if kept[t] || !contains(existing, t) {
			continue
		}
		cols, err := commonColumns(ctx, tx, t)
		if err != nil {
			return err
		}
		q := quoteIdent(t)
		if _, err := tx.ExecContext(ctx, "DELETE FROM main."+q); err != nil {
			return fmt.Errorf("reset %s: %w", t, err)
		}
		if len(cols) == 0 {
			continue
		}
		list := strings.Join(cols, ", ")
		if _, err := tx.ExecContext(ctx, "INSERT INTO main."+q+" ("+list+") SELECT "+list+" FROM snap."+q); err != nil {
			return fmt.Errorf("reset %s: %w", t, err)
		}
	}
	return tx.Commit()
}

// tableNames lists the tables of an attached schema ("main", "snap").
func tableNames(ctx context.Context, conn *sql.Conn, schema string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, "SELECT name FROM "+schema+".sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// columns lists a table's columns in a schema.
func columns(ctx context.Context, tx *sql.Tx, schema, table string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT name FROM pragma_table_info(?, ?)", table, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// commonColumns are the columns of a table both in the snapshot and now,
// quoted.
func commonColumns(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	snap, err := columns(ctx, tx, "snap", table)
	if err != nil {
		return nil, err
	}
	now, err := columns(ctx, tx, "main", table)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, c := range snap {
		if contains(now, c) {
			out = append(out, quoteIdent(c))
		}
	}
	return out, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// ---- Daily reset

func parseClock(s string) (int, int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, fmt.Errorf("demo.reset_at must be a time of day, HH:MM in UTC (e.g. 03:00), not %q", s)
	}
	return t.Hour(), t.Minute(), nil
}

// NextReset returns the next reset after now.
func NextReset(now time.Time) (time.Time, bool) {
	c := current()
	if !c.Enabled || c.ResetAt == "" {
		return time.Time{}, false
	}
	h, m, err := parseClock(c.ResetAt)
	if err != nil {
		return time.Time{}, false
	}
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next, true
}

// Reset restores the snapshot and drops the caches; after runs once it
// worked (the GraphQL schema is rebuilt there).
func Reset(ctx context.Context, after func()) error {
	if err := Restore(ctx); err != nil {
		return err
	}
	storage.ForgetCaches()
	if err := EnsureAccount(); err != nil {
		logger.Log.WithError(err).Warn("Demo reset: could not restore the demo account")
	}
	if after != nil {
		after()
	}
	logger.Log.Info("The demo was reset to its snapshot")
	return nil
}

// Run resets the demo every day at demo.reset_at until ctx ends.
func Run(ctx context.Context, after func()) {
	for {
		next, ok := NextReset(time.Now())
		if !ok {
			return
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if err := Reset(ctx, after); err != nil {
			logger.Log.WithError(err).Error("The demo reset failed")
		}
	}
}
