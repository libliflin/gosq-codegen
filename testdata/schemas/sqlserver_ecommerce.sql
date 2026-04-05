-- SQL Server ecommerce fixture for integration testing.
-- Two base tables (orders, users) with a mix of NOT NULL and nullable columns,
-- and a view (active_users) that must be excluded from introspect results.
-- This is the SQL Server counterpart to ecommerce.sql.

CREATE TABLE users (
    id         INT IDENTITY(1,1) NOT NULL,
    email      NVARCHAR(255)     NOT NULL,
    name       NVARCHAR(255)     NULL,
    created_at DATETIME2         NOT NULL
);

CREATE TABLE orders (
    id         INT IDENTITY(1,1) NOT NULL,
    user_id    INT               NOT NULL,
    total      DECIMAL(10,2)     NOT NULL,
    created_at DATETIME2         NOT NULL
);

-- View must be excluded by introspect.Tables (table_type = 'BASE TABLE' filter).
CREATE VIEW active_users AS
    SELECT id, email FROM users WHERE name IS NOT NULL;
