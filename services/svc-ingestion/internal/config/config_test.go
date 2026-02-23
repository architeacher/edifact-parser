package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: config.Config{
				Database: config.DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "user",
					Password: "pass",
					Name:     "db",
					SSLMode:  "disable",
					PoolSize: 20,
				},
				Kafka: config.KafkaConfig{
					Brokers:   []string{"localhost:9092"},
					Topic:     "edifact.events",
					Partition: 1,
				},
				Server: config.ServerConfig{Port: 8080},
				Logging: config.LoggingConfig{Level: "info", Format: "json"},
				OTel: config.OTelConfig{
					Endpoint:       "localhost:4317",
					ServiceName:    "edifact-ingestion",
					Enabled:        true,
					MetricsEnabled: true,
					LogsEnabled:    true,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid server port zero",
			cfg: config.Config{
				Database: config.DatabaseConfig{
					Host: "localhost", Port: 5432, User: "u", Password: "p", Name: "d", PoolSize: 1,
				},
				Kafka:  config.KafkaConfig{Brokers: []string{"b:9092"}},
				Server: config.ServerConfig{Port: 0},
			},
			wantErr: true,
		},
		{
			name: "invalid server port too high",
			cfg: config.Config{
				Database: config.DatabaseConfig{
					Host: "localhost", Port: 5432, User: "u", Password: "p", Name: "d", PoolSize: 1,
				},
				Kafka:  config.KafkaConfig{Brokers: []string{"b:9092"}},
				Server: config.ServerConfig{Port: 70000},
			},
			wantErr: true,
		},
		{
			name: "invalid database port",
			cfg: config.Config{
				Database: config.DatabaseConfig{
					Host: "localhost", Port: 0, User: "u", Password: "p", Name: "d", PoolSize: 1,
				},
				Kafka:  config.KafkaConfig{Brokers: []string{"b:9092"}},
				Server: config.ServerConfig{Port: 8080},
			},
			wantErr: true,
		},
		{
			name: "invalid pool size",
			cfg: config.Config{
				Database: config.DatabaseConfig{
					Host: "localhost", Port: 5432, User: "u", Password: "p", Name: "d", PoolSize: 0,
				},
				Kafka:  config.KafkaConfig{Brokers: []string{"b:9092"}},
				Server: config.ServerConfig{Port: 8080},
			},
			wantErr: true,
		},
		{
			name: "no kafka brokers",
			cfg: config.Config{
				Database: config.DatabaseConfig{
					Host: "localhost", Port: 5432, User: "u", Password: "p", Name: "d", PoolSize: 1,
				},
				Kafka:  config.KafkaConfig{Brokers: []string{}},
				Server: config.ServerConfig{Port: 8080},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	t.Parallel()

	cfg := config.DatabaseConfig{
		Host:     "db.example.com",
		Port:     5432,
		User:     "admin",
		Password: "secret",
		Name:     "edifact",
		SSLMode:  "require",
		PoolSize: 10,
	}

	dsn := cfg.DSN()

	require.Contains(t, dsn, "db.example.com")
	require.Contains(t, dsn, "admin")
	require.Contains(t, dsn, "edifact")
	require.Contains(t, dsn, "sslmode=require")
	require.Contains(t, dsn, "pool_max_conns=10")
}
