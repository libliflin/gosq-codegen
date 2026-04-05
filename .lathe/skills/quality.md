# Code Quality in gosq-codegen

This file exists to answer: what does "quality" mean specifically for this project, beyond standard Go idioms?

---

## The hard requirement: generated code must be gofmt-clean

`codegen.Generate` returns `[]byte` of Go source. That source will be written to a file and committed to users' version control. It must:

1. Compile without errors in a real Go module that imports gosq
2. Be identical to what `gofmt` would produce

This is already guaranteed: `Generate` pipes output through `go/format` before returning it. Never remove this step.

---

## Determinism is not optional

Generated files live in version control. If running gosq-codegen twice against the same schema produces different output, users get spurious diffs on every run. This destroys trust fast.

**Required behavior:** Same `[]introspect.Table` input + same `Config` → identical `[]byte` output, always.

**How it's achieved:** Tables are sorted by name before rendering. Columns are rendered in `OrdinalPos` order (as returned by `information_schema.columns`). No map iteration order anywhere in the rendering path. This is already correct — don't change it.

---

## Identifier naming: stability beats perfection

Generated variable names derive from table and column names. PostgreSQL names are typically `snake_case`. Go exported identifiers are `PascalCase` with initialisms.

The current `toExported` function is:
- Correct for all common ASCII snake_case patterns (`users.id` → `UsersID`, `orders.created_at` → `OrdersCreatedAt`)
- Correct for non-ASCII input starting with multi-byte UTF-8 characters (uses `[]rune` slicing instead of byte slicing)
- Tested against 43 subtests in `TestToExported` (including `{"éclat", "Éclat"}`, consecutive initialisations, numeric version segments)
- Stable — changing it renames variables in users' codebases

**When NOT to change `toExported`:** Aesthetic disagreements, speculative coverage, or theoretical correctness for inputs that don't appear in real schemas.

**When to change `toExported`:** A user reports that a real column name produces a wrong or non-compilable identifier. Fix it, test it, document it as a naming change.

**The current initialism list (17 entries):** `id`, `url`, `uri`, `http`, `https`, `sql`, `api`, `uid`, `uuid`, `ip`, `io`, `cpu`, `xml`, `json`, `rpc`, `tls`, `ttl`. Adding a new initialism is a breaking change for users who have the corresponding column names — document it. The list is tested via 43 subtests in `TestToExported`, including consecutive initialism cases (`api_id` → `APIID`, `oauth_api_url` → `OauthAPIURL`) and numeric version segments (`order_v2` → `OrderV2`).

---

## Error messages must say *what* failed

Error messages should name the specific table or column that caused the failure:

```go
// Good — names the problem
return nil, fmt.Errorf("tables %q and %q both produce identifier %q", prev, tbl.Name, ident)
return nil, fmt.Errorf("table %q produces blank identifier %q; it cannot be referenced in Go", tbl.Name, ident)

// Not helpful
return nil, fmt.Errorf("identifier collision detected")
```

`main.go` prefixes errors with `"gosq-codegen: "` and the operation name (e.g., `"gosq-codegen: introspect: "`). Internal packages (`introspect`, `codegen`) should not include their own package name in error strings — callers add context via wrapping. This is already correct; maintain the pattern.

---

## Collision detection covers all five cases

`Generate` detects all five ways identifier collisions can surface in generated output:

1. Two tables produce the same `TableIdent` (e.g. `user_data` and `user__data` → both `UserData`)
2. Two columns in the same table produce the same `ColIdent`
3. Two different table+column pairs produce the same full field ident (`TableIdent + ColIdent`) across tables
4. A table's ident matches a field ident from a previously-processed table (e.g. table `users_id` and field `users.id` → both `UsersID`)
5. A field ident matches a table ident registered in an earlier iteration — table `_users_name` (sorts before `users`) registers ident `UsersName`; field `users.name` then collides with it

All five are caught before rendering. Don't weaken this — silent identifier collisions produce generated Go that fails the user's build with a confusing `DO NOT EDIT` file as the locus.

---

## Keep `internal/` internal

`introspect` and `codegen` are `internal/` packages. They are not part of any public API. Only export what `main.go` (or tests) actually uses. The current exported surface:
- `introspect.Table`, `introspect.Column` — needed by codegen
- `introspect.Tables` — called by main
- `codegen.Config` — passed by main
- `codegen.Generate` — called by main

Don't add exported symbols without a concrete caller.

---

## Static analysis

CI runs these automatically on every push and PR. Before marking a cycle complete, verify locally:

```bash
go build ./...    # Must succeed
go test ./...     # Must pass
go vet ./...      # Must be clean
staticcheck ./... # Must be clean
```

Don't suppress vet or staticcheck warnings with `//nolint` or blank identifiers unless the warning is provably a false positive and you document why.

---

## The `// Code generated` header

Every generated file begins with:

```
// Code generated by gosq-codegen; DO NOT EDIT.
```

This is the Go convention (per `cmd/go` documentation). Many tools — linters, editors, code review — use it to suppress warnings for machine-generated files. Never remove it from `Generate`.

---

## CI quality

The CI workflow (`.github/workflows/ci.yml`) runs `go build`, `go vet`, `go test`, and `staticcheck` on every push and pull request. Module caching is enabled.

**Known CI quality gap:** `staticcheck` is installed via `go install honnef.co/go/tools/cmd/staticcheck@latest` on each run — not version-pinned and not cached, adding ~20–30 seconds. This is a CI speed/reliability issue, not a correctness issue. A future cycle should address it by switching to `staticcheck-action` or pinning to a specific version.

**Upcoming maintenance:** `actions/checkout@v4` and `actions/setup-go@v5` use Node.js 20, which GitHub is deprecating for Actions in September 2026. These action versions will need to be updated before the forced cutover. Not urgent, but track it.
