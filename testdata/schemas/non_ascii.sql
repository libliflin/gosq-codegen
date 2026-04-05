-- Non-ASCII schema fixture for integration tests.
-- Exercises column names with accented and non-Latin characters to verify
-- that the rune-based identifier conversion handles multi-byte UTF-8 correctly.
--
-- Key cases:
--   éditeur  — starts with 'é' (2-byte UTF-8); toExported must upper-case the
--              first *rune*, not the first *byte*, to produce "Éditeur"
--   prénom   — accent in the middle; first rune is ASCII 'p'

CREATE TABLE articles (
    id      serial NOT NULL,
    éditeur text   NOT NULL,
    prénom  text,
    titre   text   NOT NULL
);
