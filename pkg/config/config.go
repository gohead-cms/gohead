// pkg/config/config.go
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"
)

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers" yaml:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials" yaml:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age" yaml:"max_age"`
}

// Config holds all application settings.
type Config struct {
	LogLevel          string `mapstructure:"log_level"`
	Mode              string `yaml:"gin_log_level"` // For Gin framework logging
	TelemetryEnabled  bool   `mapstructure:"telemetry_enabled"`
	JWTSecret         string `mapstructure:"jwt_secret"`
	DatabaseURL       string `mapstructure:"database_url"`
	ServerPort        string `mapstructure:"server_port"`
	MinPasswordLength int    `json:"min_password_length" yaml:"min_password_length"`

	// CORS settings
	CORS CORSConfig `mapstructure:"cors"`
}

// LoadConfig loads the configuration from file and environment variables.
func LoadConfig(configPath string) (Config, error) {
	var cfg Config

	// Set default values
	viper.SetDefault("log_level", "info")
	viper.SetDefault("telemetry_enabled", true)
	viper.SetDefault("jwt_secret", "your-secret-key")
	viper.SetDefault("database_url", "sqlite://gohead-cms.db")
	viper.SetDefault("server_port", "8080")
	viper.SetDefault("min_password_length", 6)

	// CORS default values
	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers"})
	viper.SetDefault("cors.allow_credentials", true)
	viper.SetDefault("cors.max_age", 86400)

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
	viper.SetEnvPrefix("GOHEAD")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Unmarshal the config into the struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// LoadTestConfig loads a test configuration file.
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
