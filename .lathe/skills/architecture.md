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

This is a clean one-way pipeline. Each stage has a single responsibility and a clear boundary. Don't blur these boundaries.

---

## Package responsibilities

### `internal/introspect`

**What it owns:** Reading schema metadata from a live Postgres connection via `information_schema.columns`. It knows about SQL. It knows nothing about Go code generation.

**Current state:** Fully implemented. `Tables(ctx context.Context, db *sql.DB, schema string) ([]Table, error)` queries `information_schema.columns` joined with `information_schema.tables` (filtered to `table_type = 'BASE TABLE'` — views are excluded). Returns `[]Table` sorted by table name, with columns in `OrdinalPos` order.

**Driver:** `github.com/lib/pq` is registered in `main.go` (blank import). `introspect` itself has no driver dependency — it receives an already-opened `*sql.DB`. Swapping drivers means changing only `main.go`.

**Context:** `Tables` takes a `context.Context` and uses `db.QueryContext`. `main.go` creates a 30-second timeout context. The timeout is hardcoded — not configurable via flags.

**Error strings:** `introspect` does not prefix its own package name into error strings — callers (main.go) add context via wrapping. This is correct Go convention. The errors currently are: `"query information_schema: %w"`, `"scan row: %w"`, `"iterate rows: %w"`.

### `internal/codegen`

**What it owns:** Rendering `[]introspect.Table` into Go source code. It knows about Go syntax and the gosq API. It knows nothing about databases or SQL.

**Current state:** Fully implemented. `Generate(tables []introspect.Table, cfg Config) ([]byte, error)`:
- Sorts tables alphabetically (determinism guarantee)
- Detects all five identifier collision types before rendering (returns a descriptive error naming both conflicting tables/columns)
- Renders `// Code generated` header, package declaration, import, then one `var Table = NewTable(...)` + `var (...)` block per table
- Applies `go/format` to guarantee gofmt-clean output

**Config:**
- `Package string` — Go package name (default: `"schema"`)
- `DotImport bool` — whether to use `import . "github.com/libliflin/gosq"` (default: `true`)

**`toExported(name string) string`** — converts `snake_case` to `PascalCase` with Go initialisms. Not exported. Tested directly via `TestToExported` (43 subtests). The full initialism list: `id`, `url`, `uri`, `http`, `https`, `sql`, `api`, `uid`, `uuid`, `ip`, `io`, `cpu`, `xml`, `json`, `rpc`, `tls`, `ttl`.

**Identifier capitalization:** The function uses `[]rune` slicing (not byte slicing) for capitalization:
`strings.ToUpper(string([]rune(part)[:1])) + string([]rune(part)[1:])`. This correctly handles multi-byte UTF-8 characters at the start of a word.

**Blank identifier guard:** If `toExported(tbl.Name) == "_"`, `Generate` returns an error immediately. This prevents generating `var _ = NewTable(...)` which would compile but silently discard the value via the blank identifier.

### `main.go`

**What it owns:** CLI surface. Parses flags, validates them, opens the DB, calls `introspect.Tables`, calls `codegen.Generate`, creates the output directory, writes the file.

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `-dsn` | *(required)* | PostgreSQL connection string |
| `-out` | `schema/` | Output directory |
| `-pkg` | `schema` | Go package name for generated file |
| `-schema` | `public` | PostgreSQL schema to introspect. Generated identifiers do not include the schema name — use distinct `-pkg` and `-out` values when generating from multiple schemas. |
| `-dot-import` | `true` | Use dot-import for gosq |
| `-version` | | Print version and exit |

**Validation:** `-pkg` is validated with `go/token.IsIdentifier` and `go/token.IsKeyword` before connecting to the database. Invalid package names produce a clear error immediately.

**Output file:** Always `<out>/<pkg>.go`. With defaults: `schema/schema.go`.

**Timeout:** 30 seconds, hardcoded in the context passed to `introspect.Tables`.

**Success message:** `wrote schema/schema.go (N tables)` — includes count for verification.
**Warning:** `gosq-codegen: warning: no tables found in schema "<schema>"` if zero tables are returned (still writes the file, which is valid Go with just the package declaration).

**`-version` flag:** Uses `debug.ReadBuildInfo()`. With `go install`, version comes from module metadata and prints correctly (e.g., `v0.1.0`). For local builds (`go build .`), it prints `(devel)` — this is expected Go behavior, not a bug.

---

## The output format

Generated files look like this:

```go
// Code generated by gosq-codegen; DO NOT EDIT.

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

Key invariants (tested and enforced):
- `// Code generated by gosq-codegen; DO NOT EDIT.` is always the first line
- Tables sorted alphabetically
- Columns in `OrdinalPos` order within each table
- `NewField` argument is `"tablename.columnname"` (lowercase originals from Postgres)
- Field variable names are `TableIdent + toExported(columnName)`
- `var (...)` block omitted if table has no columns
- Output is gofmt-clean (tabs, not spaces; `=` signs aligned within `var (...)` blocks)
- Output is a deterministic function of input

---

## Multi-schema behavior

The `-schema` flag selects which PostgreSQL schema to introspect. Generated identifiers are based on table and column names only — the schema name is not included in any identifier. Two different schemas sharing table names would produce identical Go identifiers.

**Implication:** Users who need to generate from multiple schemas must use distinct `-pkg` and `-out` values to produce separate Go packages. This is documented in the README and in the flags table. There is no warning when a non-default schema is used.

---

## What NOT to add prematurely

- Multiple output files (one per table) — single-file is correct until a user reports the output file is unmanageably large
- A config file (`.gosq-codegen.yaml`) — flags are sufficient
- Type mapping from Postgres types to Go types — `NewField` takes a string; type information in the generated file would couple output to gosq's internals
- Interfaces or adapters over `introspect` and `codegen` for hypothetical future databases or output formats — the pipeline is the right shape now
- A `-timeout` flag — 30 seconds covers all realistic cases unless a user specifically reports hitting it
- A `-dry-run` flag — the output is deterministic and the file is overwritten atomically, so there's no need
