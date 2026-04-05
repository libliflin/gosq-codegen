# Testing in gosq-codegen

This file exists to answer: how does *this* project test, and what should new tests look like?

---

## The fundamental split

Two packages. Two completely different testing strategies.

### `internal/introspect`

The real job of this package is to query `information_schema.columns` in a live Postgres database. That can't be unit-tested without a real database.

**Unit tests:** `introspect_test.go` currently contains only `package introspect`. There is nothing to unit-test — `Table` and `Column` are pure data types, and `Tables` requires a live DB. Don't mock `database/sql`.

**Integration tests (the next major addition):** Add `//go:build integration` tests that connect to a real Postgres instance via `TEST_DSN`. These tests should:
1. Load DDL fixtures from `testdata/schemas/*.sql` into the database
2. Call `introspect.Tables` and verify the returned `[]Table` matches the expected structure
3. Pipe the result through `codegen.Generate` and verify the output compiles

DDL fixtures should represent real-world patterns:
- `testdata/schemas/ecommerce.sql` — 10-15 tables (users, orders, products, etc.), diverse column types, foreign keys, nullable columns
- `testdata/schemas/non_ascii.sql` — tables/columns with accented characters, non-Latin scripts
- `testdata/schemas/extensions.sql` — PostGIS geometry columns, pgcrypto, uuid-ossp
- `testdata/schemas/multi_schema.sql` — tables in `public` + a custom schema like `reporting`

These tests run in CI via `services: postgres:` in the GitHub Actions workflow. They are skipped locally unless `TEST_DSN` is set. Document how to run them locally with Docker:
```bash
docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=test postgres:16
TEST_DSN="postgres://postgres:test@localhost:5432/postgres?sslmode=disable" go test -tags integration ./...
```

### `internal/codegen`

This package turns `[]introspect.Table` into a `[]byte` of Go source. It requires no database. This is where most test effort belongs.

**What to test:**
- The exact shape and content of generated Go source for a known input
- That output is deterministic (same input → same output)
- Edge cases: empty table list, table with no columns, digit-leading column names (`2fa_enabled`), all-underscore column names (`___`), blank column names (`_`), non-ASCII column names (`éclat`), identifier collisions (all five types), multiple tables (sort order), `DotImport: false` path
- `toExported` directly — via the table-driven `TestToExported` (43 subtests)

**Current state (after ~37 cycles):** `codegen_test.go` has 16 test functions covering all of the above. Coverage is 98.6%. The only uncovered code is the `format.Source` error return path — unreachable from any valid input, not worth testing.

---

## Current test inventory

| Test function | What it covers |
|---|---|
| `TestGenerateEmpty` | nil table list → header + package only |
| `TestGenerateDotImportFalse` | `DotImport: false` path, `gosq.` prefix |
| `TestGenerateDigitLeadingColumn` | `2fa_enabled` → `Accounts_2faEnabled` |
| `TestGenerateBlankColumn` | column `_` → `Items_` (valid exported ident) |
| `TestGenerateAllUnderscoreColumn` | column `___` → `Items_` (all underscores collapse) |
| `TestGenerateTableNoColumns` | table with `nil` Columns → no `var (...)` block |
| `TestGenerateMultipleTablesOrdered` | out-of-order input → alphabetical output; empty Package → default |
| `TestGenerateTableCollision` | `user_data` + `user__data` → both `UserData` → error |
| `TestGenerateBlankIdentifierTable` | `_`, `__`, `___` table names → error |
| `TestGenerateTableFieldCollision` | `users_id` table + `users.id` field → both `UsersID` → error |
| `TestGenerateCrossTableFieldCollision` | `a.b_c` + `a_b.c` → both `ABC` → error |
| `TestGenerateColumnCollision` | `my_field` + `my__field` in same table → error |
| `TestGenerateMultiUnderscoreColumnCollision` | `_` + `__` in same table → both `Items_` → error |
| `TestGenerateFieldPriorTableCollision` | `_users_name` (sorts early) + `users.name` → both `UsersName` → error |
| `TestGenerateProductionScale` | 17 tables, 108 columns, diverse patterns, determinism |
| `TestGenerateSingleTable` | basic single-table output with two columns, exact string match |
| `TestToExported` | 43 subtests: all 17 initialisms, edge cases, consecutive initialisations, numeric segments, non-ASCII |

---

## What current codegen tests look like

All codegen tests construct `[]introspect.Table` inline, call `Generate`, and assert the exact output string. Example:

```go
func TestGenerateSingleTable(t *testing.T) {
    tables := []introspect.Table{
        {
            Schema: "public",
            Name:   "users",
            Columns: []introspect.Column{
                {Name: "id",   DataType: "integer", IsNullable: false, OrdinalPos: 1},
                {Name: "name", DataType: "text",    IsNullable: false, OrdinalPos: 2},
            },
        },
    }

    got, err := Generate(tables, Config{Package: "schema", DotImport: true})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    want := "// Code generated by gosq-codegen; DO NOT EDIT.\n\npackage schema\n\nimport . \"github.com/libliflin/gosq\"\n\nvar Users = NewTable(\"users\")\n\nvar (\n\tUsersID   = NewField(\"users.id\")\n\tUsersName = NewField(\"users.name\")\n)\n"
    if string(got) != want {
        t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, want)
    }
}
```

Key points:
- `want` uses `\t` for indentation (gofmt tabs, not spaces)
- The `// Code generated by gosq-codegen; DO NOT EDIT.` header is always the first line
- Ends with exactly one `\n`
- `Generate` applies `go/format` internally — output is deterministic and must match exactly
- **Important:** `go/format` aligns `=` signs within `var (...)` blocks using spaces. In multi-column tables, `UsersID   = NewField(...)` has trailing spaces before `=`. When asserting exact output for multi-column tables, paste the actual formatted output rather than guessing spacing. For large tests (like `TestGenerateProductionScale`), assert identifier presence and NewField argument presence separately rather than exact full-string match — see the production scale test for the pattern.

`toExported` is directly tested in `TestToExported` (table-driven, 43 subtests) covering all 17 initialisms, all known edge cases, consecutive initialisations (`api_id` → `APIID`), numeric version segments (`order_v2` → `OrderV2`), and non-ASCII (`éclat` → `Éclat`). New identifier transformation behavior belongs here.

---

## The five collision detection tests

The project tests all five ways identifier collisions can occur in generated output:

1. **Table-table collision** (`TestGenerateTableCollision`): `user_data` and `user__data` both map to `UserData`.
2. **Column-column collision** (`TestGenerateColumnCollision`): `my_field` and `my__field` both map to `MyField` within the same table.
3. **Cross-table field collision** (`TestGenerateCrossTableFieldCollision`): `a.b_c` and `a_b.c` both produce field ident `ABC`.
4. **Table-field collision** (`TestGenerateTableFieldCollision`): table `users_id` and field `users.id` both produce ident `UsersID`.
5. **Field-vs-prior-table collision** (`TestGenerateFieldPriorTableCollision`): table `_users_name` (sorts early due to `_` prefix) registers ident `UsersName`; table `users` with column `name` later produces field ident `UsersName` — collides with already-registered table ident.

When adding collision tests, follow the same inline-construction pattern and test that `Generate` returns a non-nil error. Don't assert the error string — just that an error occurred.

---

## CI

CI runs all tests automatically on every push and pull request via `.github/workflows/ci.yml`:

```
go build ./...
go vet ./...
go test ./...
staticcheck ./...
```

Module caching is enabled (`cache: true` on `actions/setup-go`). Staticcheck is installed fresh each run via `go install ... @latest` (not cached, adds ~20–30 seconds — a known CI improvement opportunity).

---

## Test commands

```bash
go test ./...               # Run all tests
go test ./... -v            # Verbose — see each test name
go test ./... -run TestGen  # Run matching tests only
go test ./... -count=1      # Disable caching (always re-run)
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

Integration tests should use `//go:build integration` and require `TEST_DSN`. In CI, set `TEST_DSN` from the `services: postgres:` container. Run them with: `go test -tags integration ./...`. Document the `TEST_DSN` variable and local Docker instructions in the test file's comments.

---

## No testdata/ directory

There are no golden files in `testdata/`. For a project this size with a single known output format, inline string assertions are clearer. If the output format grows complex enough that inline strings become unwieldy, introduce golden files at that point — not before.

---

## The uncovered 1.4%

The only uncovered code is the `format.Source` error return path in `Generate`. This is unreachable from any valid input — `format.Source` only errors if the source has a syntax error, which can't happen if `Generate` is working correctly. It's not worth testing.

---

## What's left to test

The unit test correctness gaps are closed. The major remaining work is **integration tests against a real Postgres database in CI**:

1. **DDL fixtures** in `testdata/schemas/` — real SQL schemas representing production patterns
2. **`//go:build integration` tests** in `internal/introspect/` — load DDL, call `Tables`, verify results
3. **End-to-end pipeline test** — DDL → introspect → codegen → write to temp dir → `go build` in a module that imports gosq
4. **`services: postgres:` in CI** — real Postgres instance in GitHub Actions

This is the single highest-value testing work remaining. It validates the entire promise of the tool — not just "does codegen produce correct strings" but "does the tool actually work against a real database and produce code that compiles."

Unit test coverage of `introspect.Tables` will go from 0% to meaningful coverage once integration tests exist.
