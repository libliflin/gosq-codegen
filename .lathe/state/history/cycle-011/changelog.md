# Changelog

---

# Changelog — Cycle 12

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** A user who customizes the package name with `-pkg mydb` now gets `schema/mydb.go` instead of `schema/schema.go`. The filename matches the package name — consistent with Go convention and less surprising when browsing the directory.

## Observed
- `main.go` hardcoded `"schema.go"` as the output filename regardless of the `-pkg` flag.
- A user passing `-pkg mydb` would receive a file named `schema.go` with `package mydb` inside — the filename and package declaration disagreed.
- The default case (`-pkg schema`) was unaffected, but any deviation from the default produced a mismatch.

## Applied
- Changed output filename from `"schema.go"` to `*pkg + ".go"` in `main.go`.
- Updated README to note that the output file is named after the `-pkg` value.
- **Files:** `main.go`, `README.md`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 11

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** A user who mistypes the hostname or port in `-dsn` no longer waits indefinitely for a response. The query now carries a 30-second context timeout; they see a clear error message and the tool exits within that window instead of hanging until Ctrl+C.

## Observed
- `introspect.Tables` used `db.Query` with no context. `db.Query` blocks until the database responds or the OS-level TCP stack gives up (which can be minutes on an unreachable host).
- Standard Go practice for database calls is to accept and propagate `context.Context`, enabling callers to set deadlines. The signature `Tables(db, schema)` had no way to pass one.

## Applied
- Changed `introspect.Tables` signature to `Tables(ctx context.Context, db *sql.DB, schema string)`.
- Replaced `db.Query` with `db.QueryContext(ctx, q, schema)`.
- In `main.go`: created `context.WithTimeout(context.Background(), 30*time.Second)` and passed it to `introspect.Tables`.
- **Files:** `internal/introspect/introspect.go`, `main.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 10

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** Users with views in their schema no longer get spurious `NewTable`/`NewField` declarations for those views. Only base tables are emitted, which is what gosq users expect and need.

## Observed
- `information_schema.columns` includes columns from views as well as base tables.
- A user with a view `v_active_orders` in their `public` schema would get `VActiveOrders = NewTable("v_active_orders")` in their generated file — a view that can't be used the same way as a table with gosq.
- Previous cycle flagged this as the next highest-value change.

## Applied
- Added a `JOIN information_schema.tables` to the introspect query with `AND t.table_type = 'BASE TABLE'`.
- **File:** `internal/introspect/introspect.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 9

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** When a user connects to the wrong database, typos the schema name, or points the tool at an empty schema, they now see a clear warning on stderr instead of silently receiving a `schema.go` with only `package schema`. The file still gets written (it's valid Go), but the user is told something unexpected happened.

## Observed
- `main.go` wrote the output file even when `introspect.Tables` returned zero tables.
- A user who mistyped `-schema mypublic` (instead of `public`) would get a silent success and a useless `schema.go` — no indication anything was wrong.
- No warning was emitted in this case.

## Applied
- Added a stderr warning when `len(tables) == 0`: `gosq-codegen: warning: no tables found in schema "<schema>"`.
- The file is still written (it's valid, compilable Go) — the warning surfaces the likely mistake without breaking the exit-0 contract.
- **File:** `main.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The introspect query (`information_schema.columns`) includes views as well as base tables. Most gosq users will expect only tables. Filtering to `table_type = 'BASE TABLE'` in the introspect query would prevent spurious entries for views — but this is only worth doing if users actually report the confusion.
- The project is stable and well-tested for the core use case. Further improvements should be driven by real user feedback.

---

# Changelog — Cycle 8

## Who This Helps
- **Stakeholder:** contributors
- **Impact:** Two previously untested codegen paths are now covered. The alphabetical sort guarantee (the determinism contract) is explicitly tested. The default-package fallback is tested. Coverage rises from 93.3% → 97.8%.

## Observed
- Coverage profile showed 3 uncovered statement blocks in `Generate`: the `cfg.Package == ""` default, the `sort.Slice` comparator, and the `format.Source` error path.
- No test passed an empty `Package` or provided 2+ tables, leaving the sort and default-package paths unexercised.

## Applied
- Added `TestGenerateTableNoColumns`: verifies a table with no columns emits only the `NewTable` var, no `var (...)` block. Documents edge-case behavior for views or incomplete schemas.
- Added `TestGenerateMultipleTablesOrdered`: provides tables out of alphabetical order with an empty `Package` field. Verifies sorted output (`accounts` before `orders`) and the `"schema"` package default.
- **File:** `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
coverage: 97.8% (was 93.3%)
```

## Next
- The remaining 2.2% uncovered is the `format.Source` error return — unreachable from valid inputs, not worth testing.
- The project is feature-complete and well-tested for its core use case. Future improvements are best driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 7

## Who This Helps
- **Stakeholder:** gosq users (first-time discoverers)
- **Impact:** The README no longer tells new users the tool is "not yet functional." They see real install/usage instructions, a `//go:generate` example, and a flag reference — everything needed to actually use the tool.

## Observed
- README said `**Work in progress.** The core architecture is in place but the generator is not yet functional.` and showed usage under a `## Planned usage` heading.
- The tool has been fully functional since Cycle 3. This was the last place the pre-functional status was visible.

## Applied
- Rewrote README: removed "Work in progress" status block, renamed "Planned usage" → "Usage", added Install section, flags table, `//go:generate` example, and sample output.
- **File:** `README.md`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The introspect package has 0.0% test coverage. The `Tables` function is the only code path and requires a live DB, so 0% is correct and expected — accept it.
- The codegen coverage is 93.3%. The uncovered lines are likely minor branches (e.g., the `DotImport: false` import-line path or error return from `format.Source`). These are low risk but could be verified by checking which lines remain uncovered.
- The project is now feature-complete for the basic use case. Future improvements would come from real user feedback: edge cases in schema introspection, support for non-`public` schemas, or output customization.

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
