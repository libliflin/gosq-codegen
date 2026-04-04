# Changelog

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
