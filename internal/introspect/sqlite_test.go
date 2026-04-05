// Package introspect_test contains SQLite-specific tests for introspect.Tables.
// These tests use an in-memory SQLite database and require no external service,
// so they run as part of the standard go test ./... suite.
package introspect_test

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libliflin/gosq-codegen/internal/codegen"
	"github.com/libliflin/gosq-codegen/internal/introspect"
	_ "modernc.org/sqlite"
)

// TestTablesSQLite creates an in-memory SQLite database with two tables and a
// view, then verifies that introspect.Tables returns only the base tables with
// correct column metadata. Views must be excluded.
func TestTablesSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	stmts := []string{
		`CREATE TABLE users (
			id      INTEGER NOT NULL,
			email   TEXT    NOT NULL,
			name    TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE orders (
			id         INTEGER NOT NULL,
			user_id    INTEGER NOT NULL,
			total      REAL    NOT NULL,
			created_at TEXT    NOT NULL
		)`,
		// View must be excluded from results.
		`CREATE VIEW active_users AS SELECT id, email FROM users`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("exec DDL: %v\nstatement: %s", err, stmt)
		}
	}

	tables, err := introspect.Tables(ctx, db, "main", introspect.DialectSQLite)
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}

	// Expect exactly 2 base tables, sorted alphabetically: orders, users.
	if len(tables) != 2 {
		names := make([]string, len(tables))
		for i, tbl := range tables {
			names[i] = tbl.Name
		}
		t.Fatalf("expected 2 tables, got %d: %v", len(tables), names)
	}
	if tables[0].Name != "orders" {
		t.Errorf("tables[0].Name = %q, want %q", tables[0].Name, "orders")
	}
	if tables[1].Name != "users" {
		t.Errorf("tables[1].Name = %q, want %q", tables[1].Name, "users")
	}

	// Schema field must match what was passed to Tables.
	for _, tbl := range tables {
		if tbl.Schema != "main" {
			t.Errorf("table %q: Schema = %q, want %q", tbl.Name, tbl.Schema, "main")
		}
	}

	// Verify users columns: name is nullable, others are NOT NULL.
	users := tables[1]
	if len(users.Columns) != 4 {
		t.Fatalf("users: expected 4 columns, got %d", len(users.Columns))
	}
	wantUsers := []struct {
		name     string
		nullable bool
	}{
		{"id", false},
		{"email", false},
		{"name", true},
		{"created_at", false},
	}
	for i, wc := range wantUsers {
		col := users.Columns[i]
		if col.Name != wc.name {
			t.Errorf("users.Columns[%d].Name = %q, want %q", i, col.Name, wc.name)
		}
		if col.IsNullable != wc.nullable {
			t.Errorf("users.Columns[%d].IsNullable = %v, want %v (column %q)", i, col.IsNullable, wc.nullable, col.Name)
		}
		if col.OrdinalPos != i+1 {
			t.Errorf("users.Columns[%d].OrdinalPos = %d, want %d", i, col.OrdinalPos, i+1)
		}
	}

	// Verify orders columns: all NOT NULL.
	orders := tables[0]
	if len(orders.Columns) != 4 {
		t.Fatalf("orders: expected 4 columns, got %d", len(orders.Columns))
	}
	for _, col := range orders.Columns {
		if col.IsNullable {
			t.Errorf("orders.%s: IsNullable = true, want false", col.Name)
		}
	}
}

// TestPipelineSQLite runs the full introspect → codegen → compile pipeline
// against an in-memory SQLite database. This verifies that DialectSQLite
// produces compilable Go output for a realistic schema with mixed naming
// patterns: snake_case columns, nullable columns, and a view that must
// be excluded from the output.
func TestPipelineSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	stmts := []string{
		`CREATE TABLE users (
			id           INTEGER NOT NULL,
			email        TEXT    NOT NULL,
			display_name TEXT,
			api_key      TEXT,
			created_at   TEXT    NOT NULL
		)`,
		`CREATE TABLE orders (
			id         INTEGER NOT NULL,
			user_id    INTEGER NOT NULL,
			total_usd  REAL    NOT NULL,
			created_at TEXT    NOT NULL
		)`,
		`CREATE TABLE http_requests (
			id         INTEGER NOT NULL,
			url_path   TEXT    NOT NULL,
			status_code INTEGER NOT NULL
		)`,
		// View must be excluded.
		`CREATE VIEW active_users AS SELECT id, email FROM users`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("exec DDL: %v\nstatement: %s", err, stmt)
		}
	}

	tables, err := introspect.Tables(ctx, db, "main", introspect.DialectSQLite)
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}

	// Must return exactly 3 base tables (view excluded).
	if len(tables) != 3 {
		names := make([]string, len(tables))
		for i, tbl := range tables {
			names[i] = tbl.Name
		}
		t.Fatalf("expected 3 tables, got %d: %v", len(tables), names)
	}

	// Run codegen.
	src, err := codegen.Generate(tables, codegen.Config{Package: "schema", DotImport: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	generated := string(src)

	// Verify key identifiers: initialisations (ID, API, HTTP, URL) must apply.
	identChecks := []struct {
		ident string
		desc  string
	}{
		{"UsersID", "id column initialism"},
		{"UsersAPIKey", "api initialism"},
		{"UsersDisplayName", "snake_case split"},
		{"OrdersUserID", "user_id with id initialism"},
		{"OrdersTotalUsd", "total_usd snake_case split"},
		{"HTTPRequests", "http initialism on table name"},
		{"HTTPRequestsURLPath", "http + url initialisations"},
		{"HTTPRequestsStatusCode", "snake_case column"},
	}
	for _, tc := range identChecks {
		if !strings.Contains(generated, tc.ident) {
			t.Errorf("generated source missing %q (%s)\ngenerated:\n%s", tc.ident, tc.desc, generated)
		}
	}

	// View must not appear.
	if strings.Contains(generated, "ActiveUsers") {
		t.Error("generated source must not contain ActiveUsers (view must be excluded)")
	}

	// Compile the generated output.
	dir := t.TempDir()

	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	write(filepath.Join(dir, "gosqstub", "go.mod"), "module github.com/libliflin/gosq\n\ngo 1.22\n")
	write(filepath.Join(dir, "gosqstub", "gosq.go"), `package gosq

type Table struct{}
type Field struct{}

func NewTable(name string) Table { return Table{} }
func NewField(name string) Field { return Field{} }
`)
	write(filepath.Join(dir, "schema", "schema.go"), string(src))
	write(filepath.Join(dir, "go.mod"),
		"module testmod\n\ngo 1.22\n\nrequire github.com/libliflin/gosq v0.0.1\n\nreplace github.com/libliflin/gosq => ./gosqstub\n")

	cmd := exec.Command("go", "build", "./schema")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code does not compile:\n%s\n\ngenerated source:\n%s", out, src)
	}
}
