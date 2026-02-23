-- 006_domain_events.down.sql
-- Drop domain_events table and its triggers/functions.

BEGIN;

DROP TRIGGER IF EXISTS trg_domain_events_no_delete ON domain_events;
DROP TRIGGER IF EXISTS trg_domain_events_no_update ON domain_events;
DROP FUNCTION IF EXISTS prevent_domain_events_mutation();
DROP TABLE IF EXISTS domain_events;

COMMIT;
