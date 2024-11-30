// pkg/logger/logger.go
package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func InitLogger(logLevel string) {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.JSONFormatter{})

	level, err := logrus.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		Log.Warn("Invalid log level. Using 'info' level as default.")
		level = logrus.InfoLevel
	}
	Log.SetLevel(level)
}
