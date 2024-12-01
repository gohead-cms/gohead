package logger

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

// GormLogger wraps logrus for GORM
type GormLogger struct {
	logLevel logger.LogLevel
}

// NewGormLogger initializes a GORM logger using logrus
func NewGormLogger(logLevel logger.LogLevel) *GormLogger {
	return &GormLogger{
		logLevel: logLevel,
	}
}

// LogMode sets the log level for GORM
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &GormLogger{logLevel: level}
}

// Info logs informational messages
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		Log.WithFields(logrus.Fields{
			"source": "gorm",
			"data":   data,
		}).Info(msg)
	}
}

// Warn logs warning messages
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		Log.WithFields(logrus.Fields{
			"source": "gorm",
			"data":   data,
		}).Warn(msg)
	}
}

// Error logs error messages
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		Log.WithFields(logrus.Fields{
			"source": "gorm",
			"data":   data,
		}).Error(msg)
	}
}

// Trace logs SQL queries with execution time
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := logrus.Fields{
		"source":        "gorm",
		"elapsed":       elapsed.Seconds(),
		"rows_affected": rows,
		"sql":           sql,
	}

	if err != nil {
		fields["error"] = err
		Log.WithFields(fields).Error("GORM query error")
		return
	}

	if l.logLevel == logger.Info || (l.logLevel == logger.Warn && elapsed > time.Second) {
		Log.WithFields(fields).Info("GORM query")
	}
}
