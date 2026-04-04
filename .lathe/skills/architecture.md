# Architecture of gosq-codegen

This file exists to answer: what are the structural decisions already made in this codebase, and what do they imply for future changes?

---

## The pipeline

```
PostgreSQL DB
     │
     ▼
internal/introspect   — queries information_schema, returns []Table
     │
     ▼
internal/codegen      — renders []Table into []byte of Go source
     │
     ▼
main.go               — CLI: parses flags, calls introspect, calls codegen, writes files
```

This is a clean one-way pipeline. Each stage has a single responsibility and a clear boundary. The existing design is correct — don't blur these boundaries.

---

## Package responsibilities

### `internal/introspect`

**What it owns:** Reading schema metadata from a live Postgres connection. It knows about SQL and `information_schema`. It knows nothing about Go code generation.

**Current state:** Has `Table` and `Column` types but no query function. The missing piece is a function like:

```go
func Tables(db *sql.DB, schema string) ([]Table, error)
```

This function runs a query against `information_schema.columns`, groups results by table name, and returns `[]Table` sorted by table name with columns ordered by `OrdinalPos`.

**Dependencies it will need:** A PostgreSQL driver. The standard choice is `github.com/lib/pq` (pure Go, no cgo) or `github.com/jackc/pgx/v5`. Either works; `lib/pq` is simpler for a tool that doesn't need connection pooling or advanced features.

### `internal/codegen`

**What it owns:** Rendering `[]introspect.Table` into Go source code. It knows about Go syntax and the gosq API. It knows nothing about databases or SQL.

**Current state:** Has `Config` struct and `Generate([]introspect.Table, Config) ([]byte, error)` signature, but `Generate` returns `nil, nil`. This is the highest-priority implementation gap.

**The `Config` struct has two fields:**
- `Package string` — the Go package name for the generated file (default: `"schema"`)
- `DotImport bool` — whether to use `import . "github.com/libliflin/gosq"` (dot-import)

These are the right knobs. Don't add more configuration until a real user need surfaces.

### `main.go`

**What it owns:** CLI surface. Parses `-dsn` and `-out` flags, calls `introspect.Tables`, calls `codegen.Generate`, writes the output file. Error messages go to stderr. Exit code 1 on failure.

**Current state:** Prints `"gosq-codegen: not yet implemented"` and exits 1. The doc comment at the top shows exactly what it should eventually do.

---

## The output format

Based on the package doc in `main.go` and the README, the expected output is a single `.go` file per schema (or all tables in one file). The format:

```go
package schema

import . "github.com/libliflin/gosq"

var Users = NewTable("users")

var (
    UsersID    = NewField("users.id")
    UsersName  = NewField("users.name")
    UsersEmail = NewField("users.email")
)

var Orders = NewTable("orders")

var (
    OrdersID     = NewField("orders.id")
    OrdersUserID = NewField("orders.user_id")
)
```

Key observations:
- One `var TableName = NewTable(...)` per table
- One `var (...)` block of `NewField` calls per table, grouped with the table
- Field variable names are `TableName` + PascalCase(column) — e.g., `UsersID` for `users.id`
- `NewField` argument is `"tablename.columnname"` (lowercase, dot-separated, as stored in Postgres)
- Tables sorted alphabetically; columns in `OrdinalPos` order within each table

This output format is implied by existing examples but not yet enforced by any test. Once `Generate` works, lock this format with a test.

---

## What main.go will need

When `main.go` is fully implemented, it needs:
1. `flag.String("dsn", "", "PostgreSQL connection string")` and `flag.String("out", "schema/", "output directory")`
2. Open a `*sql.DB` with the dsn
3. Call `introspect.Tables(db, "public")` (or allow schema to be configured)
4. Call `codegen.Generate(tables, codegen.Config{Package: pkg, DotImport: true})`
5. Write the result to `<out>/schema.go` (or a configurable filename)
6. Close the DB

The output directory should be created if it doesn't exist (`os.MkdirAll`).

---

## What NOT to add prematurely

- Multi-schema support (support `public` first; add `-schema` flag when a user asks)
- Multiple output files (one file per table) unless the single-file approach proves unwieldy
- A config file (`.gosq-codegen.yaml` or similar) — flags are sufficient at this stage
- Type mapping from Postgres data types to Go types — the generated code doesn't need Go types; `NewField` takes a string and gosq handles the rest
