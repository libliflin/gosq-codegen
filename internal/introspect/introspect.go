// Package introspect reads database schema metadata (tables, columns, types)
// from a live database connection using information_schema queries.
package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// Dialect identifies the SQL dialect used for parameterized queries.
type Dialect string

const (
	// DialectPostgres uses $1 positional placeholders (PostgreSQL, CockroachDB).
	DialectPostgres Dialect = "postgres"
	// DialectMySQL uses ? positional placeholders (MySQL, MariaDB).
	DialectMySQL Dialect = "mysql"
	// DialectSQLite introspects via sqlite_master and PRAGMA table_info.
	// The schema parameter is used as the Table.Schema field only; SQLite
	// does not support named schemas in the PostgreSQL/MySQL sense.
	DialectSQLite Dialect = "sqlite"
	// DialectSQLServer uses @p1 positional placeholders (Microsoft SQL Server).
	DialectSQLServer Dialect = "sqlserver"
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
// d selects the SQL dialect: DialectPostgres uses $1 placeholders,
// DialectMySQL uses ?, and DialectSQLite uses sqlite_master + PRAGMA table_info.
func Tables(ctx context.Context, db *sql.DB, schema string, d Dialect) ([]Table, error) {
	if d == DialectSQLite {
		return tablesFromSQLite(ctx, db, schema)
	}

	placeholder := "$1"
	switch d {
	case DialectMySQL:
		placeholder = "?"
	case DialectSQLServer:
		placeholder = "@p1"
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

// tablesFromSQLite introspects an SQLite database using sqlite_master and
// PRAGMA table_info. Views are excluded (sqlite_master type = 'view').
// The schema parameter is stored in Table.Schema but does not filter results —
// SQLite databases contain one schema (the attached database file).
func tablesFromSQLite(ctx context.Context, db *sql.DB, schema string) ([]Table, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name FROM sqlite_master
WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query sqlite_master: %w", err)
	}

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		tableNames = append(tableNames, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate table names: %w", err)
	}
	rows.Close()

	tables := make([]Table, 0, len(tableNames))
	for _, name := range tableNames {
		cols, err := sqliteColumnsForTable(ctx, db, name)
		if err != nil {
			return nil, err
		}
		tables = append(tables, Table{Schema: schema, Name: name, Columns: cols})
	}
	return tables, nil
}

// sqliteColumnsForTable returns columns for one SQLite table via PRAGMA table_info.
// Results are ordered by cid (column definition order). The table name is
// double-quoted to handle names with special characters.
func sqliteColumnsForTable(ctx context.Context, db *sql.DB, table string) ([]Column, error) {
	// Double-quote the table name; escape embedded double quotes by doubling.
	quoted := `"` + strings.ReplaceAll(table, `"`, `""`) + `"`
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+quoted+")")
	if err != nil {
		return nil, fmt.Errorf("pragma table_info(%s): %w", table, err)
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var cid, notNull, pk int
		var colName, dataType string
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &colName, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("scan column for table %s: %w", table, err)
		}
		if dataType == "" {
			dataType = "text"
		}
		cols = append(cols, Column{
			Name:       colName,
			DataType:   dataType,
			IsNullable: notNull == 0,
			OrdinalPos: cid + 1,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate columns for %s: %w", table, err)
	}
	return cols, nil
}
