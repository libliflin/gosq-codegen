# Alignment Summary

**For the project owner to review before starting cycles.**

---

## Who this serves

- **gosq users** — Go developers using the gosq query builder who want to stop hand-writing table/field declarations. They install this CLI, point it at their Postgres DB, and get a compilable Go file back. They care about correct identifiers, no crashes on unusual schemas, and staying in sync after migrations.
- **Contributors / maintainer** — currently the author. They need fast CI feedback and confidence that changes don't break the tool on schemas they didn't think to test.

---

## Key tensions

**Integration fixture simplicity vs. realistic coverage.** The unit tests (codegen_test.go) are thorough and use a 17-table, 108-column fixture with complex naming patterns. The integration tests (which hit real Postgres) use 2-table toy schemas. This gap matters: a gosq user who runs this against their real 30-table database is in territory the integration tests haven't covered. I've favored expanding integration coverage as the priority, since that's what closes the gap between "tests pass" and "tool handles real usage."

**Collision detection completeness vs. complexity.** The existing detection is comprehensive. I didn't favor expanding it further — the current cases are well-tested and the code is already non-trivial. New collision detection only makes sense if a real user can actually hit the undetected case.

---

## Current focus

The agent will prioritize closing the gap between what the unit tests simulate and what the integration tests actually verify. Specifically:

1. The integration test fixtures are minimal (2–4 tables). A fixture that exercises the full pipeline with 10+ tables, mixed naming patterns, views (which should be excluded), and digit-prefixed columns would give real confidence in the tool.
2. Views in the same schema are explicitly excluded by the SQL query, but no test verifies this against real Postgres.
3. `introspect_test.go` is an empty placeholder — fast-path unit coverage for introspect is entirely absent.

After those gaps are closed, the project reaches stage 3 (battle-tested) and the agent should shift to DX improvements and documentation.

---

## What could be wrong

- **I assumed the repo is public** based on the GitHub URL pattern. If it's private, the prompt-injection risk is lower and branch protection matters less.
- **Branch protection status is unknown.** I couldn't check the actual GitHub repo settings. The alignment summary recommends the owner verify that the default branch requires PR reviews before starting autonomous cycles.
- **The `coverage.out` file is committed** to the repo (it appears in the file listing). This is unusual — typically coverage output is gitignored. I've flagged it in the agent's rules but didn't investigate further.
- **The parent `gosq` library** is a dependency but I haven't read it. The generated output must be compatible with `gosq.NewTable` and `gosq.NewField` — if those signatures change, this tool breaks. The agent will notice if the test stubs stop matching.
- **Multi-schema usage** (running with `-schema` multiple times to generate separate packages) is documented in the README but has no test coverage. A user following that pattern is in untested territory.
