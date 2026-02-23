package ports

import (
	"context"
)

// Transaction represents an active database transaction.
type (
	Transaction interface {
		// Commit commits the transaction.
		Commit(ctx context.Context) error

		// Rollback aborts the transaction. No-op if already committed.
		Rollback(ctx context.Context) error
	}
)

// TransactionManager creates database transactions.
type (
	TransactionManager interface {
		// Begin starts a new transaction.
		Begin(ctx context.Context) (Transaction, error)
	}
)
