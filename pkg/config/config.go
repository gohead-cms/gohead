package config

import (
	"errors"
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"io/fs"
	"os"
	"strings"
	"time"

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

// LLMConfig holds settings for Large Language Model providers.
type LLMConfig struct {
	Provider  string `mapstructure:"provider" yaml:"provider"`
	Model     string `mapstructure:"model" yaml:"model"`
	APIKey    string `mapstructure:"api_key" yaml:"api_key"`
	APISecret string `mapstructure:"api_secret" yaml:"api_secret"`
	BaseURL   string `mapstructure:"base_url" yaml:"base_url"` // optional API endpoint override
}

// RateLimitConfig protects the authentication endpoints.
type RateLimitConfig struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
	// AuthPerMinute is the number of /auth/* requests allowed per client IP
	// per minute (0 disables it).
	AuthPerMinute int `mapstructure:"auth_per_minute" yaml:"auth_per_minute"`
	// LoginFailures is the number of failed logins allowed per username and
	// client IP within LoginWindow before further attempts get 429.
	LoginFailures int           `mapstructure:"login_failures" yaml:"login_failures"`
	LoginWindow   time.Duration `mapstructure:"login_window" yaml:"login_window"`
}

// RedisConfig holds settings for the Redis connection.
type RedisConfig struct {
	Address  string `mapstructure:"address" yaml:"address"`
	Password string `mapstructure:"password" yaml:"password"`
	DB       int    `mapstructure:"db" yaml:"db"`
}

// Config holds all application settings.
type Config struct {
	LogLevel          string `mapstructure:"log_level"`
	Mode              string `mapstructure:"mode" yaml:"mode"` // Gin mode: development | production | test
	TelemetryEnabled  bool   `mapstructure:"telemetry_enabled"`
	JWTSecret         string `mapstructure:"jwt_secret"`
	DatabaseURL       string `mapstructure:"database_url"`
	ServerPort        string `mapstructure:"server_port"`
	MinPasswordLength int    `json:"min_password_length" yaml:"min_password_length"`

	// Redis settings
	Redis RedisConfig `mapstructure:"redis"`

	// CORS settings
	CORS CORSConfig `mapstructure:"cors"`

	// LLM settings
	LLM LLMConfig `mapstructure:"llm" yaml:"llm"`

	// Rate limiting for /auth/* (shared across replicas through Redis)
	RateLimit RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit"`

	// TrustedProxies lists the proxies (IPs or CIDRs) whose X-Forwarded-For
	// header is trusted to find the client IP. Empty: use the TCP peer.
	TrustedProxies []string `mapstructure:"trusted_proxies" yaml:"trusted_proxies"`
}

// LoadConfig loads the configuration from file and environment variables.
func LoadConfig(configPath string) (Config, error) {
	var cfg Config

	// Set default values
	viper.SetDefault("log_level", "info")
	viper.SetDefault("telemetry_enabled", false)
	viper.SetDefault("jwt_secret", "your-secret-key")
	viper.SetDefault("database_url", "sqlite://gohead-cms.db")
	viper.SetDefault("server_port", "8080")
	viper.SetDefault("mode", "production") // gin mode: development | production | test
	viper.SetDefault("min_password_length", 6)

	// CORS default values
	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"Origin", "Accept", "Content-Type", "Authorization", "X-Requested-With", "Webhook-Token"})
	viper.SetDefault("cors.allow_credentials", true)
	viper.SetDefault("cors.max_age", 86400)

	// Rate limiting and client IP
	viper.SetDefault("rate_limit.enabled", true)
	viper.SetDefault("rate_limit.auth_per_minute", 20)
	viper.SetDefault("rate_limit.login_failures", 5)
	viper.SetDefault("rate_limit.login_window", "15m")
	viper.SetDefault("trusted_proxies", []string{})

	// Redis default values
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// LLM default values
	viper.SetDefault("llm.provider", "openai")
	viper.SetDefault("llm.model", "gpt-4o")
	viper.SetDefault("llm.api_key", os.Getenv("OPENAI_API_KEY"))
	viper.SetDefault("llm.base_url", "")

	// Set the config file path
	viper.SetConfigFile(configPath)

	// Read the config file
	// A missing file is fine: configuration can come entirely from
	// GOHEAD_* environment variables (containers, Kubernetes).
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist) {
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

// RedisClientOpt returns the asynq connection options for the configured
// Redis, including password and database index.
func (c Config) RedisClientOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     c.Redis.Address,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	}
}

// RedisOptions returns go-redis options for the configured Redis.
func (c Config) RedisOptions() *redis.Options {
	return &redis.Options{
		Addr:     c.Redis.Address,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
		// Honour context deadlines so a Redis outage cannot stall requests
		// (schema notifications, rate limits) for the full retry budget.
		ContextTimeoutEnabled: true,
		DialTimeout:           2 * time.Second,
		ReadTimeout:           time.Second,
		WriteTimeout:          time.Second,
		MaxRetries:            1,
	}
}
