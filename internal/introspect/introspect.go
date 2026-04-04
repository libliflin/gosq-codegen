// Package introspect reads database schema metadata (tables, columns, types)
// from a live database connection using information_schema queries.
package introspect

// Table represents a database table and its columns.
type Table struct {
	Schema  string
	Name    string
	Columns []Column
}

// Column represents a single column in a table.
type Column struct {
	Name       string
	DataType   string
	IsNullable bool
	OrdinalPos int
}
