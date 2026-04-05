-- Ecommerce schema fixture for integration tests.
-- Represents a realistic production schema with common patterns:
-- NOT NULL and nullable columns, multiple tables, diverse column types.

CREATE TABLE users (
    id         serial    NOT NULL,
    email      text      NOT NULL,
    name       text,
    created_at timestamp NOT NULL
);

CREATE TABLE orders (
    id         serial  NOT NULL,
    user_id    integer NOT NULL,
    total      numeric NOT NULL,
    created_at timestamp NOT NULL
);
