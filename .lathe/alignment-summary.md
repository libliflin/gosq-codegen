# Alignment Summary

Read this before starting cycles. It summarizes the judgment calls baked into the agent so you can gut-check them before running.

---

## Who this serves

**gosq users** — Go developers who've adopted the gosq query builder and want their schema auto-generated. The tool works end-to-end: `go install`, run the CLI against their database, get a `schema.go` file. The agent's job is no longer "make it work" — it's "make sure it handles what real production databases throw at it."

**Contributors** — developers working on gosq-codegen itself (currently the author). The codebase is clean and well-tested for the cases it covers. They need new edge cases handled correctly with tests that document the behavior.

**The gosq project** — gosq-codegen validates gosq's `NewTable`/`NewField` API by using it from the outside. A robust generator is a vote of confidence in the library design.

---

## Key tensions and how they were resolved

**Stability vs. improving identifiers.** Identifier changes in generated output rename variables across users' codebases. The naming conventions are currently correct and consistent for ASCII inputs. The agent should favor stability — only change naming logic for a clear bug (produces non-compilable or semantically wrong output), and document it as a renaming change.

**Edge-case completeness vs. simplicity.** The current CLI flag surface (`-dsn`, `-out`, `-pkg`, `-schema`, `-dot-import`, `-version`) is the right size. New flags should only appear when a real user can't accomplish something they need — not for hypothetical requirements.

**Unit testability vs. integration confidence.** `introspect.Tables` has 0% automated coverage because it requires a live Postgres database — this is correct and expected. The agent should not introduce a test-DB dependency. The `codegen` package is fully unit-tested at 98.2% — that's where test effort belongs.

---

## Current focus

The project has been through 23 cycles. The most recent meaningful improvements: collision detection (cycle 23), package name validation (cycle 22), `-version` flag (cycle 20), success message with table count (cycle 19), view filtering (cycle 10), context timeout (cycle 11).

After 23 cycles of improvement — many of them polish cycles — the highest-value open question is: **does this tool handle real production database schemas without surprises?** The test suite uses small, clean, ASCII-only inputs. Production databases don't.

Specific gaps to investigate:

1. **Non-ASCII column names.** `toExported` uses `part[:1]` (byte-slicing). A column name starting with a multi-byte UTF-8 character produces a broken byte at position 0. This is a real potential bug for any user with non-ASCII Postgres identifiers (common in European/Asian database schemas with quoted column names).

2. **Blank identifier collision.** A table named `_` produces `var _ = NewTable("_")`, which silently discards the value via the blank identifier. This is a different kind of failure from a name collision — it produces valid Go that compiles but can't be used. Should it error?

3. **`TestTableStructure` in `introspect_test.go`.** This test exercises zero code paths. It's not harmful, but it's misleading. Whether to remove it or expand it is a judgment call.

The agent should prefer digging into one of these real gaps over another round of doc polish.

---

## What could be wrong

**The non-ASCII assessment.** I identified the byte-slicing issue in `toExported` from reading the code. I haven't verified that Postgres actually sends non-ASCII column names over the wire when using `lib/pq` with `information_schema.columns`. It's possible the driver normalizes them or that such column names are rare enough in practice that no user has ever hit this. The issue is real in the code, but whether it matters in production is uncertain.

**The blank identifier edge case.** A table named `_` is unusual. The probability of a real user having this in their schema is low. But the silent-discard behavior (valid Go that can't be used) is qualitatively different from a collision error, and it's worth at least documenting whether the current behavior is intentional.

**Non-`public` schema behavior.** The `-schema` flag passes the schema name to the `information_schema.columns` query correctly. But it's only been manually tested (if at all) against non-public schemas. Users pointing the tool at `myapp` or `reporting` schemas haven't been considered.

**The 30-second timeout.** Fine for most schemas. A user with a very large schema on a slow network could hit it, but there's no evidence this has happened. Flag it only if a user reports it.
