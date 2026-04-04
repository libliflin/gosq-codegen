# Alignment Summary

Read this before starting cycles. It summarizes the judgment calls baked into the agent so you can gut-check them before running.

---

## Who this serves

**gosq users** — Go developers who've adopted the gosq query builder and want their schema auto-generated. The tool works end-to-end: `go install`, run the CLI against their database, get a `schema.go` file. The agent's job is no longer "make it work" — it's "make sure it handles what real production databases throw at it, at realistic scale."

**Contributors** — developers working on gosq-codegen itself (currently the author). The codebase is clean, well-structured, and well-tested for the cases it covers. They need new edge cases handled correctly with tests that document the behavior.

**The gosq project** — gosq-codegen validates gosq's `NewTable`/`NewField` API by using it from the outside. A robust generator is a vote of confidence in the library design.

---

## Key tensions and how they were resolved

**Stability vs. improving identifiers.** Identifier changes in generated output rename variables across users' codebases. The naming conventions are correct and consistent (ASCII and non-ASCII). The agent should favor stability — only change naming logic for a clear bug (produces non-compilable or semantically wrong output), and document it as a renaming change.

**Edge-case completeness vs. simplicity.** The current CLI flag surface (`-dsn`, `-out`, `-pkg`, `-schema`, `-dot-import`, `-version`) is the right size. New flags should only appear when a real user can't accomplish something they need — not for hypothetical requirements.

**Unit testability vs. integration confidence.** `introspect.Tables` has 0% automated coverage because it requires a live Postgres database — this is correct and expected. The agent should not introduce a test-DB dependency. The `codegen` package is fully unit-tested at 98.5% — that's where test effort belongs.

---

## Current focus (after cycle 28)

The project has been through 28 cycles. The core correctness work is done:
- Non-ASCII column names: fixed (cycle 24, rune-slicing in `toExported`)
- Blank identifier table names: error-on-generate (cycle 25)
- Misleading introspect test stub: removed (cycle 26)
- Cross-table field identifier collision: detected (cycle 27)
- Table-field identifier collision: detected (cycle 28)

The agent is now entering the phase where the highest-value question is scale and diversity: **does the tool handle real production database schemas — many tables, many columns, diverse naming patterns — without surprises?** The test suite still uses tiny, clean, hand-crafted inputs.

The agent should prioritize building realistic-scale test inputs over further polish. You can construct a `[]introspect.Table` with 15-20 tables and 100 columns right now — no live database needed. That's more valuable than another README edit or doc cleanup.

---

## What could be wrong

**Scale behavior.** I've assessed the tool as "correct" based on the existing tests, but those tests are all small (2-3 tables, 1-3 columns). Collision detection uses maps that scale correctly, and sorting is trivially correct at scale — but I haven't verified that the output at 50+ tables is deterministic, correctly formatted, and collision-free. It very likely is. But "very likely" isn't tested.

**Non-public schema behavior.** The `-schema` flag passes the schema name to `information_schema.columns` correctly. But the generated output doesn't include the schema name anywhere — field identifiers are `TableIdent + ColIdent` regardless of schema. A user running `-schema reporting` and `-schema public` on two different schemas that share table names would get identical identifier names for different tables. Is this the right behavior? Is it documented? I'm not certain.

**`staticcheck` status.** The snapshot confirms `go vet` is clean. I haven't verified `staticcheck`. It's likely clean given the codebase style, but it's a quick check the agent should do.

**Column blank identifier behavior.** A column named `_` produces `toExported("_") = "_"` for the column part. The full field ident is `TableIdent + "_"` (e.g., `Items_`). This is a valid exported identifier in Go and should work fine. But it hasn't been explicitly tested. It's a lower priority than scale testing since `_` as a column name is extremely unlikely in practice.
