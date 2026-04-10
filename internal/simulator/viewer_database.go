package simulator

import (
	"database/sql"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// ReadDatabaseContent reads information from a database file
func ReadDatabaseContent(path string) (*DatabaseInfo, error) {
	return readDatabaseInfo(path)
}

// openReadOnlyDB opens a SQLite database in read-only mode. The "file:"
// URI prefix is required: without it, go-sqlite3 treats "?mode=ro" as
// part of the filename rather than as a URI parameter, and a missing
// file is silently created as an empty database instead of returning
// an error.
func openReadOnlyDB(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", "file:"+path+"?mode=ro")
}

// readDatabaseInfo reads information from a database file
func readDatabaseInfo(path string) (*DatabaseInfo, error) {
	// Try to open as SQLite database
	db, err := openReadOnlyDB(path)
	if err != nil {
		return &DatabaseInfo{Error: err.Error()}, nil
	}
	defer func() { _ = db.Close() }()

	// Test connection
	if err := db.Ping(); err != nil {
		return &DatabaseInfo{Error: "Not a valid SQLite database: " + err.Error()}, nil
	}

	dbInfo := &DatabaseInfo{
		Format: "SQLite",
	}

	// Get database file size
	if stat, err := os.Stat(path); err == nil {
		dbInfo.FileSize = stat.Size()
	}

	// Get SQLite version
	if version, err := getSQLiteVersion(db); err == nil {
		dbInfo.Version = version
	}

	// Get all tables
	tables, err := getAllTables(db)
	if err != nil {
		dbInfo.Error = err.Error()
		return dbInfo, nil
	}

	dbInfo.Tables = tables
	dbInfo.TableCount = len(tables)

	// Generate schema dump
	if schema, err := generateSchema(db, tables); err == nil {
		dbInfo.Schema = schema
	}

	return dbInfo, nil
}

// getSQLiteVersion gets the SQLite version
func getSQLiteVersion(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow("SELECT sqlite_version()").Scan(&version)
	return version, err
}

// getAllTables gets information about all tables in the database
func getAllTables(db *sql.DB) ([]TableInfo, error) {
	// Query sqlite_master for all tables
	rows, err := db.Query(`
		SELECT name, sql FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tables []TableInfo
	for rows.Next() {
		var tableName, tableSQL string
		if err := rows.Scan(&tableName, &tableSQL); err != nil {
			continue
		}

		table := TableInfo{
			Name:   tableName,
			Schema: tableSQL,
		}

		// Get row count
		if count, err := getTableRowCount(db, tableName); err == nil {
			table.RowCount = count
		}

		// Get column info
		if columns, err := getTableColumns(db, tableName); err == nil {
			table.Columns = columns
		}

		// Get sample data (first 5 rows)
		if sample, err := getTableSample(db, tableName, 5); err == nil {
			table.Sample = sample
		}

		tables = append(tables, table)
	}

	return tables, nil
}

// quoteSQLiteIdentifier returns name safely double-quoted as a SQLite
// identifier. Embedded double quotes are doubled per SQL standard.
// SQLite does not allow parameterizing identifiers, so this is the
// only safe way to interpolate a table name into a query.
func quoteSQLiteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// getTableRowCount gets the number of rows in a table
func getTableRowCount(db *sql.DB, tableName string) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM " + quoteSQLiteIdentifier(tableName)
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

// getTableColumns gets information about table columns
func getTableColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	query := "PRAGMA table_info(" + quoteSQLiteIdentifier(tableName) + ")"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var columns []ColumnInfo
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}

		columns = append(columns, ColumnInfo{
			Name:    name,
			Type:    dataType,
			NotNull: notNull == 1,
			PK:      pk == 1,
		})
	}

	return columns, nil
}

// getTableSample gets sample data from a table
func getTableSample(db *sql.DB, tableName string, limit int) ([]map[string]any, error) {
	query := "SELECT * FROM " + quoteSQLiteIdentifier(tableName) +
		" LIMIT " + strconv.Itoa(limit)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for rows.Next() {
		// Create slice to hold values
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		// Create map from column names to values
		row := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				// Convert byte slices to strings for display
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		result = append(result, row)
	}

	return result, nil
}

// generateSchema generates a schema dump for the database
func generateSchema(db *sql.DB, tables []TableInfo) (string, error) {
	var schema strings.Builder

	schema.WriteString("-- SQLite Database Schema\n\n")

	for _, table := range tables {
		if table.Schema != "" {
			schema.WriteString(table.Schema)
			schema.WriteString(";\n\n")
		}
	}

	return schema.String(), nil
}

// ReadTableData reads paginated data from a specific table
func ReadTableData(dbPath, tableName string, offset, limit int) ([]map[string]any, error) {
	db, err := openReadOnlyDB(dbPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Build query with pagination
	query := "SELECT * FROM " + quoteSQLiteIdentifier(tableName) +
		" LIMIT " + strconv.Itoa(limit) +
		" OFFSET " + strconv.Itoa(offset)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for rows.Next() {
		// Create slice to hold values
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		// Create map from column names to values
		row := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				// Convert byte slices to strings for display
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		result = append(result, row)
	}

	return result, nil
}
