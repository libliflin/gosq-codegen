# Testing conventions in gosq-codegen

This file answers the question: how does *this project* test, and what should new tests look like?

---

## Two test tiers

### Unit tests — no Postgres required

Location: `internal/codegen/codegen_test.go`  
Run with: `go test ./...`

These test `codegen.Generate` and `toExported` directly. They construct `[]introspect.Table` inline — no fixtures, no files. The pattern is:

```go
func TestGenerateSomething(t *testing.T) {
    tables := []introspect.Table{
        {Schema: "public", Name: "users", Columns: []introspect.Column{
            {Name: "id", DataType: "integer", OrdinalPos: 1},
        }},
    }
    got, err := Generate(tables, Config{Package: "schema", DotImport: true})
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    want := "..."
    if string(got) != want { t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, want) }
}
```

Error cases use the same structure but check `err != nil`:
```go
_, err := Generate(tables, Config{...})
if err == nil { t.Fatal("expected error for ..., got nil") }
```

`TestToExported` uses a table-driven pattern with `t.Run(tc.in, ...)` — new `toExported` cases go there.

`TestGenerateProductionScale` (17 tables, 108 columns) tests the full codegen path with realistic naming complexity, plus determinism (calls Generate twice, compares output).

`TestGenerateCompiles` writes the generated source into a temp module with a minimal gosq stub and runs `go build` — no network required. This is the gold standard for verifying generated output.

### Integration tests — requires Postgres

Location: `internal/introspect/integration_test.go`  
Build tag: `//go:build integration`  
Run with: `go test -tags integration ./...`  
Requires: `TEST_DSN` environment variable pointing to a live Postgres instance.

These load SQL fixtures from `testdata/schemas/` into temporary schemas (named `gosq_*`), call `introspect.Tables`, and assert on the returned data. They clean up after themselves via `t.Cleanup`.

**Existing fixtures:**
- `testdata/schemas/ecommerce.sql` — 2 tables (users, orders), NOT NULL and nullable columns, diverse types. Used by `TestTablesEcommerce` and `TestPipelineEcommerce`.
- `testdata/schemas/non_ascii.sql` — 1 table (articles), columns with accented characters. Used by `TestTablesNonASCII`.
- `testdata/schemas/reporting.sql` — 1 table (reports). Used by `TestTablesSchemaIsolation`.

**How to add an integration test:**
1. Write a new `.sql` fixture in `testdata/schemas/` — DDL only, no data, with a header comment explaining the purpose and what edge cases it covers.
2. Add a test function in `integration_test.go` that:
   - Calls `openIntegrationDB(t)` for the connection
   - Creates a uniquely-named schema (`gosq_<purpose>_test`)
   - Loads the fixture via a dedicated `db.Conn` with `SET search_path TO <schema>`
   - Calls `introspect.Tables(ctx, db, schema)`
   - Asserts on the result
   - Uses `t.Cleanup` to drop the schema
3. Optionally runs the full pipeline (introspect → codegen.Generate → go build in temp dir) as `TestPipelineEcommerce` does.

### The empty introspect unit test file

`internal/introspect/introspect_test.go` contains only `package introspect`. It's a placeholder. Unit-level tests for introspect behavior (e.g., testing the SQL query construction or scan logic with a mock `*sql.DB`) could go here but none exist yet. Adding tests here would give contributors fast feedback without needing Postgres.

---

## What the integration fixtures are missing

The current fixtures are minimal. The unit tests simulate complex schemas (17 tables, digit-prefixed columns, initialisations, etc.) but the integration tests only verify simple cases against real Postgres. Gaps:

- **Views in the schema** — the introspect query filters `t.table_type = 'BASE TABLE'`, but no integration test verifies views are excluded.
- **Large/complex fixture** — no integration test exercises a realistic schema (10+ tables, mixed naming, nullable patterns) through the full introspect → codegen → compile pipeline.
- **Non-ASCII table names** — `TestTablesNonASCII` covers non-ASCII *column* names but not non-ASCII *table* names.
- **Digit-prefixed columns in real Postgres** — tested in unit tests but not integration tests.
- **Empty schema** — no integration test for what happens when the target schema exists but has no base tables.

---

## CI behavior

`go test ./...` runs all unit tests (no build tag needed).  
`go test -tags integration ./...` runs both unit and integration tests (requires `TEST_DSN`).  

In CI (`.github/workflows/ci.yml`), a Postgres 16 service is spun up and `TEST_DSN` is set automatically. Both test runs happen in sequence.

---

## Generated-code compilation test pattern

Both `TestGenerateCompiles` (unit) and `TestPipelineEcommerce` (integration) verify that generated output actually compiles. They:
1. Write the generated `.go` file into `t.TempDir()`
2. Create a minimal gosq stub module (no network) with `NewTable` and `NewField` functions
3. Write a `go.mod` that replaces `github.com/libliflin/gosq` with the local stub
4. Run `exec.Command("go", "build", "./schema")` from the temp dir

New tests that want to verify compilation should follow this pattern exactly. The gosq stub is intentionally minimal — it just needs the two constructor signatures.
