-- Fixture for view-exclusion test.
-- Contains one base table and one view derived from it.
-- introspect.Tables must return only the base table.
CREATE TABLE products (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    price      NUMERIC(10,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE VIEW active_products AS
    SELECT id, name, price FROM products WHERE price > 0;
