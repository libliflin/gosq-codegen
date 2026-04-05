-- Production-scale schema fixture for integration pipeline tests.
-- Covers naming patterns that must survive the full introspect→codegen→build
-- pipeline: initialisms (api, http, url, ip, uuid, id), digit-prefixed columns
-- (2fa_enabled), consecutive initialisations (user_uuid, http_status_code),
-- and common snake_case patterns (created_at, updated_at, is_active).

CREATE TABLE users (
    id            serial       NOT NULL,
    email         text         NOT NULL,
    full_name     text,
    is_active     boolean      NOT NULL,
    role          text         NOT NULL,
    "2fa_enabled" boolean      NOT NULL,
    created_at    timestamptz  NOT NULL,
    updated_at    timestamptz  NOT NULL
);

CREATE TABLE api_keys (
    id           serial       NOT NULL,
    user_id      integer      NOT NULL,
    api_key      text         NOT NULL,
    expires_at   timestamptz,
    is_active    boolean      NOT NULL,
    created_at   timestamptz  NOT NULL
);

CREATE TABLE http_logs (
    id                serial       NOT NULL,
    url               text         NOT NULL,
    http_status_code  integer      NOT NULL,
    ip_addr           inet         NOT NULL,
    method            text         NOT NULL,
    logged_at         timestamptz  NOT NULL
);

CREATE TABLE sessions (
    id           serial       NOT NULL,
    user_id      integer      NOT NULL,
    user_uuid    uuid         NOT NULL,
    expires_at   timestamptz  NOT NULL,
    created_at   timestamptz  NOT NULL
);

CREATE TABLE audit_logs (
    id           bigserial    NOT NULL,
    table_name   text         NOT NULL,
    row_id       integer      NOT NULL,
    action       text         NOT NULL,
    performed_by integer,
    ip_addr      inet,
    created_at   timestamptz  NOT NULL
);
