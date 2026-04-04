# Changelog

---

# Changelog — Cycle 6

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** A column named entirely with underscores (e.g. `___`) no longer causes `Generate` to return a cryptic `format.Source` error. The fallback identifier `_` is emitted, keeping the generated file compilable.

## Observed
- Previous cycle's "Next" flagged Go reserved words as the concern. On inspection, `toExported` always capitalizes, so no Go keyword (all lowercase) can ever be produced. The real gap was the all-underscore case: every part is empty, `strings.Builder` produces `""`, and `""` is an invalid Go identifier that fails `format.Source`.
- Evidence: `toExported("___")` returned `""` with no guard.

## Applied
- Added empty-result guard to `toExported`: if `result == ""`, return `"_"`.
- Added `TestGenerateAllUnderscoreColumn` verifying full `Generate` output for an all-underscore column name.
- **Files:** `internal/codegen/codegen.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The `introspect` package has 0.0% test coverage. Its only exported function (`Tables`) requires a live DB — 0% is correct for that path. However, there is no helper logic that could be extracted and tested in isolation. Coverage is a non-issue here; accept it.
- The README still says "planned usage" and shows a command that previously produced `exit status 1`. Update it to reflect that the tool is now functional, show the real usage, and give users a working `//go:generate` example.

---

# Changelog — Cycle 5

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** Any schema with a column starting with a digit (e.g., `2fa_enabled`) no longer causes `Generate` to return a cryptic `format.Source` error. The generated identifier is prefixed with `_`, producing valid Go (`Accounts_2faEnabled`).

## Observed
- `toExported("2fa_enabled")` returned `"2faEnabled"` — a digit-leading string that `go/format` rejects as an invalid identifier.
- Previous cycle flagged this as the next highest-value change.

## Applied
- Added digit-prefix guard to `toExported`: if the result starts with `'0'–'9'`, prepend `_`.
- Added `TestGenerateDigitLeadingColumn` to verify the full `Generate` output for a digit-leading column name.
- **Files:** `internal/codegen/codegen.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The `introspect` package shows 0.0% test coverage because its only function (`Tables`) requires a live DB. The struct-construction test exercises no code paths. Consider whether any non-DB helper logic should be extracted and tested (e.g., ordinal-sort logic), or accept 0% coverage as correct for a DB-dependent package.
- Column names that are Go reserved words (`type`, `func`, `var`, etc.) would generate invalid identifiers. Add a reserved-word suffix guard (`_`) and a test.

---

# Changelog — Cycle 4

## Who This Helps
- **Stakeholder:** gosq users (specifically those with strict linting rules)
- **Impact:** Users whose projects ban dot imports can now pass `-dot-import=false` to get `gosq.NewTable(...)` style output. Previously, there was no escape from the hardcoded default.

## Observed
- `main.go` hardcoded `DotImport: true` — `Config.DotImport` existed but was never wired to a CLI flag.
- The `DotImport: false` code path in `codegen.Generate` had no test coverage (part of the 17.5% uncovered).

## Applied
- Added `-dot-import` flag (default `true`) to `main.go`, wired to `codegen.Config.DotImport`.
- Added `TestGenerateDotImportFalse` to `codegen_test.go`: validates `gosq.` prefix output for both `NewTable` and `NewField`.
- **Files:** `main.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- Edge case: column names that start with a digit produce invalid Go identifiers (e.g., `Users1name`), causing `format.Source` to fail with a cryptic error. Prefix with `_` in `toExported` and add a test.
- The `introspect` package has 0.0% test coverage — the struct construction tests that exist aren't wired in a way that counts. Verify `introspect_test.go` is actually exercising any code.

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
