package repos_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDBName     = "ingestion_test"
	testDBUser     = "test"
	testDBPassword = "test"
)

// setupSchema applies the essential DDL for repository tests.
// We skip pg_uuidv7 because the repos always supply explicit IDs.
var schemaStatements = []string{
	`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
	`CREATE TABLE interchanges (
		id                 UUID        PRIMARY KEY,
		sender_id          TEXT        NOT NULL,
		sender_qualifier   TEXT        NOT NULL,
		receiver_id        TEXT        NOT NULL,
		receiver_qualifier TEXT        NOT NULL,
		prepared_at        TIMESTAMPTZ NOT NULL,
		reference          TEXT        NOT NULL,
		raw_content        BYTEA       NOT NULL,
		content_hash       TEXT        NOT NULL,
		status             TEXT        NOT NULL DEFAULT 'completed'
		                   CHECK (status IN ('completed', 'failed')),
		error_detail       TEXT,
		created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT uq_interchanges_content_hash UNIQUE (content_hash)
	)`,
	`CREATE INDEX idx_interchanges_sender_id  ON interchanges (sender_id)`,
	`CREATE INDEX idx_interchanges_created_at ON interchanges (created_at)`,
	`CREATE TABLE processing_versions (
		id              UUID        PRIMARY KEY,
		interchange_id  UUID        NOT NULL REFERENCES interchanges (id) ON DELETE RESTRICT,
		version_number  INT         NOT NULL,
		parser_version  TEXT        NOT NULL,
		is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
		created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT uq_processing_versions_interchange_version
			UNIQUE (interchange_id, version_number),
		CONSTRAINT uq_processing_versions_active_interchange
			EXCLUDE (interchange_id WITH =) WHERE (is_active = TRUE)
	)`,
	`CREATE TABLE subscriptions (
		id         UUID        PRIMARY KEY,
		identifier TEXT        NOT NULL,
		is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT uq_subscriptions_identifier UNIQUE (identifier)
	)`,
	`CREATE TABLE messages (
		id                    UUID        PRIMARY KEY,
		interchange_id        UUID        NOT NULL REFERENCES interchanges (id) ON DELETE RESTRICT,
		processing_version_id UUID        NOT NULL REFERENCES processing_versions (id) ON DELETE RESTRICT,
		subscription_id       UUID        REFERENCES subscriptions (id) ON DELETE RESTRICT,
		message_type          TEXT        NOT NULL,
		reference             TEXT        NOT NULL,
		segments              JSONB       NOT NULL DEFAULT '[]'::JSONB,
		created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`,
	`CREATE INDEX idx_messages_segments ON messages USING GIN (segments)`,
}

// newTestPool creates a PostgreSQL testcontainer and returns a connected pgxpool.Pool.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}

	t.Cleanup(func() {
		if termErr := container.Terminate(context.Background()); termErr != nil {
			t.Logf("terminating postgres container: %v", termErr)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("creating pgxpool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	for index, stmt := range schemaStatements {
		if _, execErr := pool.Exec(ctx, stmt); execErr != nil {
			t.Fatalf("applying schema statement %d: %v\nSQL: %s", index, execErr, stmt)
		}
	}

	return pool
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// insertTestInterchange inserts a minimal interchange row and returns its UUID.
func insertTestInterchange(t *testing.T, pool *pgxpool.Pool, hash string) uuid.UUID {
	t.Helper()

	id := uuid.Must(uuid.NewV7())
	ctx := context.Background()
	sql := `INSERT INTO interchanges (id, sender_id, sender_qualifier, receiver_id, receiver_qualifier, prepared_at, reference, raw_content, content_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	now := time.Now().UTC()

	if _, err := pool.Exec(ctx, sql,
		id, "SENDER01", "14", "RECEIVER01", "14",
		now, fmt.Sprintf("REF-%s", hash[:8]), []byte("UNA:+.? 'UNB+..."), hash,
		"completed", now, now,
	); err != nil {
		t.Fatalf("inserting test interchange: %v", err)
	}

	return id
}

// insertTestProcessingVersion inserts a processing version and returns its UUID.
func insertTestProcessingVersion(t *testing.T, pool *pgxpool.Pool, interchangeID uuid.UUID, versionNumber int, active bool) uuid.UUID {
	t.Helper()

	id := uuid.Must(uuid.NewV7())
	ctx := context.Background()
	sql := `INSERT INTO processing_versions (id, interchange_id, version_number, parser_version, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	if _, err := pool.Exec(ctx, sql,
		id, interchangeID, versionNumber, "v1.0.0", active, time.Now().UTC(),
	); err != nil {
		t.Fatalf("inserting test processing version: %v", err)
	}

	return id
}

// insertTestSubscription inserts a subscription and returns its UUID.
func insertTestSubscription(t *testing.T, pool *pgxpool.Pool, identifier string) uuid.UUID {
	t.Helper()

	id := uuid.Must(uuid.NewV7())
	ctx := context.Background()
	sql := `INSERT INTO subscriptions (id, identifier, created_at) VALUES ($1, $2, $3)`

	if _, err := pool.Exec(ctx, sql, id, identifier, time.Now().UTC()); err != nil {
		t.Fatalf("inserting test subscription: %v", err)
	}

	return id
}
