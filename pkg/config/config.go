// pkg/config/config.go
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	LogLevel         string `mapstructure:"log_level"`
	TelemetryEnabled bool   `mapstructure:"telemetry_enabled"`
	JWTSecret        string `mapstructure:"jwt_secret"`
	DatabaseURL      string `mapstructure:"database_url"`
	ServerPort       string `mapstructure:"server_port"`
}

func LoadConfig(configPath string) (Config, error) {
	var cfg Config

	// Set default values
	viper.SetDefault("log_level", "info")
	viper.SetDefault("telemetry_enabled", true)
	viper.SetDefault("jwt_secret", "your-secret-key")
	viper.SetDefault("database_url", "sqlite://cms.db")
	viper.SetDefault("server_port", "8080")

	// Set the path to look for the config file
	viper.SetConfigFile(configPath)

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	// Override with environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CMS")
	// Replace dots with underscores in env vars
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Unmarshal the config into the struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
