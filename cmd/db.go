package cmd

import (
	"fmt"

	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openDatabase loads the configuration and connects to the database, for
// one-shot commands (migrate, admin).
func openDatabase(cfgPath string) (config.Config, *gorm.DB, error) {
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return cfg, nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	logger.InitLogger(cfg.LogLevel)
	db, err := database.InitDatabase(cfg.DatabaseURL, gormlogger.Error)
	if err != nil {
		return cfg, nil, err
	}
	return cfg, db, nil
}
