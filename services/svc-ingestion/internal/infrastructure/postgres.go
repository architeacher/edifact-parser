package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
)

const (
	maxRetries      = 3
	initialInterval = 1 * time.Second
)

// NewPostgres creates a pgxpool connection pool with retry logic.
// When OTEL is enabled, an otelpgx tracer is attached to every connection,
// producing spans for queries, prepare, and batch operations.
func NewPostgres(ctx context.Context, dbCfg config.DatabaseConfig, otelCfg config.OTelConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dbCfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parsing postgres config: %w", err)
	}

	poolCfg.MaxConns = int32(dbCfg.PoolSize)

	if otelCfg.Enabled {
		poolCfg.ConnConfig.Tracer = otelpgx.NewTracer(
			otelpgx.WithTrimSQLInSpanName(),
			otelpgx.WithDisableSQLStatementInAttributes(), // Security: no SQL in spans
		)
	}

	var pool *pgxpool.Pool

	operation := func() (*pgxpool.Pool, error) {
		p, connErr := pgxpool.NewWithConfig(ctx, poolCfg)
		if connErr != nil {
			return nil, fmt.Errorf("connecting to postgres: %w", connErr)
		}

		if pingErr := p.Ping(ctx); pingErr != nil {
			p.Close()

			return nil, fmt.Errorf("pinging postgres: %w", pingErr)
		}

		return p, nil
	}

	pool, err = backoff.Retry(ctx, operation,
		backoff.WithBackOff(backoff.NewExponentialBackOff()),
		backoff.WithMaxTries(maxRetries),
	)
	if err != nil {
		return nil, fmt.Errorf("postgres connection failed after retries: %w", err)
	}

	return pool, nil
}
