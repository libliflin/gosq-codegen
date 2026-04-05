# Changelog — Cycle N

## Who This Helps
- Stakeholder: gosq users working in enterprise Go shops that use Microsoft SQL Server
- Impact: they can now run `gosq-codegen -driver sqlserver -dsn "sqlserver://..." -schema dbo -out schema/` and get compilable Go output from their SQL Server schema

## Observed
- Supported drivers were: postgres, cockroach, mysql, mariadb, sqlite
- SQL Server is widely used in enterprise Go — the obvious next gap in "add support for more databases"
- The `information_schema` query already works on SQL Server; only the placeholder style differs (`@p1` vs `$1`/`?`)

## Applied
- Added `DialectSQLServer Dialect = "sqlserver"` to `internal/introspect/introspect.go`
- Replaced the `if d == DialectMySQL` placeholder branch with a `switch` that also handles `DialectSQLServer → "@p1"`
- Added `github.com/microsoft/go-mssqldb v1.9.8` as a direct dependency; imported as blank in `main.go`
- Added `sqlserver` case to the driver switch in `main.go`
- Updated `-driver` and `-schema` flag descriptions to include sqlserver
- Added `testdata/schemas/sqlserver_ecommerce.sql` — SQL Server DDL fixture (users, orders, active_users view)
- Added `TestPipelineSQLServerEcommerce` + `openSQLServerIntegrationDB` + `splitSQLServerStatements` to `integration_test.go`, gated on `TEST_SQLSERVER_DSN` (same pattern as MySQL before its CI service was added)

## Validated
- `go build ./...` — clean
- `go test ./...` — all pass (SQL Server test skips without TEST_SQLSERVER_DSN)
- `go vet ./...` — clean
- `staticcheck ./...` — clean

## Next
- Add a SQL Server service to CI (mcr.microsoft.com/mssql/server:2022-latest) so TestPipelineSQLServerEcommerce runs on every PR, completing the verification loop for SQL Server support
