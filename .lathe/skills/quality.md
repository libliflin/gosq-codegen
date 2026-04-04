# Code Quality in gosq-codegen

This file exists to answer: what does "quality" mean specifically for this project, beyond standard Go idioms?

---

## The hard requirement: generated code must be gofmt-clean

`codegen.Generate` returns `[]byte` of Go source. That source will be written to a file and committed to users' version control. It must:

1. Compile without errors in a real Go module that imports gosq
2. Be identical to what `gofmt` would produce

The way to guarantee (2) is to pipe the output through `go/format` before returning it:

```go
import "go/format"

formatted, err := format.Source(src)
if err != nil {
    return nil, fmt.Errorf("formatting generated source: %w", err)
}
return formatted, nil
```

If `format.Source` returns an error, it means the generated source has a syntax error. Treat that as a bug in `Generate`, not in the caller.

---

## Determinism is not optional

Generated files live in version control. If running gosq-codegen twice against the same schema produces different output, users get spurious diffs on every run. This destroys trust.

**Required behavior:** Same `[]introspect.Table` input + same `Config` → identical `[]byte` output, always.

**How to achieve it:** Sort tables by name (alphabetically) before rendering. Within each table, render columns in `OrdinalPos` order (as returned by `information_schema.columns`). Don't use map iteration order anywhere in the rendering path.

---

## Identifier naming

Generated variable names are derived from table and column names. PostgreSQL names are typically `snake_case`. Go exported identifiers are `PascalCase`. The conversion must be consistent:

- `users` → `Users`
- `user_id` → `UserID` (note: "ID" not "Id" — follow Go initialisms: `ID`, `URL`, `HTTP`, `SQL`)
- `created_at` → `CreatedAt`
- `oauth_token` → `OauthToken` (no initialism for "oauth" unless explicitly listed)

Use `golang.org/x/text/cases` or a simple manual split-on-underscore approach. Whichever you choose, be consistent across tables and columns.

**Edge cases to be aware of** (handle them when they arise, not prematurely):
- Column names that start with a digit (not valid Go identifiers — prefix with `_` or handle explicitly)
- Column names that are Go reserved words (`type`, `func`, etc.)
- Column names with non-ASCII characters

---

## Error messages must say *what* failed

When `introspect` fails to query a table, or when `codegen` fails to format output, the error must include enough context to locate the problem:

```go
// Good
return fmt.Errorf("introspect table %q: %w", tableName, err)

// Not helpful
return fmt.Errorf("query failed: %w", err)
```

---

## Keep `internal/` internal

`introspect` and `codegen` are `internal/` packages. They are not part of any public API. Don't add exported symbols "just in case" — only export what `main.go` (or tests) actually uses. The bar for adding an exported function to an internal package is: does something outside this package need to call it right now?

---

## Static analysis

Before marking a cycle complete:

```bash
go build ./...   # Must succeed
go test ./...    # Must pass
go vet ./...     # Must be clean
```

If `staticcheck` is available:
```bash
staticcheck ./...
```

Don't suppress vet or staticcheck warnings with `//nolint` or blank identifiers unless the warning is provably a false positive and you document why.
