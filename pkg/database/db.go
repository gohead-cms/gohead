package database

import (
	"fmt"
	"net/url"
	"strings"

	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
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

	if strings.HasPrefix(databaseURL, "sqlite://") {
		// SQLite configuration
		dbPath := strings.TrimPrefix(databaseURL, "sqlite://")
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: gormLogger,
		})
	} else if strings.HasPrefix(databaseURL, "mysql://") {
		// MySQL configuration
		dsn := strings.TrimPrefix(databaseURL, "mysql://")
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: gormLogger,
		})
	} else if strings.HasPrefix(databaseURL, "postgres://") {
		// Parse the PostgreSQL URL
		dsn, parseErr := parsePostgresURL(databaseURL)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid PostgreSQL URL: %w", parseErr)
		}

		// PostgreSQL configuration
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: gormLogger,
		})
	} else {
		return nil, fmt.Errorf("unsupported database type: %s", databaseURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Assign the initialized database to the global DB
	DB = db
	return DB, nil
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
