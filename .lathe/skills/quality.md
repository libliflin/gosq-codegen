# Go Code Quality

## Static Analysis

Run these checks before marking any cycle complete:

```bash
go vet ./...           # Built-in static analysis
go build ./...         # Compilation check
```

If available:
```bash
staticcheck ./...      # Extended static analysis
golangci-lint run      # Meta-linter (if configured)
```

## Go Idioms

- **Error handling:** Always check errors. Use `fmt.Errorf("context: %w", err)` for wrapping.
- **Naming:** Short names for short scopes. `r` not `reader` in a 3-line function. But `reader` in a struct field.
- **Interfaces:** Define interfaces where they're used, not where they're implemented. Keep them small (1-3 methods).
- **Zero values:** Design structs so the zero value is useful. Don't require constructor functions when the zero value works.
- **Exported vs unexported:** Only export what's part of the public API. Start unexported, promote later.

## Error Patterns

```go
// Sentinel errors for expected conditions
var ErrNotFound = errors.New("not found")

// Wrap with context
return fmt.Errorf("loading config %s: %w", path, err)

// Check wrapped errors
if errors.Is(err, ErrNotFound) { ... }
```

## Package Organization

- One package per directory
- Package names are lowercase, single word, no underscores
- `internal/` for packages that shouldn't be imported externally
- Avoid `utils`, `helpers`, `common` — put functions where they belong
