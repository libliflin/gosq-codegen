// Package introspect reads database schema metadata (tables, columns, types)
// from a live database connection using information_schema queries.
package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

// Dialect identifies the SQL dialect used for parameterized queries.
type Dialect string

const (
	// DialectPostgres uses $1 positional placeholders (PostgreSQL, CockroachDB).
	DialectPostgres Dialect = "postgres"
	// DialectMySQL uses ? positional placeholders (MySQL, MariaDB).
	DialectMySQL Dialect = "mysql"
)

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

// Tables queries information_schema.columns and returns all base tables in the
// given schema, sorted by table name with columns ordered by ordinal position.
// d selects the SQL placeholder style: DialectPostgres uses $1, DialectMySQL uses ?.
func Tables(ctx context.Context, db *sql.DB, schema string, d Dialect) ([]Table, error) {
	placeholder := "$1"
	if d == DialectMySQL {
		placeholder = "?"
	}
	q := `
SELECT c.table_name, c.column_name, c.data_type, c.is_nullable, c.ordinal_position
FROM information_schema.columns c
JOIN information_schema.tables t
  ON t.table_schema = c.table_schema AND t.table_name = c.table_name
WHERE c.table_schema = ` + placeholder + `
  AND t.table_type = 'BASE TABLE'
ORDER BY c.table_name, c.ordinal_position`

	rows, err := db.QueryContext(ctx, q, schema)
	if err != nil {
		return nil, fmt.Errorf("query information_schema: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*Table)
	var tableOrder []string

	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		var ordinalPos int
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &ordinalPos); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		if _, ok := tableMap[tableName]; !ok {
			tableMap[tableName] = &Table{Schema: schema, Name: tableName}
			tableOrder = append(tableOrder, tableName)
		}

		tableMap[tableName].Columns = append(tableMap[tableName].Columns, Column{
			Name:       columnName,
			DataType:   dataType,
			IsNullable: isNullable == "YES",
			OrdinalPos: ordinalPos,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	sort.Strings(tableOrder)
	tables := make([]Table, 0, len(tableOrder))
	for _, name := range tableOrder {
		tables = append(tables, *tableMap[name])
	}
	return tables, nil
}
