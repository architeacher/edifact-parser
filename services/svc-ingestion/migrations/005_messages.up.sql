-- 005_messages.up.sql
-- Create messages table for parsed EDIFACT messages.

BEGIN;

CREATE TABLE messages (
    id                    UUID        PRIMARY KEY DEFAULT uuid_generate_v7(),
    interchange_id        UUID        NOT NULL
                                      REFERENCES interchanges (id) ON DELETE RESTRICT,
    processing_version_id UUID        NOT NULL
                                      REFERENCES processing_versions (id) ON DELETE RESTRICT,
    subscription_id       UUID
                                      REFERENCES subscriptions (id) ON DELETE RESTRICT,
    message_type          TEXT        NOT NULL,
    reference             TEXT        NOT NULL,
    segments              JSONB       NOT NULL DEFAULT '[]'::JSONB,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_segments ON messages USING GIN (segments);

COMMIT;
