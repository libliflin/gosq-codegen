# Alignment Summary

Read this before starting cycles. It summarizes the judgment calls made during setup so you can gut-check them before the agent starts running.

---

## Who this serves

**gosq users** — Go developers who've adopted the gosq query builder and need to auto-generate schema declarations instead of hand-writing 450+ `NewField` calls. Their first experience with this tool right now is an immediate hard stop: "not yet implemented". This is the person the agent should be working for.

**Contributors** — developers working on gosq-codegen itself (currently the author). They have clean architecture but no behavior. They need real implementations to build on.

**The gosq project** — gosq-codegen validates gosq's API design by using it from the outside. A clean, working generator is a vote of confidence in the library.

---

## Key tensions and how they were resolved

**User wants it to work now vs. contributor wants clean internals.** Favored the user. The architecture is already good enough — `introspect` and `codegen` are cleanly separated. Fill in the behavior before adding more structure.

**Unit testability vs. integration confidence.** Favored keeping tests dependency-free. The `codegen` package (pure rendering logic) should be tested aggressively with inline assertions. The `introspect` package can't be meaningfully tested without Postgres — don't fake it. Don't add a test-DB dependency without explicit direction.

**Dot-import vs. Go linting conventions.** Both options kept via `Config.DotImport`. Default stays dot-import (matches gosq examples). The flag must work.

---

## Current focus

The agent should focus entirely on getting the tool to actually work. In order:

1. Implement `codegen.Generate` so it produces real, gofmt-clean Go source
2. Turn the placeholder `// TODO` tests in `codegen_test.go` into real assertions
3. Implement `introspect.Tables` with a real `information_schema.columns` query
4. Add the PostgreSQL driver to `go.mod`
5. Wire `main.go` to parse flags and run the full pipeline

Nothing at Layer 3+ (code quality, architecture, docs, features) is worth touching until the tool can produce output.

---

## What could be wrong

**The gosq library API.** `agent.md` assumes `NewTable(string)` and `NewField(string)` are the correct gosq API. This was inferred from the `main.go` doc comment and the README. If gosq's actual API differs (e.g., requires additional arguments, uses a different package path), the generated code format will be wrong. The agent should verify by checking the gosq source before generating output.

**The output file structure.** It's assumed the generator writes one file containing all tables. If gosq-codegen should write one file per table (e.g., `users.go`, `orders.go`), the `codegen.Generate` signature may need to change from `([]byte, error)` to something that maps table names to file content. The current signature was kept because it matches what's already in the code — but this is an assumption.

**The module path.** `go.mod` says `module github.com/libliflin/gosq-codegen`. The README says `go install github.com/libliflin/gosq-codegen@latest`. These match. But the gosq import path in generated code (`github.com/libliflin/gosq`) was inferred from the README link — it hasn't been verified against the actual gosq module.

**Identifier conversion rules.** The quality skill documents Go initialisms (`ID`, `URL`, etc.) for column name → variable name conversion. These are standard Go conventions, but the exact list (does `oauth` become `Oauth` or `OAuth`?) is a judgment call. The agent should use `golang.org/x/text/cases` or a simple approach and be consistent, not exhaustive.
