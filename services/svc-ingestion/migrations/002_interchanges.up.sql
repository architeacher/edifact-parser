-- 002_interchanges.up.sql
-- Create interchanges table for raw EDIFACT file storage.

BEGIN;

CREATE TABLE interchanges (
    id                 UUID        PRIMARY KEY DEFAULT uuid_generate_v7(),
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
);

CREATE INDEX idx_interchanges_sender_id  ON interchanges (sender_id);
CREATE INDEX idx_interchanges_created_at ON interchanges (created_at);

COMMIT;
