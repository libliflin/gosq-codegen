package introspect

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
)

// driverSeq provides unique driver names across test runs; sql.Register panics on duplicates.
var driverSeq atomic.Int64

// newMockDB registers a fresh mock driver and returns a *sql.DB whose queries
// return the provided rows in order.
// Each row must be [table_name, column_name, data_type, is_nullable, ordinal_position]
// with ordinal_position as int64.
func newMockDB(t *testing.T, rows [][]driver.Value) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("introspect_mock_%d", driverSeq.Add(1))
	sql.Register(name, &mockDriver{rows: rows})
	db, err := sql.Open(name, "mock")
	if err != nil {
		t.Fatalf("sql.Open mock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- minimal mock driver (stdlib only) ---

type mockDriver struct{ rows [][]driver.Value }
type mockConn struct{ rows [][]driver.Value }
type mockStmt struct{ rows [][]driver.Value }
type mockRows struct {
	data [][]driver.Value
	pos  int
}

func (d *mockDriver) Open(_ string) (driver.Conn, error) {
	return &mockConn{rows: d.rows}, nil
}

func (c *mockConn) Prepare(_ string) (driver.Stmt, error) {
	return &mockStmt{rows: c.rows}, nil
}
func (c *mockConn) Close() error              { return nil }
func (c *mockConn) Begin() (driver.Tx, error) { return nil, fmt.Errorf("not supported") }

func (s *mockStmt) Close() error                                 { return nil }
func (s *mockStmt) NumInput() int                                { return -1 }
func (s *mockStmt) Exec(_ []driver.Value) (driver.Result, error) { return nil, fmt.Errorf("not supported") }
func (s *mockStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &mockRows{data: s.rows}, nil
}

func (r *mockRows) Columns() []string {
	return []string{"table_name", "column_name", "data_type", "is_nullable", "ordinal_position"}
}
func (r *mockRows) Close() error { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

// TestTablesEmpty verifies that Tables returns an empty slice (not an error) when
// the schema contains no base tables.
func TestTablesEmpty(t *testing.T) {
	db := newMockDB(t, nil)
	tables, err := Tables(context.Background(), db, "public")
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(tables))
	}
}

// TestTablesNullableParsing verifies that is_nullable="YES" maps to IsNullable=true
// and is_nullable="NO" maps to IsNullable=false.
func TestTablesNullableParsing(t *testing.T) {
	rows := [][]driver.Value{
		{"users", "id", "integer", "NO", int64(1)},
		{"users", "bio", "text", "YES", int64(2)},
	}
	db := newMockDB(t, rows)
	tables, err := Tables(context.Background(), db, "public")
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	cols := tables[0].Columns
	if cols[0].IsNullable {
		t.Error(`id: is_nullable="NO" should produce IsNullable=false`)
	}
	if !cols[1].IsNullable {
		t.Error(`bio: is_nullable="YES" should produce IsNullable=true`)
	}
}

// TestTablesSorting verifies that tables are returned in alphabetical order
// regardless of the order in which rows arrive from the database.
func TestTablesSorting(t *testing.T) {
	// Rows arrive zebra-first; Tables must sort them.
	rows := [][]driver.Value{
		{"zebra", "id", "integer", "NO", int64(1)},
		{"alpha", "id", "integer", "NO", int64(1)},
		{"middle", "id", "integer", "NO", int64(1)},
	}
	db := newMockDB(t, rows)
	tables, err := Tables(context.Background(), db, "public")
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if len(tables) != 3 {
		t.Fatalf("expected 3 tables, got %d", len(tables))
	}
	want := []string{"alpha", "middle", "zebra"}
	for i, w := range want {
		if tables[i].Name != w {
			t.Errorf("tables[%d].Name = %q, want %q", i, tables[i].Name, w)
		}
	}
}

// TestTablesSchemaAndOrdinal verifies that the Schema field is populated from the
// schema argument and that OrdinalPos is correctly scanned from the query result.
func TestTablesSchemaAndOrdinal(t *testing.T) {
	rows := [][]driver.Value{
		{"orders", "id", "integer", "NO", int64(1)},
		{"orders", "amount", "numeric", "NO", int64(2)},
		{"orders", "note", "text", "YES", int64(3)},
	}
	const schema = "myschema"
	db := newMockDB(t, rows)
	tables, err := Tables(context.Background(), db, schema)
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	tbl := tables[0]
	if tbl.Schema != schema {
		t.Errorf("Schema = %q, want %q", tbl.Schema, schema)
	}
	if len(tbl.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(tbl.Columns))
	}
	for i, col := range tbl.Columns {
		if col.OrdinalPos != i+1 {
			t.Errorf("Columns[%d].OrdinalPos = %d, want %d", i, col.OrdinalPos, i+1)
		}
	}
}
