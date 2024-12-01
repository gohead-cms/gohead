// pkg/config/config.go
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel         string `mapstructure:"log_level"`
	Mode             string `yaml:"gin_log_level"` // For Gin framework logging
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

	// Set the config file path
	viper.SetConfigFile(configPath)

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return cfg, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Override with environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CMS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Unmarshal the config into the struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

func LoadTestConfig() (Config, error) {
	return LoadConfig("//Users/nbo/go/src/gohead/config_test.yaml")
}

// SaveConfig writes the provided Config struct to a YAML file.
func SaveConfig(cfg *Config, filePath string) error {
	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Marshal the config into YAML format
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config to YAML: %w", err)
	}

	return nil
}
