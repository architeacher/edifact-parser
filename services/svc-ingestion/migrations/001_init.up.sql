-- 001_init.up.sql
-- Enable required PostgreSQL extensions.

BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_uuidv7";

COMMIT;
