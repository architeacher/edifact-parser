package config

import (
	"errors"
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

const (
	defaultDBPoolSize      = 20
	defaultKafkaPartition  = 1
	defaultServerPort      = 8080
	defaultLogLevel        = "info"
	defaultOTLPEndpoint    = "localhost:4317"
	defaultKafkaTopic      = "edifact.events"
	defaultServiceName     = "edifact-ingestion"

	minPort = 1
	maxPort = 65535
)

// Config holds all runtime settings loaded from environment variables.
type (
	Config struct {
		Database DatabaseConfig `envconfig:"DB"`
		Kafka    KafkaConfig    `envconfig:"KAFKA"`
		Server   ServerConfig   `envconfig:"SERVER"`
		Logging  LoggingConfig  `envconfig:"LOG"`
		OTel     OTelConfig     `envconfig:"OTEL"`
	}

	// DatabaseConfig holds PostgreSQL connection settings.
	DatabaseConfig struct {
		Host     string `envconfig:"HOST" required:"true"`
		Port     int    `envconfig:"PORT" default:"5432"`
		User     string `envconfig:"USER" required:"true"`
		Password string `envconfig:"PASSWORD" required:"true"`
		Name     string `envconfig:"NAME" required:"true"`
		SSLMode  string `envconfig:"SSLMODE" default:"disable"`
		PoolSize int    `envconfig:"POOL_SIZE" default:"20"`
	}

	// KafkaConfig holds Kafka connection settings.
	KafkaConfig struct {
		Brokers   []string `envconfig:"BROKERS" required:"true"`
		Topic     string   `envconfig:"TOPIC" default:"edifact.events"`
		Partition int      `envconfig:"PARTITION" default:"1"`
	}

	// ServerConfig holds HTTP server settings.
	ServerConfig struct {
		Port int `envconfig:"PORT" default:"8080"`
	}

	// LoggingConfig holds logging settings.
	LoggingConfig struct {
		Level  string `envconfig:"LEVEL" default:"info"`
		Format string `envconfig:"FORMAT" default:"json"`
	}

	// OTelConfig holds OpenTelemetry settings for traces, metrics, and logs.
	OTelConfig struct {
		Endpoint       string `envconfig:"ENDPOINT" default:"localhost:4317"`
		ServiceName    string `envconfig:"SERVICE_NAME" default:"edifact-ingestion"`
		Enabled        bool   `envconfig:"ENABLED" default:"true"`
		MetricsEnabled bool   `envconfig:"METRICS_ENABLED" default:"true"`
		LogsEnabled    bool   `envconfig:"LOGS_ENABLED" default:"true"`
	}
)

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=%d",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode, d.PoolSize,
	)
}

// Load reads configuration from environment variables with the given prefix.
func Load(prefix string) (*Config, error) {
	var cfg Config

	if err := envconfig.Process(prefix, &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// FromEnv loads configuration from environment variables using the default "EDIFACT" prefix.
func FromEnv() (*Config, error) {
	return Load("EDIFACT")
}

// Validate checks that all configuration values are within acceptable ranges.
func (c *Config) Validate() error {
	var errs []error

	if c.Server.Port < minPort || c.Server.Port > maxPort {
		errs = append(errs, fmt.Errorf("server port must be between %d and %d, got %d", minPort, maxPort, c.Server.Port))
	}

	if c.Database.Port < minPort || c.Database.Port > maxPort {
		errs = append(errs, fmt.Errorf("database port must be between %d and %d, got %d", minPort, maxPort, c.Database.Port))
	}

	if c.Database.PoolSize < 1 {
		errs = append(errs, fmt.Errorf("database pool size must be positive, got %d", c.Database.PoolSize))
	}

	if len(c.Kafka.Brokers) == 0 {
		errs = append(errs, fmt.Errorf("at least one Kafka broker is required"))
	}

	return errors.Join(errs...)
}
