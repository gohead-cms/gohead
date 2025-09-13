package logger

import (
	"fmt"
)

// AsynqLoggerAdapter implements the asynq.Logger interface and routes
// logs to our application's standard logger (Logrus) with a structured "source" field.
type AsynqLoggerAdapter struct{}

// Debug logs debug-level messages from asynq.
func (l *AsynqLoggerAdapter) Debug(args ...any) {
	Log.WithField("source", "asynq").Debug(fmt.Sprint(args...))
}

// Info logs informational messages from asynq.
func (l *AsynqLoggerAdapter) Info(args ...any) {
	Log.WithField("source", "asynq").Info(fmt.Sprint(args...))
}

// Warn logs warning messages from asynq.
func (l *AsynqLoggerAdapter) Warn(args ...any) {
	Log.WithField("source", "asynq").Warn(fmt.Sprint(args...))
}

// Error logs error messages from asynq.
func (l *AsynqLoggerAdapter) Error(args ...any) {
	Log.WithField("source", "asynq").Error(fmt.Sprint(args...))
}

// Fatal logs fatal error messages from asynq and exits.
func (l *AsynqLoggerAdapter) Fatal(args ...any) {
	Log.WithField("source", "asynq").Fatal(fmt.Sprint(args...))
}
