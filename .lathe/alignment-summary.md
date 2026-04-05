# Alignment Summary

Read this before starting cycles. It summarizes the judgment calls baked into the agent so you can gut-check them before running.

---

## Who this serves

**gosq users** — Go developers who've adopted the gosq query builder and want their schema auto-generated. The tool works end-to-end: `go install`, run the CLI against their database, get a `schema.go` file. The correctness work is done — all known edge cases are handled and tested.

**Contributors** — developers working on gosq-codegen itself (currently the author). The codebase is clean, well-structured, and well-tested. They need CI to get automated feedback on their changes. Right now there's none.

**The gosq project** — gosq-codegen validates gosq's `NewTable`/`NewField` API by using it from the outside. A robust generator is a vote of confidence in the library design.

---

## Key tensions and how they were resolved

**Stability vs. improving identifiers.** Identifier changes in generated output rename variables across users' codebases. The naming conventions are correct and comprehensive (17 initialisms, 43 test cases). The agent should favor stability — only change naming logic for a clear bug (produces non-compilable or semantically wrong output), and document it as a renaming change.

**Edge-case completeness vs. simplicity.** The current CLI flag surface (`-dsn`, `-out`, `-pkg`, `-schema`, `-dot-import`, `-version`) is the right size. New flags should only appear when a real user can't accomplish something they need — not for hypothetical requirements.

**Unit testability vs. integration confidence.** `introspect.Tables` has 0% automated coverage because it requires a live Postgres database — this is correct and expected. The agent should not introduce a test-DB dependency. The `codegen` package is fully unit-tested at 98.6% — that's where test effort belongs.

---

## Current focus (after cycle 32)

The project has been through 32 cycles. All correctness work is complete:

- Non-ASCII column names: fixed (cycle 24, rune-slicing in `toExported`)
- Blank identifier table names: error-on-generate (cycle 25)
- Misleading introspect test stub: removed (cycle 26)
- Cross-table field identifier collision: detected (cycle 27)
- Table-field identifier collision: detected (cycle 28)
- Production-scale test (17 tables, 108 columns): added (cycle 29)
- Blank column identifier (`_`): documented and tested (cycle 30)
- Multi-underscore column collision: tested (cycle 30)
- Field-vs-prior-table collision: tested (cycle 31); coverage 98.6%
- Consecutive initialisations + numeric version segments in `TestToExported`: 43 subtests (cycle 32)
- `staticcheck` confirmed clean (cycle 31)

**The agent's next priority is CI/CD.** There is no `.github/workflows/` directory. The project has no automated validation pipeline — no workflow runs on push, no CI status on PRs. A minimal GitHub Actions workflow (`go build ./...`, `go test ./...`, `go vet ./...` on push and pull_request) is the single highest-value change available. It doesn't need to be elaborate; it needs to exist.

After CI exists, the next tier is: adding `staticcheck` to CI, documenting the non-public schema behavior, and — if real user feedback arrives — addressing any schema patterns not yet covered.

---

## What could be wrong

**CI gap is the biggest known unknown.** I've assessed the tool as "correct" based on the test suite, but without CI, there's no guarantee that the test suite actually runs on the version users install. A minimal workflow closes this completely.

**Non-public schema behavior.** The `-schema` flag generates output with identifiers based on table/column names only, not schema names. A user running `-schema reporting` and `-schema public` on two different schemas that share table names would get identical identifier names for different tables. This might be surprising. The behavior is consistent and deterministic but not documented. Worth a README note or a clear warning in the tool output.

**Branch protection unknown.** Whether the default branch is protected (requiring PR reviews before merge) can't be verified without GitHub API access. If the repo is public and unprotected, autonomous cycles could push directly to main — a risk worth asking the user about before starting.

**The `format.Source` error path** (1.4% uncovered) is unreachable from any valid input. Not a gap.
