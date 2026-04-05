-- Non-ASCII table name fixture for integration tests.
-- Exercises table names with accented characters to verify that the full
-- pipeline handles multi-byte UTF-8 table names correctly.
--
-- Key cases:
--   étagères  — starts with 'é' (2-byte UTF-8); toExported must upper-case the
--               first *rune*, not the first *byte*, to produce "Étagères"
--   données   — starts with ASCII 'd' but contains non-ASCII runes; produces "Données"
--
-- Both tables have simple columns so the test focuses on the table-name path.

CREATE TABLE étagères (
    id   serial NOT NULL,
    nom  text   NOT NULL
);

CREATE TABLE données (
    id       serial NOT NULL,
    contenu  text,
    créé_le  timestamptz NOT NULL
);
