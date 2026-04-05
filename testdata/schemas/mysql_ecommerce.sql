-- MySQL-compatible ecommerce schema fixture for MySQL integration tests.
-- Mirrors the structure of ecommerce.sql but uses MySQL syntax:
-- INT AUTO_INCREMENT instead of serial, DATETIME instead of timestamp,
-- DECIMAL instead of numeric. Two base tables with NOT NULL and nullable
-- columns — the same patterns exercised against PostgreSQL in TestPipelineEcommerce.

CREATE TABLE users (
    id         INT           NOT NULL AUTO_INCREMENT,
    email      VARCHAR(255)  NOT NULL,
    name       VARCHAR(255)  DEFAULT NULL,
    created_at DATETIME      NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE orders (
    id         INT            NOT NULL AUTO_INCREMENT,
    user_id    INT            NOT NULL,
    total      DECIMAL(10,2)  NOT NULL,
    created_at DATETIME       NOT NULL,
    PRIMARY KEY (id)
);
