# Changelog

---

# Changelog — Cycle 27

## Who This Helps
- **Stakeholder:** gosq users with schemas where table and column naming patterns interact
- **Impact:** A cross-table field identifier collision (where two different table+column combinations produce the same Go variable name) now returns a clear error naming both sources: `field "a"."b_c" and "a_b"."c" both produce identifier "ABC"`. Before, the generated file compiled silently — until the user's own Go build failed with `ABC redeclared in this block`, pointing at a `DO NOT EDIT` file with no indication the schema was the cause.

## Observed
- Field variable names are `tableIdent + colIdent` (e.g. table `a_b` + column `c` → `"AB" + "C"` = `"ABC"`).
- The existing collision detection checked table identifiers against each other and column identifiers within a single table — but not full field identifiers across tables.
- A schema with table `a` + column `b_c` and table `a_b` + column `c` passes both existing checks (table idents differ: `A` ≠ `AB`; each table has only one column) yet both produce field variable `ABC`, causing a redeclaration error in the emitted file.

## Applied
- Added a `fieldIdents` map in the pre-render collision check that accumulates every full field identifier (`tableIdent + colIdent`) across all tables. Returns an error if any two table+column pairs produce the same full identifier.
- Added `TestGenerateCrossTableFieldCollision` with the `a`/`b_c` + `a_b`/`c` case.
- **Files:** `internal/codegen/codegen.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project handles all known correctness gaps. Further improvements should be driven by real user feedback — production schemas with unusual patterns, non-public schema names, or scale issues encountered in large databases.

---

# Changelog — Cycle 26

## Who This Helps
- **Stakeholder:** contributors
- **Impact:** A contributor reading `internal/introspect/introspect_test.go` no longer sees a test that implies the package has unit tests. The removed test constructed a `Table` struct inline and asserted `len(Columns)` and `Columns[2].IsNullable` — testing Go's struct literal syntax, not any code in the package. Its presence created a false expectation. The package now honestly reflects its test status: `Tables` requires a live database and has 0% automated coverage, which is correct and expected.

## Observed
- `TestTableStructure` called no function in the `introspect` package. It built a `Table` value directly and asserted things the Go compiler already guarantees.
- The snapshot showed `coverage: 0.0% of statements` for `introspect` — the test had zero effect on this number.
- A contributor seeing a `_test.go` file with a passing test would reasonably assume something was being tested. Nothing was.
- Cycle 25 identified this as the next action: "Consider removing it or replacing it with a test that documents something real about the struct contract."
- There is nothing real to document: `Table` and `Column` are pure data types with no behavior. The only logic lives in `Tables`, which requires a live DB.

## Applied
- Removed `TestTableStructure` and the `"testing"` import from `internal/introspect/introspect_test.go`.
- The file now contains only `package introspect`, honestly declaring the package without pretending it has testable logic.
- **File:** `internal/introspect/introspect_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (introspect: no tests to run)
go vet ./...     — OK
```

## Next
- The project is stable and well-tested for all known edge cases. Further improvements should be driven by real user feedback — production schemas with unusual patterns, non-public schema names, or scale issues encountered in large databases.

---

# Changelog — Cycle 25

## Who This Helps
- **Stakeholder:** gosq users with pathological schema names
- **Impact:** A table named `_` (or `__`, `___` — any name consisting entirely of underscores) previously generated `var _ = NewTable("_")`. The Go blank identifier silently discards the value; the table can never be referenced in user code. The file compiled without error but produced dead code. `Generate` now returns a clear error: `table "_" produces blank identifier "_"; it cannot be referenced in Go`.

## Observed
- `toExported("_")` returns `"_"` (via the empty-string guard added in cycle 6).
- `var _ = NewTable("_")` is valid Go — it compiles and `go/format` accepts it — but the blank identifier discards the return value, making the declaration permanently unreachable.
- The existing collision detection (cycle 23) establishes the pattern: when generated code would be functionally unusable, `Generate` should error with a message naming the problematic table.
- Blank identifier tables are not caught by collision detection because `_` is not a normal identifier that can collide — two tables both producing `_` would silently generate two `var _ = ...` declarations (each valid individually) rather than a redeclaration error.

## Applied
- Added a blank identifier check in `Generate`: immediately after computing `ident := toExported(tbl.Name)`, returns an error if `ident == "_"`.
- Added `TestGenerateBlankIdentifierTable` covering `"_"`, `"__"`, and `"___"` — all names that reduce to the blank identifier.
- **Files:** `internal/codegen/codegen.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- `TestTableStructure` in `introspect_test.go` constructs a `Table` inline and asserts `len(Columns)` and `Columns[2].IsNullable`. It exercises zero code paths (no functions in `introspect` are called). A contributor reading it would assume the package has unit tests. Consider removing it or replacing it with a test that documents something real about the struct contract.
- The project handles all known ASCII, UTF-8, and identifier-edge cases. Further improvements should be driven by real user feedback.

---

# Changelog — Cycle 24

## Who This Helps
- **Stakeholder:** gosq users with non-ASCII column names in their Postgres schema
- **Impact:** A column name starting with a multi-byte UTF-8 character (e.g. `éclat`, `ñoño`, `über`) no longer causes `Generate` to return a cryptic `formatting generated source: ...` error. Before, `toExported` byte-sliced the first character of each word (`part[:1]`), passing an incomplete rune to `strings.ToUpper`. `ToUpper` replaced the invalid byte with U+FFFD (3 bytes), then the remaining bytes of the original rune were prepended to the rest of the word — producing invalid UTF-8. `go/format` failed attempting to parse the malformed source, with no indication the schema column was the cause. After the fix, the first rune is correctly uppercased: `éclat` → `Éclat`.

## Observed
- `toExported` capitalized the first letter of each word part with `strings.ToUpper(part[:1]) + part[1:]` — byte-slicing, not rune-slicing.
- For `"éclat"`: `part[:1]` = `"\xc3"` (first byte of the 2-byte UTF-8 sequence for `é`). `strings.ToUpper("\xc3")` = `"<U+FFFD>"` (3 bytes). Concatenated with `"\xa9clat"` (rest of the string), the result was `"\xef\xbf\xbd\xa9clat"` — invalid UTF-8.
- `go/format` rejects invalid UTF-8 source; error message pointed at source positions, not the column name.
- Verified with a standalone Go program: `ToUpper(part[:1])` produces `"<U+FFFD>"` and the recombined string fails `isValidUTF8`.

## Applied
- Changed `part[:1]` / `part[1:]` byte-slicing to `[]rune` conversion in `toExported`:
  `strings.ToUpper(string([]rune(part)[:1])) + string([]rune(part)[1:])`
- Added `{"éclat", "Éclat"}` to `TestToExported` to document and guard this behavior.
- **Files:** `internal/codegen/codegen.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass, new subtest éclat)
go vet ./...     — OK
```

## Next
- **Blank identifier from `_` table name.** A table named `_` produces `toExported("_") = "_"`, generating `var _ = NewTable("_")`. The blank identifier discards the value silently — it can never be referenced. This should be investigated: should it error like a collision, or is a `_`-prefixed rename appropriate?
- The project handles all known ASCII and UTF-8 edge cases. Further improvements should be driven by real user feedback.

---

# Changelog — Cycle 23

## Who This Helps
- **Stakeholder:** gosq users with unusual schemas
- **Impact:** If two table names (e.g., `user_data` and `user__data`) or two column names within a table produce the same Go identifier after conversion, `Generate` now returns a clear error: `tables "user_data" and "user__data" both produce identifier "UserData"`. Before, the generated file contained duplicate variable declarations and failed to compile with a cryptic Go error pointing into the generated code — no indication the schema was the cause.

## Observed
- `toExported` strips consecutive underscores (empty parts are skipped), so `user_data` and `user__data` both produce `UserData`.
- `Generate` had no duplicate-identifier check; it would silently emit `var UserData = NewTable("user_data")` twice, producing a file that fails with `UserData redeclared in this block`.
- The user's only signal was a Go compiler error in a file marked `DO NOT EDIT`, with no pointer to the schema naming conflict.

## Applied
- Added pre-render collision detection in `Generate`: after sorting, checks all table identifiers for duplicates across the file, and all column identifiers for duplicates within each table.
- Returns a descriptive error naming both conflicting originals and the identifier they produce.
- Added `TestGenerateTableCollision` and `TestGenerateColumnCollision` to `codegen_test.go`.
- **Files:** `internal/codegen/codegen.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
coverage: 98.2% (was 97.8%)
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 22

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** A user who passes `-pkg my-schema` (or any other invalid Go identifier like `-pkg 123abc` or `-pkg var`) now gets an immediate, clear error: `gosq-codegen: invalid package name "my-schema"`. Before, the error surfaced deep in the pipeline as a cryptic `formatting generated source: 1:9: expected 'STRING', found '-'` from `go/format`, with no indication that the package name was the problem.

## Observed
- `main.go` used `*pkg` as a Go package name without validating it at the flag-parsing boundary.
- Invalid identifiers (containing dashes, starting with digits, or matching Go keywords like `var`) would reach `codegen.Generate`, produce syntactically invalid source, and fail in `format.Source` with a message that pointed at source positions rather than the flag.
- `go/token.IsIdentifier` and `go/token.IsKeyword` are the standard library tools for this check.

## Applied
- Added validation of `-pkg` using `token.IsIdentifier` and `token.IsKeyword` immediately after flag parsing (before the DSN check).
- Added `"go/token"` to the import list in `main.go`.
- **File:** `main.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 21

## Who This Helps
- **Stakeholder:** gosq users (first-time discoverers and bug reporters)
- **Impact:** The README flags table now lists all CLI flags. A user reading the docs can discover `-version` exists; before, it was only visible via `-help`.

## Observed
- Cycle 20 added `-version` to `main.go` but did not update the README flags table.
- A user reading the README had no indication the flag existed.

## Applied
- Added `-version` row to the flags table in `README.md`.
- **File:** `README.md`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 20

## Who This Helps
- **Stakeholder:** gosq users (bug reporters) and maintainers
- **Impact:** A user who runs into unexpected behavior can now run `gosq-codegen -version` to report the exact version they're using. Before, the flag didn't exist — running it produced a usage error. This is standard practice for Go CLI tools; its absence was an obvious gap.

## Observed
- `gosq-codegen -version` produced: `flag provided but not defined: -version`.
- Any user filing a bug report had no way to state what version they were running without digging into `go version -m $(which gosq-codegen)`.
- `debug.ReadBuildInfo()` is available in Go 1.18+ and is the standard way to surface module version in a Go binary built with `go install`.

## Applied
- Added `-version` flag to `main.go`.
- If set, prints `bi.Main.Version` from `debug.ReadBuildInfo()` (e.g. `v0.1.2` for installed releases, `(devel)` for local builds) and exits 0.
- **File:** `main.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 19

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** After a successful run, users now see how many tables were generated (e.g., `wrote schema/schema.go (47 tables)`). Before, the output was just `wrote schema/schema.go` with no indication of scale. This lets users immediately verify the tool captured the expected number of tables — catching misconfigurations like a wrong `-schema` flag or unexpected view filtering without opening the file.

## Observed
- `main.go` printed `wrote schema/schema.go` on success with no count.
- A user with a large schema (30+ tables) had no way to verify output completeness at a glance.
- The zero-table warning (cycle 9) handles the empty case, but the non-empty success path gave no quantitative feedback.

## Applied
- Changed the success message from `wrote <file>` to `wrote <file> (N table/tables)`.
- Singular/plural handled correctly: "1 table", "N tables".
- **File:** `main.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 18

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** Users who encounter an error (wrong DSN, network timeout, permission denied) now see a clean, non-redundant error message. Before: `gosq-codegen: introspect: introspect: query information_schema: ...`. After: `gosq-codegen: introspect: query information_schema: ...`.

## Observed
- `introspect.go` prefixed all its errors with `"introspect: "`.
- `main.go` wraps the returned error with `"gosq-codegen: introspect: %v"`.
- The result was a doubled `introspect:` prefix in every error from that package.
- Standard Go convention: library packages do not include their own name in error strings — callers add context via wrapping.

## Applied
- Removed `"introspect: "` prefix from three error strings in `introspect.Tables`:
  `"introspect: query information_schema: %w"` → `"query information_schema: %w"`
  `"introspect: scan row: %w"` → `"scan row: %w"`
  `"introspect: iterate rows: %w"` → `"iterate rows: %w"`
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

# Changelog — Cycle 17

## Who This Helps
- **Stakeholder:** contributors
- **Impact:** The `toExported` name-conversion function is now directly tested across 31 cases. Previously it was covered only indirectly through `Generate`. A contributor adding a new initialism, changing the underscore-stripping logic, or handling a new edge case now has an explicit test suite to run against — and knows exactly what the expected behavior is for each input pattern.

## Observed
- `toExported` is the function that every generated identifier flows through. Every user-visible variable name (`UsersID`, `HTTPStatusCode`, `OauthToken`) is produced by it.
- The function was covered only via `Generate` end-to-end tests, which test a small subset of inputs. A bug in initialism handling (e.g., `"url"` → `"Url"` instead of `"URL"`) would only be caught if that specific initialism appeared in an existing `Generate` test.
- No test explicitly documented the behavior for leading/trailing underscores, compound patterns like `http_status_code`, or the empty-string guard.

## Applied
- Added `TestToExported` to `codegen_test.go`: 31 table-driven subtests covering all 17 initialisms, common snake_case patterns, and all edge cases (digit-leading, all-underscores, empty string, leading/trailing underscores).
- **File:** `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass, 31 new subtests)
go vet ./...     — OK
coverage: 97.8% (unchanged)
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 16

## Who This Helps
- **Stakeholder:** contributors
- **Impact:** A contributor reading `go.mod` no longer sees `// indirect` on `lib/pq`, which is directly imported in `main.go`. The misleading marker could lead someone to believe the blank import is incidental and safe to remove — it is not.

## Observed
- `go.mod` had `require github.com/lib/pq v1.12.3 // indirect`.
- `main.go` imports `_ "github.com/lib/pq"` directly (blank import for driver registration).
- A direct import should not carry the `// indirect` marker. The marker is added by `go mod` only for transitive dependencies not imported by any package in the module.

## Applied
- Ran `go mod tidy`, which removed `// indirect` from the `lib/pq` require line.
- **File:** `go.mod`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 15

## Who This Helps
- **Stakeholder:** contributors / gosq users reading source
- **Impact:** A developer running `go doc github.com/libliflin/gosq-codegen` (or reading `main.go`) now sees output that exactly matches what the tool produces: the `// Code generated` header is present, there's a blank line between the `NewTable` var and the `var (...)` block, and indentation uses tabs (gofmt standard) rather than spaces.

## Observed
- The `main.go` package doc comment was written before cycle 13 added the generated-file header. Cycle 14 fixed the README but the source comment was missed.
- The `var (...)` block in the doc example used 4 spaces for indentation; actual gofmt output uses tabs.
- A developer reading the package docs via `go doc` would see output that differed from actual tool output in three ways: missing header, missing blank line before `var (...)`, and wrong indentation.

## Applied
- Added `// Code generated by gosq-codegen; DO NOT EDIT.` and a blank line to the doc comment example.
- Added blank line between `var Users = NewTable(...)` and `var (...)` in the example.
- Changed indentation inside `var (...)` from 4 spaces to a tab.
- **File:** `main.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 14

## Who This Helps
- **Stakeholder:** gosq users (first-time discoverers)
- **Impact:** A user reading the README now sees the exact output they'll get from the tool. Before this fix, the example omitted the `// Code generated by gosq-codegen; DO NOT EDIT.` header added in cycle 13 — creating a small but real discrepancy between what the docs show and what the tool produces.

## Observed
- The README "Example output" block was written before the generated-file header existed. After cycle 13 added the header to `Generate`, the README was left showing stale output.
- A user comparing the README example to their actual `schema.go` would see an extra line and wonder whether their output is correct.

## Applied
- Added `// Code generated by gosq-codegen; DO NOT EDIT.` as the first line of the README example output block.
- **File:** `README.md`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 13

## Who This Helps
- **Stakeholder:** gosq users
- **Impact:** Generated files now carry the standard Go `// Code generated by gosq-codegen; DO NOT EDIT.` header. Linters, editors, and code reviewers recognize this as a machine-generated file — many tools suppress warnings for such files automatically. Users are clearly signaled not to hand-edit a file that will be overwritten on the next run.

## Observed
- Generated output had no file-level header. A user who edited `schema/schema.go` by hand would have their changes silently overwritten on the next `gosq-codegen` run with no warning.
- The Go convention for generated files is defined in `cmd/go` docs: the first non-blank line of the file should be `// Code generated by <tool>; DO NOT EDIT.`

## Applied
- Prepended `// Code generated by gosq-codegen; DO NOT EDIT.\n\n` to the output in `codegen.Generate`.
- Updated all seven `want` strings in `codegen_test.go` to include the header.
- **Files:** `internal/codegen/codegen.go`, `internal/codegen/codegen_test.go`

## Validated
```
go build ./...   — OK
go test ./...    — OK (all pass)
go vet ./...     — OK
```

## Next
- The project is stable and correct for the core use case. Future improvements should be driven by real user feedback: non-`public` schema support, multiple output files, or schema edge cases encountered in production databases.

---

# Changelog — Cycle 12

## Who This Helps
- **Stakeholder:** contributors
- **Impact:** The `internal/introspect` package no longer silently depends on the `lib/pq` driver. A contributor wanting to swap drivers (e.g., to `pgx`) now changes only `main.go`. The introspect test binary no longer pulls in the postgres driver as a side effect.

## Observed
- `introspect.go` had `_ "github.com/lib/pq"` as a blank import, alongside the identical import in `main.go`.
- The driver only needs to be registered once per binary (via `init()`). The `main.go` import is the right place — it's the entry point that controls which driver is in use.
- The duplicate blank import in an internal package obscures the package's actual dependencies and ties the introspect package to a specific driver unnecessarily.

## Applied
- Removed `_ "github.com/lib/pq"` blank import from `internal/introspect/introspect.go`.
- Driver registration is now solely `main.go`'s responsibility.
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
