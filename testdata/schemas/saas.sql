-- SaaS schema fixture for integration tests.
-- Mirrors the 17-table schema from TestGenerateProductionScale so that the
-- integration path exercises the same naming patterns the unit tests simulate:
--   - compound snake_case with initialisms (http, url, tls, api, ip, uuid, id)
--   - digit-prefixed column ("2fa_enabled" → _2faEnabled)
--   - inet, jsonb, uuid, text[] column types
--   - nullable and NOT NULL columns
--   - a VIEW (active_users) that introspect.Tables must exclude
-- Verifies that Tables returns exactly 17 base tables, not 18.

CREATE TABLE accounts (
    id         serial       NOT NULL,
    name       text         NOT NULL,
    owner_id   integer      NOT NULL,
    plan_id    integer      NOT NULL,
    created_at timestamptz  NOT NULL,
    updated_at timestamptz  NOT NULL
);

CREATE TABLE api_keys (
    id         serial      NOT NULL,
    account_id integer     NOT NULL,
    key_hash   text        NOT NULL,
    expires_at timestamptz,
    is_active  boolean     NOT NULL,
    scopes     text[]      NOT NULL
);

CREATE TABLE audit_logs (
    id           bigserial   NOT NULL,
    table_name   text        NOT NULL,
    row_id       integer     NOT NULL,
    action       text        NOT NULL,
    performed_by integer     NOT NULL,
    ip_addr      inet        NOT NULL,
    created_at   timestamptz NOT NULL
);

CREATE TABLE billing_plans (
    id            serial  NOT NULL,
    name          text    NOT NULL,
    price_cents   integer NOT NULL,
    interval_days integer NOT NULL,
    is_active     boolean NOT NULL
);

CREATE TABLE campaigns (
    id         serial      NOT NULL,
    account_id integer     NOT NULL,
    name       text        NOT NULL,
    status     text        NOT NULL,
    launched_at timestamptz,
    ended_at   timestamptz
);

CREATE TABLE devices (
    id          serial      NOT NULL,
    user_id     integer     NOT NULL,
    device_uuid uuid        NOT NULL,
    platform    text        NOT NULL,
    last_seen_at timestamptz,
    push_token  text
);

CREATE TABLE email_templates (
    id         serial      NOT NULL,
    name       text        NOT NULL,
    subject    text        NOT NULL,
    html_body  text        NOT NULL,
    text_body  text        NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE TABLE feature_flags (
    id          serial      NOT NULL,
    name        text        NOT NULL,
    enabled     boolean     NOT NULL,
    rollout_pct integer     NOT NULL,
    updated_at  timestamptz NOT NULL
);

CREATE TABLE http_requests (
    id          bigserial   NOT NULL,
    method      text        NOT NULL,
    url_path    text        NOT NULL,
    status_code integer     NOT NULL,
    duration_ms integer     NOT NULL,
    user_agent  text,
    created_at  timestamptz NOT NULL
);

CREATE TABLE invitations (
    id          serial      NOT NULL,
    email       text        NOT NULL,
    account_id  integer     NOT NULL,
    token       text        NOT NULL,
    sent_at     timestamptz NOT NULL,
    accepted_at timestamptz
);

CREATE TABLE job_queue (
    id           bigserial   NOT NULL,
    job_type     text        NOT NULL,
    payload      jsonb       NOT NULL,
    status       text        NOT NULL,
    attempts     integer     NOT NULL,
    last_error   text,
    scheduled_at timestamptz NOT NULL,
    completed_at timestamptz
);

CREATE TABLE oauth_clients (
    id            serial  NOT NULL,
    name          text    NOT NULL,
    client_id     text    NOT NULL,
    client_secret text    NOT NULL,
    redirect_uri  text    NOT NULL,
    scopes        text[]  NOT NULL
);

CREATE TABLE products (
    id          serial  NOT NULL,
    name        text    NOT NULL,
    sku         text    NOT NULL,
    price_cents integer NOT NULL,
    stock_qty   integer NOT NULL,
    category_id integer NOT NULL
);

CREATE TABLE sessions (
    id         text        NOT NULL,
    user_id    integer     NOT NULL,
    token      text        NOT NULL,
    ip_addr    inet        NOT NULL,
    user_agent text,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE TABLE tls_certificates (
    id          serial      NOT NULL,
    domain      text        NOT NULL,
    issued_at   timestamptz NOT NULL,
    expires_at  timestamptz NOT NULL,
    issuer      text        NOT NULL,
    fingerprint text        NOT NULL
);

CREATE TABLE users (
    id            serial      NOT NULL,
    email         text        NOT NULL,
    full_name     text,
    password_hash text        NOT NULL,
    is_active     boolean     NOT NULL,
    role          text        NOT NULL,
    last_login_at timestamptz,
    "2fa_enabled" boolean     NOT NULL
);

CREATE TABLE webhook_endpoints (
    id          serial      NOT NULL,
    account_id  integer     NOT NULL,
    url         text        NOT NULL,
    secret_hash text        NOT NULL,
    event_types text[]      NOT NULL,
    is_active   boolean     NOT NULL,
    created_at  timestamptz NOT NULL
);

-- This view must NOT appear in introspect.Tables output.
-- The introspect query filters table_type = 'BASE TABLE', excluding views.
CREATE VIEW active_users AS
    SELECT id, email, full_name FROM users WHERE is_active = true;
