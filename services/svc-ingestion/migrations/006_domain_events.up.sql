-- 006_domain_events.up.sql
-- Create domain_events audit table (append-only).

BEGIN;

CREATE TABLE domain_events (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v7(),
    interchange_id  UUID
                                REFERENCES interchanges (id) ON DELETE RESTRICT,
    event_type      TEXT        NOT NULL,
    payload         JSONB       NOT NULL DEFAULT '{}'::JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_domain_events_interchange_id ON domain_events (interchange_id);
CREATE INDEX idx_domain_events_event_type     ON domain_events (event_type);

-- Prevent UPDATE and DELETE on domain_events (immutable audit log).
CREATE OR REPLACE FUNCTION prevent_domain_events_mutation()
    RETURNS TRIGGER
    LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'domain_events table is immutable: % operations are not allowed', TG_OP;
    RETURN NULL;
END;
$$;

CREATE TRIGGER trg_domain_events_no_update
    BEFORE UPDATE ON domain_events
    FOR EACH ROW
    EXECUTE FUNCTION prevent_domain_events_mutation();

CREATE TRIGGER trg_domain_events_no_delete
    BEFORE DELETE ON domain_events
    FOR EACH ROW
    EXECUTE FUNCTION prevent_domain_events_mutation();

COMMIT;
