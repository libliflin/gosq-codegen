-- Reporting schema fixture for multi-schema isolation tests.
-- This table is loaded into a separate schema to verify that introspect.Tables
-- filters by schema name and does not return tables from other schemas.

CREATE TABLE reports (
    id         serial    NOT NULL,
    title      text      NOT NULL,
    created_at timestamp NOT NULL
);
