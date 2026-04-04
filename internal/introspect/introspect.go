// Package introspect reads database schema metadata (tables, columns, types)
// from a live database connection using information_schema queries.
package introspect

import (
	"database/sql"
	"fmt"
	"sort"

	_ "github.com/lib/pq"
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

// Tables queries information_schema.columns and returns all tables in the given
// schema, sorted by table name with columns ordered by ordinal position.
func Tables(db *sql.DB, schema string) ([]Table, error) {
	const q = `
SELECT c.table_name, c.column_name, c.data_type, c.is_nullable, c.ordinal_position
FROM information_schema.columns c
JOIN information_schema.tables t
  ON t.table_schema = c.table_schema AND t.table_name = c.table_name
WHERE c.table_schema = $1
  AND t.table_type = 'BASE TABLE'
ORDER BY c.table_name, c.ordinal_position`

	rows, err := db.Query(q, schema)
	if err != nil {
		return nil, fmt.Errorf("introspect: query information_schema: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*Table)
	var tableOrder []string

	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		var ordinalPos int
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &ordinalPos); err != nil {
			return nil, fmt.Errorf("introspect: scan row: %w", err)
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
		return nil, fmt.Errorf("introspect: iterate rows: %w", err)
	}

	sort.Strings(tableOrder)
	tables := make([]Table, 0, len(tableOrder))
	for _, name := range tableOrder {
		tables = append(tables, *tableMap[name])
	}
	return tables, nil
}
