package database

import (
	"fmt"
	"net/url"
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

// SQLiteMaxConns caps the connections to a SQLite file.
var SQLiteMaxConns = 4

// sqliteTimeFormat is how times are stored in SQLite: the layout of the
// cgo driver earlier GoHead versions used, so existing databases keep one
// format (and their times compare as text in SQL).
const sqliteTimeFormat = "2006-01-02 15:04:05.999999999-07:00"

// SQLiteDSN turns the path of a sqlite:// URL into a DSN of the pure-Go
// SQLite driver: WAL journal (readers do not wait for the writer), a busy
// timeout and immediate transactions instead of "database is locked"
// errors, foreign keys not enforced, and the time format. Parameters
// already in the URL are kept.
func SQLiteDSN(path string) string {
	params := url.Values{}
	params.Set("_timefmt", sqliteTimeFormat)
	// Transactions take the write lock when they begin, waiting for it
	// with the busy timeout: a deferred one that reads, then writes while
	// another connection wrote, fails at once with SQLITE_BUSY.
	params.Set("_txlock", "immediate")
	params.Add("_pragma", "busy_timeout(10000)")
	// As with the cgo driver of earlier versions (and its SQLite default).
	params.Add("_pragma", "foreign_keys(0)")
	if !strings.Contains(path, ":memory:") {
		params.Add("_pragma", "journal_mode(wal)")
		params.Add("_pragma", "synchronous(normal)")
	}
	file, query, _ := strings.Cut(path, "?")
	if !strings.HasPrefix(file, "file:") {
		file = "file:" + strings.NewReplacer("%", "%25", "#", "%23").Replace(file)
	}
	if query != "" {
		return file + "?" + query + "&" + params.Encode()
	}
	return file + "?" + params.Encode()
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
