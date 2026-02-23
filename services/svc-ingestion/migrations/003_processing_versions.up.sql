-- 003_processing_versions.up.sql
-- Create processing_versions table for reprocessing support.

BEGIN;

CREATE TABLE processing_versions (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v7(),
    interchange_id  UUID        NOT NULL
                                REFERENCES interchanges (id) ON DELETE RESTRICT,
    version_number  INT         NOT NULL,
    parser_version  TEXT        NOT NULL,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_processing_versions_interchange_version
        UNIQUE (interchange_id, version_number),

    CONSTRAINT uq_processing_versions_active_interchange
        EXCLUDE (interchange_id WITH =) WHERE (is_active = TRUE)
);

COMMIT;
