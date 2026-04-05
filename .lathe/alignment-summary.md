# Alignment Summary

Read this before starting cycles. It summarizes the judgment calls baked into the agent so you can gut-check them before running.

---

## Who this serves

**gosq users** — Go developers who've adopted the gosq query builder and want their schema auto-generated. The tool is production-ready: `go install`, run the CLI against their database, get a `schema.go` file. Correctness, integration tests, and CI are all in place. The CI badge is green and visible in the README.

**Contributors** — developers working on gosq-codegen itself (currently the author). The codebase is clean, well-structured, and fully tested at both the unit and integration tiers. CI runs automatically on every push and PR: go build, go vet, go test, go test -tags integration (against real Postgres), staticcheck.

**The gosq project** — gosq-codegen validates gosq's `NewTable`/`NewField` API from the outside. A robust generator is a vote of confidence in the library design. This stakeholder benefits silently.

---

## Key tensions and how they were resolved

**Stability vs. improving identifiers.** Generated identifiers live in users' codebases. The naming conventions are correct and comprehensive (17 initialisms, 43 subtests). The agent favors stability — only change naming logic for a clear bug (produces non-compilable or semantically wrong output), and document it as a renaming change.

**Edge-case completeness vs. simplicity.** The current flag surface (`-dsn`, `-out`, `-pkg`, `-schema`, `-dot-import`, `-version`) is the right size. New flags only appear when a real user can't accomplish something they need.

**Fast local tests vs. real database confidence.** Two tiers are in place and working: unit tests (no DB, fast) and integration tests (`//go:build integration`, real Postgres, runs in CI). The integration tier now covers the full pipeline: introspect, non-ASCII columns, schema isolation, end-to-end DDL → codegen → compile. The remaining gap is coverage depth (small DDL fixtures, no view-exclusion test).

**CI speed vs. CI coverage.** Current CI takes ~60–90 seconds. Staticcheck installation (`go install @v0.5.1`) is not cached, adding ~20–30 seconds. Switching to a cached install would improve contributor feedback time without sacrificing coverage.

---

## Current focus (after 43 cycles)

The project is battle-tested. All prior priorities — correctness, collision detection, non-ASCII handling, integration tests, CI infrastructure, Node.js 22 actions — are complete. The CI badge is green.

**Remaining gaps, roughly in priority order:**

1. **No integration test for view exclusion.** The introspect query filters `table_type = 'BASE TABLE'`, but no test has ever verified this against a schema that actually contains views. A DDL fixture with a view alongside tables, plus an assertion that the view is absent from `Tables()` output, would close this.

2. **Small integration DDL fixtures.** The ecommerce fixture (2 tables, 8 columns) is minimal. The full pipeline test (`TestPipelineEcommerce`) exercises the same tiny schema. A larger DDL fixture (10–15 tables with diverse column naming patterns) would give meaningfully more confidence in the integration path.

3. **Empty `introspect_test.go`.** The file contains only `package introspect`. It's a vestigial placeholder. Options: delete it (one-line cleanup) or fill it with unit tests for `introspect` behaviors that don't require a real database (currently: none — `Tables` requires a DB, and `Table`/`Column` are pure data types). Deleting is lower-value than adding a real integration test elsewhere.

4. **No GoDoc `Example` functions.** Neither package has `Example` functions. A runnable example in `codegen_test.go` (package `codegen_test`) showing `Generate` in action would double as documentation and test.

5. **Staticcheck install is not cached.** Known issue, low priority since CI passes. Caching staticcheck would shave ~20–30 seconds off every CI run.

---

## What could be wrong

**The integration fixtures are small.** I assessed the project as "battle-tested" based on the completeness of the test strategy — unit, integration, and pipeline tests all exist. But the integration path has only been exercised against 2-table schemas. A future regression in the introspect query that only manifests with more complex schemas (many tables, deep column counts) might not be caught.

**Branch protection unknown.** Whether the default branch requires PR reviews before merge can't be verified from files alone. If unprotected, autonomous cycles could push directly to main. Enable branch protection (require PR review) before running many cycles autonomously.

**The `format.Source` error path (1.4% uncovered)** is unreachable from valid input — documented as such. Not a gap.

**No released version tags.** Users who `go install @latest` get whatever main is. The `-version` flag will print `(devel)` for local builds but correct version info for tagged installs. This isn't a bug, but it limits version pinning.

**View exclusion is assumed, not verified.** The `WHERE t.table_type = 'BASE TABLE'` clause in `introspect.Tables` looks correct, but it has never been tested against a real schema with views. If the clause were accidentally dropped or changed, no test would catch it.
