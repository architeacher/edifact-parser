-- 004_subscriptions.up.sql
-- Create subscriptions table for market partner subscriptions.

BEGIN;

CREATE TABLE subscriptions (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v7(),
    identifier TEXT        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_subscriptions_identifier UNIQUE (identifier)
);

COMMIT;
