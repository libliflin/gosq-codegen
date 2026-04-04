# Changelog

---

# Changelog — Cycle 3

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** The tool is no longer a stub. Running `gosq-codegen -dsn "postgres://..." -out schema/` now produces a working `schema/schema.go` file. The first-try experience goes from `exit status 1` to a file on disk.

## Observed
- `main.go` printed `"gosq-codegen: not yet implemented"` and exited 1.
- `introspect.Tables` and `codegen.Generate` were both fully implemented — only the CLI wiring was missing.

## Applied
- Replaced stub `main()` with full CLI: parses `-dsn`, `-out`, `-pkg`, `-schema` flags; calls `introspect.Tables`; calls `codegen.Generate`; creates output directory; writes `<out>/schema.go`; prints the output path on success.
- **File:** `main.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The tool is functionally complete for the happy path. The next highest-value change is verifying real edge cases: tables with no columns, column names that are Go reserved words, or column names starting with a digit. Add tests or try against a real schema.
- Consider adding a `-dot-import=false` flag path through the CLI (currently hardcoded to `DotImport: true`).

---

# Changelog — Cycle 2

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** The pipeline now has a real DB query layer. `main.go` can be wired end-to-end next cycle — one cycle away from a working first-try experience.

## Observed
- `introspect.go` had `Table`/`Column` types but no function to populate them from a database.
- `codegen.Generate` was working but had no input source — the pipeline was severed at the bottom.

## Applied
- Implemented `introspect.Tables(db *sql.DB, schema string) ([]Table, error)`: queries `information_schema.columns`, groups by table, returns sorted `[]Table` with columns in ordinal position order.
- Added `github.com/lib/pq` driver (blank import) via `go get`.
- **Files:** `internal/introspect/introspect.go`, `go.mod`, `go.sum`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- Wire `main.go` end-to-end: parse `-dsn` and `-out` flags, call `introspect.Tables`, call `codegen.Generate`, write `<out>/schema.go`.
- This is the last missing piece before the first-try user experience works.

---

# Changelog — Cycle 1

## Who This Helps
- **Stakeholder:** gosq users and contributors
- **Impact:** `codegen.Generate` now produces real Go source. The engine exists. Every subsequent cycle builds on working output rather than `nil, nil`.

## Observed
- `codegen.Generate` returned `nil, nil` unconditionally — the entire generation path was a stub.
- Tests in `codegen_test.go` discarded output with `_ = out` and asserted nothing.
- Coverage showed 100% on `codegen` only because the stub had no branches — meaningless.

## Applied
- Implemented `codegen.Generate`: sorts tables alphabetically, renders `NewTable`/`NewField` declarations, pipes output through `go/format` for gofmt-clean output.
- Implemented `toExported`: converts `snake_case` → `PascalCase` with Go initialisms (`id` → `ID`, `url` → `URL`, etc.).
- Handles both `DotImport: true` (`.` import) and `DotImport: false` (`gosq.` prefix).
- Replaced `// TODO` placeholders in `codegen_test.go` with real string assertions.
- **Files:** `internal/codegen/codegen.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- Implement `introspect.Tables(db *sql.DB, schema string) ([]Table, error)` — the live DB query. Requires adding `lib/pq` or `pgx` to `go.mod`.
- Wire `main.go` end-to-end: parse flags → introspect → generate → write file.
- Once `main.go` is wired, the first-try user experience goes from "exit status 1" to a working `.go` file.
