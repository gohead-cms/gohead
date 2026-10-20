package database

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// InitDatabase initializes the database connection
func InitDatabase(databaseURL string, logLevel gormlogger.LogLevel) (*gorm.DB, error) {
	// Create a custom GORM logger
	gormLogger := logger.NewGormLogger(logLevel)

	var db *gorm.DB
	var err error
	switch {
	case strings.HasPrefix(databaseURL, "sqlite://"):
		// SQLite configuration
		dbPath := strings.TrimPrefix(databaseURL, "sqlite://")
		db, err = gorm.Open(gormlite.Open(SQLiteDSN(dbPath)), &gorm.Config{
			Logger:         gormLogger,
			TranslateError: true,
		})
		// Every new connection to ":memory:" opens a separate, empty
		// database, so an in-memory DB must be pinned to one connection.
		if err == nil {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				if strings.Contains(dbPath, ":memory:") {
					sqlDB.SetMaxOpenConns(1)
				} else {
					// SQLite writes one at a time, and each connection
					// holds its own page cache: a few are enough.
					sqlDB.SetMaxOpenConns(SQLiteMaxConns)
					sqlDB.SetMaxIdleConns(2)
					_, _ = sqlDB.Exec("SELECT count(*) FROM sqlite_master") // creates -wal and -shm
					matchJournalMode(strings.TrimPrefix(strings.SplitN(dbPath, "?", 2)[0], "file:"))
				}
			}
		}

	case strings.HasPrefix(databaseURL, "mysql://"):
		// MySQL configuration
		dsn := strings.TrimPrefix(databaseURL, "mysql://")
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger:         gormLogger,
			TranslateError: true,
		})

	case strings.HasPrefix(databaseURL, "postgres://"):
		// Parse the PostgreSQL URL
		dsn, parseErr := parsePostgresURL(databaseURL)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid PostgreSQL URL: %w", parseErr)
		}

		// PostgreSQL configuration
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger:         gormLogger,
			TranslateError: true,
		})

	default:
		// Unsupported database type
		return nil, fmt.Errorf("unsupported database type: %s", databaseURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Assign the initialized database to the global DB
	DB = db
	return DB, nil
}

// matchJournalMode gives the -wal and -shm files of a private database the
// database's own permissions (the driver creates them 0644).
func matchJournalMode(dbFile string) {
	info, err := os.Stat(dbFile)
	if err != nil || info.Mode().Perm()&0o077 != 0 {
		return
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Chmod(dbFile+suffix, info.Mode().Perm())
	}
}

// SQLiteMaxConns caps the connections to a SQLite file.
var SQLiteMaxConns = 4

// sqliteTimeFormat is how times are stored in SQLite: the layout of the
// cgo driver earlier GoHead versions used, so existing databases keep one
// format (and their times compare as text in SQL).
const sqliteTimeFormat = "2006-01-02 15:04:05.999999999-07:00"

// SQLiteDSN turns the path of a sqlite:// URL into a DSN of the pure-Go
// SQLite driver: WAL journal (readers do not wait for the writer), a busy
// timeout and immediate transactions instead of "database is locked"
// errors, foreign keys not enforced, and the time format. Settings given in
// the URL (_pragma=…, _txlock, _timefmt) win over these defaults.
func SQLiteDSN(path string) string {
	file, query, _ := strings.Cut(path, "?")
	user, _ := url.ParseQuery(query)
	setPragmas := map[string]bool{}
	for _, p := range user["_pragma"] {
		name, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(p)), "(")
		name, _, _ = strings.Cut(name, "=")
		setPragmas[strings.TrimSpace(name)] = true
	}
	params := url.Values{}
	if !user.Has("_timefmt") {
		params.Set("_timefmt", sqliteTimeFormat)
	}
	// Transactions take the write lock when they begin, waiting for it
	// with the busy timeout: a deferred one that reads, then writes while
	// another connection wrote, fails at once with SQLITE_BUSY.
	if !user.Has("_txlock") {
		params.Set("_txlock", "immediate")
	}
	pragma := func(name, value string) {
		if !setPragmas[name] {
			params.Add("_pragma", name+"("+value+")")
		}
	}
	pragma("busy_timeout", "10000")
	// As with the cgo driver of earlier versions (and its SQLite default).
	pragma("foreign_keys", "0")
	if !strings.Contains(file, ":memory:") {
		pragma("journal_mode", "wal")
		pragma("synchronous", "normal")
	}
	if !strings.HasPrefix(file, "file:") {
		file = "file:" + strings.NewReplacer("%", "%25", "#", "%23").Replace(file)
	}
	dsn := file + "?" + params.Encode()
	if query != "" {
		dsn += "&" + query
	}
	return dsn
}

// parsePostgresURL parses a PostgreSQL URL into a DSN string
func parsePostgresURL(databaseURL string) (string, error) {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	user := parsedURL.User.Username()
	password, _ := parsedURL.User.Password()
	dbname := strings.TrimPrefix(parsedURL.Path, "/")
	sslmode := parsedURL.Query().Get("sslmode")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
	return dsn, nil
}
