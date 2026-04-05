# Architecture of gosq-codegen

This file answers the question: how is this project structured, and what are the key decisions the runtime agent needs to know to make safe changes?

---

## Pipeline overview

```
main.go  →  introspect.Tables(ctx, db, schema)  →  codegen.Generate(tables, cfg)  →  os.WriteFile
```

Three packages, one linear data flow. No interfaces, no dependency injection.

---

## `internal/introspect`

**Single exported function:** `Tables(ctx, db, schema) ([]Table, error)`

Runs one SQL query against `information_schema.columns` joined to `information_schema.tables`, filtered by schema and `table_type = 'BASE TABLE'`. Returns tables sorted alphabetically by name, columns ordered by `ordinal_position` (as returned by Postgres — the ORDER BY in the query guarantees this).

**Key types:**
```go
type Table struct {
    Schema  string
    Name    string
    Columns []Column
}
type Column struct {
    Name       string
    DataType   string
    IsNullable bool
    OrdinalPos int
}
```

`DataType` and `OrdinalPos` are collected from `information_schema` but currently unused by `codegen.Generate` — they're present for future use or downstream consumers.

**What the query does NOT return:** views, sequences, materialized views, foreign tables — only base tables. This is an explicit design decision enforced by `AND t.table_type = 'BASE TABLE'`.

**Sort behavior:** The function sorts `tableOrder` alphabetically after collecting all rows. The SQL also has `ORDER BY c.table_name, c.ordinal_position` — so column order within tables comes from Postgres, and table order in the returned slice comes from the Go-level sort.

---

## `internal/codegen`

**Single exported function:** `Generate(tables []Table, cfg Config) ([]byte, error)`

Two phases:
1. **Collision detection** — iterates all tables and columns, builds identifier maps, returns an error if any two names would produce the same Go identifier.
2. **Rendering** — writes Go source into a `bytes.Buffer`, then passes it through `go/format` for canonical formatting.

**`toExported(name string) string`** — the core naming logic. Converts snake_case DB identifiers to exported Go identifiers:
- Splits on `_`
- Applies Go initialism map (`id` → `ID`, `url` → `URL`, etc.) to each part
- Capitalizes first rune (not first byte — handles non-ASCII correctly)
- Handles edge cases: empty string → `"_"`, all-underscore → `"_"`, digit-leading → prefix with `"_"`

The initialism map is the canonical list of Go-idiomatic abbreviations. To add an initialism, add to `goInitialisms` and add a test case to `TestToExported`.

**Collision detection is two-pass conceptually but one-pass in code.** It checks:
- Table–table: two different table names producing the same exported identifier
- Table–field: a table name and a field name (tableIdent + colIdent) producing the same identifier
- Field–field: two different (table, column) pairs from different tables producing the same field identifier
- Column–column within a table: two columns in the same table producing the same per-column identifier

**`go/format.Source`** is applied to the final output. This means the raw buffer doesn't need perfect whitespace — `go/format` normalizes it. It also means if the raw buffer contains invalid Go syntax, `Generate` returns an error. In practice this shouldn't happen (the rendered code is always syntactically valid), but it's a safety net.

---

## `main.go`

CLI wiring only. Validates:
- `-pkg` is a valid, non-keyword Go identifier via `go/token.IsIdentifier` and `token.IsKeyword`
- `-dsn` is non-empty

No validation of the `-schema` flag beyond passing it to `introspect.Tables`. The schema name is used directly in a parameterized SQL query (`WHERE c.table_schema = $1`), so SQL injection is not a concern.

**Timeout:** 30 seconds on the database context. This is a CLI tool — long-running introspection is a user experience problem, not a correctness problem.

**Output:** Prints `wrote schema/schema.go (N tables)` to stdout on success. All errors go to stderr with the `gosq-codegen: ` prefix. Exits 1 on any error.

---

## What to leave alone

- The `information_schema` SQL query — it is minimal and correct. Changing it risks breaking schema isolation or including non-base-table objects.
- The `go/format` call — it must stay. Removing it would produce unformatted output that violates Go conventions.
- The collision detection logic — it's comprehensive and well-tested. Changes here need matching test cases.
- The `toExported` initialism list — additions are fine; removals would break existing generated files for anyone with columns named `id`, `url`, etc.

---

## Extension points

The `Column.DataType` and `Column.IsNullable` fields are populated by `introspect.Tables` but ignored by `codegen.Generate`. A future feature that generates typed fields (e.g., `UsersID int`, `UsersEmail string`) would use these. Right now they're informational only.

The `Config` struct has only `Package` and `DotImport`. Any new generation options (custom naming function, filter for specific tables, type generation) would go in `Config`.
