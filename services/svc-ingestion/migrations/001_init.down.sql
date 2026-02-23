-- 001_init.down.sql
-- Remove extensions added in 001_init.up.sql.

BEGIN;

DROP EXTENSION IF EXISTS "pg_uuidv7";
DROP EXTENSION IF EXISTS "uuid-ossp";

COMMIT;
