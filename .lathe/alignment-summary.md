# Alignment Summary

Read this before starting cycles. It summarizes the judgment calls baked into the agent so you can gut-check them before running.

---

## Who this serves

**gosq users** — Go developers who've adopted the gosq query builder and want their schema auto-generated. The tool works end-to-end: `go install`, run the CLI against their database, get a `schema.go` file. The correctness work is done — all known edge cases are handled and tested. CI is in place and the badge is visible in the README.

**Contributors** — developers working on gosq-codegen itself (currently the author). The codebase is clean, well-structured, and well-tested. CI runs automatically on every push and PR: go build, go vet, go test, staticcheck. Contributors get automated feedback within ~1–2 minutes.

**The gosq project** — gosq-codegen validates gosq's `NewTable`/`NewField` API by using it from the outside. A robust generator is a vote of confidence in the library design.

---

## Key tensions and how they were resolved

**Stability vs. improving identifiers.** Identifier changes in generated output rename variables across users' codebases. The naming conventions are correct and comprehensive (17 initialisms, 43 test cases). The agent should favor stability — only change naming logic for a clear bug (produces non-compilable or semantically wrong output), and document it as a renaming change.

**Edge-case completeness vs. simplicity.** The current CLI flag surface (`-dsn`, `-out`, `-pkg`, `-schema`, `-dot-import`, `-version`) is the right size. New flags should only appear when a real user can't accomplish something they need — not for hypothetical requirements.

**Fast local tests vs. real database confidence.** Two tiers: unit tests (`go test ./...`) run fast with no database — always. Integration tests (`//go:build integration`) run against real Postgres in CI via `services: postgres:`, with DDL fixtures in `testdata/schemas/`. The integration tier is the next major piece of work — `introspect.Tables` has 0% automated coverage and the full pipeline has never been tested against a real database automatically.

**CI speed vs. CI coverage.** Current CI is ~60–90 seconds. The staticcheck installation step adds ~20–30 seconds and is not cached. Switching to a cached or version-pinned staticcheck would improve contributor feedback time without sacrificing any coverage. This tension should be resolved in favor of speed without sacrificing the staticcheck step.

---

## Current focus (after ~37 cycles)

The project has been through ~37 cycles. All correctness and core infrastructure work is complete:

- Non-ASCII column names: fixed (rune-slicing in `toExported`)
- Blank identifier table names: error-on-generate
- Misleading introspect test stub: removed
- Cross-table field identifier collision: detected
- Table-field identifier collision: detected
- Production-scale test (17 tables, 108 columns): added
- Blank column identifier (`_`): documented and tested
- Multi-underscore column collision: tested
- Field-vs-prior-table collision: tested; coverage 98.6%
- Consecutive initialisations + numeric version segments in `TestToExported`: 43 subtests
- `staticcheck` confirmed clean
- **CI added**: GitHub Actions with go build, go vet, go test, staticcheck (cycles 33–34)
- **Module caching added** to CI (`cache: true` on setup-go)
- **README CI badge** added (cycle 36)
- **`-schema` flag behavior** documented in README (cycle 35)

**The agent's next priority is integration testing against a real Postgres database in CI.** The tool's full pipeline — DDL → introspect → codegen → compilable Go — has never been tested automatically against a real database. This is the biggest remaining confidence gap.

The work:
1. Add `testdata/schemas/*.sql` DDL fixtures (e-commerce, non-ASCII columns, PostGIS types, multi-schema)
2. Add `//go:build integration` tests that load fixtures into real Postgres, run introspect, pipe through codegen, verify output compiles
3. Add `services: postgres:` to `.github/workflows/ci.yml`
4. Keep `go test ./...` fast and DB-free for local development

After integration tests are in place, remaining work is incremental: pin staticcheck, update Actions versions, add GoDoc examples.

---

## What could be wrong

**staticcheck installation is slow and unpinned.** The current approach works but is suboptimal. Pinning to a version or caching the binary would both improve CI and make the staticcheck version explicit in the repo's history.

**Branch protection unknown.** Whether the default branch requires PR reviews before merge can't be verified without GitHub API access. If the repo is unprotected and autonomous cycles are running, changes could land on main without review. Enable branch protection (require PR review) before running many cycles autonomously.

**The `format.Source` error path** (1.4% uncovered) is unreachable from any valid input. Not a gap.

**No end-to-end pipeline test against a real database.** The tool has never been automatically tested against a real Postgres instance. `introspect.Tables` has 0% automated coverage. The full pipeline (DDL → introspect → codegen → compile) is verified only by hand. Integration tests in CI with `services: postgres:` and DDL fixtures would close this gap completely.

**No released version tags.** Users who `go install @latest` get whatever main is. There are no semver tags, so the `-version` flag prints module version from build info only for installs after tagging. This isn't a bug, but it limits users' ability to pin to a specific version.
