-- MySQL SaaS schema fixture for integration tests.
-- Exercises the MySQL dialect against naming patterns that match the Postgres
-- TestPipelineComplexSchema fixture:
--   - compound snake_case with initialisms (http, url, tls, api, ip, uuid, id, uri)
--   - digit-prefixed column (`2fa_enabled` -> _2faEnabled)
--   - nullable and NOT NULL columns
--   - a VIEW (active_users) that introspect.Tables must exclude
-- Verifies that Tables returns exactly 10 base tables, not 11.

CREATE TABLE users (
    id            INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    email         VARCHAR(255) NOT NULL,
    display_name  VARCHAR(255),
    api_key       VARCHAR(64),
    `2fa_enabled` TINYINT(1)   NOT NULL DEFAULT 0,
    ip_address    VARCHAR(45),
    created_at    DATETIME     NOT NULL,
    updated_at    DATETIME
);

CREATE TABLE organizations (
    id         INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    plan_id    INT          NOT NULL,
    created_at DATETIME     NOT NULL
);

CREATE TABLE api_keys (
    id         INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    account_id INT          NOT NULL,
    key_hash   VARCHAR(64)  NOT NULL,
    expires_at DATETIME,
    created_at DATETIME     NOT NULL
);

CREATE TABLE http_requests (
    id          INT           NOT NULL AUTO_INCREMENT PRIMARY KEY,
    url_path    VARCHAR(2048) NOT NULL,
    method      VARCHAR(10)   NOT NULL,
    status_code INT           NOT NULL,
    user_id     INT,
    created_at  DATETIME      NOT NULL
);

CREATE TABLE tls_certificates (
    id         INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    domain     VARCHAR(255) NOT NULL,
    expires_at DATETIME     NOT NULL,
    created_at DATETIME     NOT NULL
);

CREATE TABLE audit_logs (
    id         INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id    INT          NOT NULL,
    action     VARCHAR(255) NOT NULL,
    ip_addr    VARCHAR(45),
    created_at DATETIME     NOT NULL
);

CREATE TABLE oauth_clients (
    id           INT           NOT NULL AUTO_INCREMENT PRIMARY KEY,
    client_id    VARCHAR(64)   NOT NULL,
    redirect_uri VARCHAR(2048) NOT NULL,
    created_at   DATETIME      NOT NULL
);

CREATE TABLE devices (
    id          INT         NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     INT         NOT NULL,
    device_uuid VARCHAR(36) NOT NULL,
    created_at  DATETIME    NOT NULL
);

CREATE TABLE subscriptions (
    id        INT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    org_id    INT      NOT NULL,
    plan_id   INT      NOT NULL,
    starts_at DATETIME NOT NULL,
    ends_at   DATETIME
);

CREATE TABLE plans (
    id        INT            NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name      VARCHAR(64)    NOT NULL,
    price_usd DECIMAL(10, 2) NOT NULL
);

-- View must be excluded by introspect.Tables (MySQL: table_type = 'BASE TABLE').
CREATE VIEW active_users AS
    SELECT id, email FROM users WHERE `2fa_enabled` = 1
