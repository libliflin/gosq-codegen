//go:build integration

// Integration tests for introspect.Tables. These tests require a real
// PostgreSQL instance. Set TEST_DSN to run them locally:
//
//	docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=test postgres:16
//	TEST_DSN="postgres://postgres:test@localhost:5432/postgres?sslmode=disable" \
//	  go test -tags integration ./...
//
// In CI, the Postgres service is provided via GitHub Actions services: postgres:
// and TEST_DSN is set automatically.
package introspect_test

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/libliflin/gosq-codegen/internal/codegen"
	"github.com/libliflin/gosq-codegen/internal/introspect"
	_ "github.com/lib/pq"
)

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set; skipping integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("ping db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestTablesEcommerce loads the ecommerce DDL fixture into a temporary Postgres
// schema and verifies that Tables returns the correct tables and columns.
func TestTablesEcommerce(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := context.Background()

	const schema = "gosq_integration_test"

	// Clean up any previous run, then create fresh schema.
	if _, err := db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
	})

	// Load the DDL fixture via a dedicated connection so SET search_path
	// applies to the same connection that executes the DDL.
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("get conn: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SET search_path TO "+schema); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	ddl, err := os.ReadFile("../../testdata/schemas/ecommerce.sql")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if _, err := conn.ExecContext(ctx, string(ddl)); err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	tables, err := introspect.Tables(ctx, db, schema)
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}

	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}

	// Tables are sorted alphabetically: orders, users.
	if tables[0].Name != "orders" {
		t.Errorf("tables[0].Name = %q, want %q", tables[0].Name, "orders")
	}
	if tables[1].Name != "users" {
		t.Errorf("tables[1].Name = %q, want %q", tables[1].Name, "users")
	}

	// Verify schema field is set correctly on each table.
	for _, tbl := range tables {
		if tbl.Schema != schema {
			t.Errorf("table %q: Schema = %q, want %q", tbl.Name, tbl.Schema, schema)
		}
	}

	// Verify orders columns (id, user_id, total, created_at — all NOT NULL).
	orders := tables[0]
	if len(orders.Columns) != 4 {
		t.Fatalf("orders: expected 4 columns, got %d", len(orders.Columns))
	}
	wantOrders := []struct {
		name     string
		nullable bool
	}{
		{"id", false},
		{"user_id", false},
		{"total", false},
		{"created_at", false},
	}
	for i, wc := range wantOrders {
		col := orders.Columns[i]
		if col.Name != wc.name {
			t.Errorf("orders.Columns[%d].Name = %q, want %q", i, col.Name, wc.name)
		}
		if col.IsNullable != wc.nullable {
			t.Errorf("orders.Columns[%d].IsNullable = %v, want %v", i, col.IsNullable, wc.nullable)
		}
		if col.OrdinalPos != i+1 {
			t.Errorf("orders.Columns[%d].OrdinalPos = %d, want %d", i, col.OrdinalPos, i+1)
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
			t.Errorf("users.Columns[%d].IsNullable = %v, want %v", i, col.IsNullable, wc.nullable)
		}
		if col.OrdinalPos != i+1 {
			t.Errorf("users.Columns[%d].OrdinalPos = %d, want %d", i, col.OrdinalPos, i+1)
		}
	}
}

// TestPipelineEcommerce runs the full pipeline end-to-end:
// DDL fixture → introspect.Tables (real Postgres) → codegen.Generate → go build.
// This verifies that the tool's core promise holds: point it at a database,
// get compilable Go out the other end.
func TestPipelineEcommerce(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := context.Background()

	const schema = "gosq_pipeline_test"

	if _, err := db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
	})

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("get conn: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SET search_path TO "+schema); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	ddl, err := os.ReadFile("../../testdata/schemas/ecommerce.sql")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if _, err := conn.ExecContext(ctx, string(ddl)); err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	tables, err := introspect.Tables(ctx, db, schema)
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if len(tables) == 0 {
		t.Fatal("Tables returned no tables")
	}

	src, err := codegen.Generate(tables, codegen.Config{Package: "schema", DotImport: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Write the generated output into a temporary module with a minimal gosq
	// stub and verify it compiles.
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
