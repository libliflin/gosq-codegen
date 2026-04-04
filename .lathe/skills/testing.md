# Testing in gosq-codegen

This file exists to answer: how does *this* project test, and what should new tests look like?

---

## The fundamental split

Two packages. Two completely different testing strategies.

### `internal/introspect`

The real job of this package is to query `information_schema.columns` in a live Postgres database. That can't be unit-tested without a real database.

**What to test:** Struct construction and behavior that doesn't require a DB. Column ordering by `OrdinalPos`, nil-safety on empty column slices, correct field assignment when building `Table` values from data you control.

**What not to do:** Don't mock `database/sql` to simulate query results. The mock/real divergence risk is high and the tests would be fragile. If you need confidence on the query path, test it manually against a real database or add an integration test (clearly labeled `//go:build integration`) that requires a `TEST_DSN` environment variable.

**Existing test:** `introspect_test.go` constructs a `Table` inline and checks `len(tbl.Columns)` and `tbl.Columns[2].IsNullable`. That is the right pattern. New tests should look like this.

### `internal/codegen`

This package turns `[]introspect.Table` into a `[]byte` of Go source. It requires no database — you construct tables inline and assert on the output. This is where most test effort belongs.

**What to test:** 
- The shape and content of generated Go source for a known input
- That output is gofmt-clean (i.e., `go/format` applied to the output produces the same bytes — or equivalently, that `Generate` applies `go/format` itself)
- That output is deterministic: call `Generate` twice with the same input, get identical bytes
- Edge cases: empty table list, table with no columns, column name that needs identifier escaping, multiple tables (are they sorted?)

**Current state:** `codegen_test.go` has two tests (`TestGenerateEmpty`, `TestGenerateSingleTable`) that call `Generate` and immediately discard the output with `_ = out`. Once `Generate` is implemented, these placeholders must become real assertions — not be deleted.

---

## What real codegen tests look like

For a single-table case, assert against the expected string directly:

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

    want := `package schema

import . "github.com/libliflin/gosq"

var Users = NewTable("users")

var (
	UsersID   = NewField("users.id")
	UsersName = NewField("users.name")
)
`
    if string(got) != want {
        t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", got, want)
    }
}
```

Note: the exact whitespace in `want` matters — use tabs for indentation (gofmt convention). If `Generate` runs `go/format` internally, the output will be gofmt-clean; your test string must match exactly.

---

## Table-driven tests for codegen

Once the basic cases pass, use table-driven tests for edge cases:

```go
tests := []struct {
    name   string
    tables []introspect.Table
    cfg    Config
    want   string
}{
    {"empty", nil, Config{Package: "schema", DotImport: true}, "package schema\n"},
    // add cases: no columns, DotImport false, multiple tables (check sort order), etc.
}
```

---

## Test commands

```bash
go test ./...               # Run all tests
go test ./... -v            # Verbose — see each test name
go test ./... -run TestGen  # Run matching tests only
go test ./... -count=1      # Disable caching (always re-run)
```

There are no integration tags configured yet. If you add `//go:build integration` tests, document the `TEST_DSN` variable they require.

---

## No testdata/ directory (yet)

There are no golden files in `testdata/`. For a project this small with a single known output format, inline string assertions are clearer and easier to maintain. If the output format grows complex enough that inline strings become unwieldy, introduce golden files at that point — not before.
