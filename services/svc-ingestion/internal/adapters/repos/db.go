package repos

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

// DBTX abstracts pgxpool.Pool and pgx.Tx for repository operations.
type (
	DBTX interface {
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
		Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
		QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	}
)

// TransactionManager implements ports.TransactionManager for PostgreSQL.
type (
	TransactionManager struct {
		pool *pgxpool.Pool
	}

	// pgxTransaction wraps pgx.Tx to implement ports.Transaction.
	pgxTransaction struct {
		tx pgx.Tx
	}
)

// NewTransactionManager creates a TransactionManager backed by a connection pool.
func NewTransactionManager(pool *pgxpool.Pool) ports.TransactionManager {
	return &TransactionManager{pool: pool}
}

// Begin starts a new transaction.
func (tm *TransactionManager) Begin(ctx context.Context) (ports.Transaction, error) {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}

	return &pgxTransaction{tx: tx}, nil
}

// Commit commits the transaction.
func (t *pgxTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction.
func (t *pgxTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}
