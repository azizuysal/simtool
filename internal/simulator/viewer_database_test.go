package simulator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// createTestDB creates a SQLite database file at path and populates it
// with a known schema and data. Any statement failure is fatal for the
// test.
func createTestDB(t *testing.T, path string, statements ...string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		t.Fatalf("ping test db: %v", err)
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
}

func TestQuoteSQLiteIdentifier(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "users", `"users"`},
		{"with_underscore", "user_profiles", `"user_profiles"`},
		{"empty", "", `""`},
		{"single_quote_in_name", "O'Brien", `"O'Brien"`},
		{"single_double_quote", `foo"bar`, `"foo""bar"`},
		{"multiple_double_quotes", `a"b"c`, `"a""b""c"`},
		{"only_double_quotes", `"""`, `""""""""`},
		{"with_space", "my table", `"my table"`},
		{"unicode", "café", `"café"`},
		{"sql_keyword", "SELECT", `"SELECT"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteSQLiteIdentifier(tt.in)
			if got != tt.want {
				t.Errorf("quoteSQLiteIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestReadDatabaseInfo_ValidDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	createTestDB(t, dbPath,
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)`,
		`INSERT INTO users (name, email) VALUES ('alice', 'alice@example.com')`,
		`INSERT INTO users (name, email) VALUES ('bob', 'bob@example.com')`,
		`CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT, body TEXT)`,
		`INSERT INTO posts (title, body) VALUES ('hello', 'world')`,
	)

	info, err := readDatabaseInfo(dbPath)
	if err != nil {
		t.Fatalf("readDatabaseInfo: %v", err)
	}
	if info.Error != "" {
		t.Fatalf("info.Error = %q, expected empty", info.Error)
	}

	if info.Format != "SQLite" {
		t.Errorf("Format = %q, want SQLite", info.Format)
	}
	if info.Version == "" {
		t.Error("Version is empty")
	}
	if info.FileSize == 0 {
		t.Error("FileSize is 0")
	}
	if info.TableCount != 2 {
		t.Errorf("TableCount = %d, want 2", info.TableCount)
	}
	if len(info.Tables) != 2 {
		t.Fatalf("len(Tables) = %d, want 2", len(info.Tables))
	}

	// Tables are sorted alphabetically: posts, users
	if info.Tables[0].Name != "posts" {
		t.Errorf("Tables[0].Name = %q, want posts", info.Tables[0].Name)
	}
	if info.Tables[1].Name != "users" {
		t.Errorf("Tables[1].Name = %q, want users", info.Tables[1].Name)
	}

	users := info.Tables[1]
	if users.RowCount != 2 {
		t.Errorf("users.RowCount = %d, want 2", users.RowCount)
	}
	if len(users.Columns) != 3 {
		t.Errorf("users.Columns count = %d, want 3", len(users.Columns))
	}
	// First column: id INTEGER PRIMARY KEY
	if users.Columns[0].Name != "id" || !users.Columns[0].PK {
		t.Errorf("users.Columns[0] = %+v, want name=id PK=true", users.Columns[0])
	}
	// Second column: name TEXT NOT NULL
	if users.Columns[1].Name != "name" || !users.Columns[1].NotNull {
		t.Errorf("users.Columns[1] = %+v, want name=name NotNull=true", users.Columns[1])
	}

	if !strings.Contains(info.Schema, "CREATE TABLE users") {
		t.Error("Schema does not contain users CREATE TABLE")
	}
	if !strings.Contains(info.Schema, "CREATE TABLE posts") {
		t.Error("Schema does not contain posts CREATE TABLE")
	}
}

func TestReadDatabaseInfo_MissingFile(t *testing.T) {
	// Regression test: previously readDatabaseInfo silently succeeded
	// on a missing file because go-sqlite3 didn't honor ?mode=ro
	// without the file: URI prefix. After the fix, a missing file
	// reports an error in DatabaseInfo.Error.
	info, err := readDatabaseInfo(filepath.Join(t.TempDir(), "does-not-exist.db"))
	if err != nil {
		t.Fatalf("readDatabaseInfo returned err: %v", err)
	}
	if info.Error == "" {
		t.Error("info.Error is empty for missing file, want non-empty")
	}
}

func TestReadDatabaseInfo_NotSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garbage.db")
	if err := os.WriteFile(path, []byte("this is not a sqlite database"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	info, err := readDatabaseInfo(path)
	if err != nil {
		t.Fatalf("readDatabaseInfo returned error: %v", err)
	}
	if info.Error == "" {
		t.Error("info.Error is empty for non-sqlite file")
	}
}

func TestReadDatabaseContent_DelegatesToReadDatabaseInfo(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "delegate.db")
	createTestDB(t, dbPath, `CREATE TABLE t (x INTEGER)`)

	info, err := ReadDatabaseContent(dbPath)
	if err != nil {
		t.Fatalf("ReadDatabaseContent: %v", err)
	}
	if info.TableCount != 1 {
		t.Errorf("TableCount = %d, want 1", info.TableCount)
	}
}

func TestReadTableData_Pagination(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pagination.db")
	createTestDB(t, dbPath,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, label TEXT)`,
	)

	// Insert 10 rows
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for i := 1; i <= 10; i++ {
		if _, err := db.Exec(`INSERT INTO items (label) VALUES (?)`, fmt.Sprintf("row-%d", i)); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	_ = db.Close()

	// First page: offset 0, limit 3
	page1, err := ReadTableData(dbPath, "items", 0, 3)
	if err != nil {
		t.Fatalf("ReadTableData page1: %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("page1 len = %d, want 3", len(page1))
	}
	if page1[0]["label"] != "row-1" {
		t.Errorf("page1[0].label = %v, want row-1", page1[0]["label"])
	}

	// Second page: offset 3, limit 3
	page2, err := ReadTableData(dbPath, "items", 3, 3)
	if err != nil {
		t.Fatalf("ReadTableData page2: %v", err)
	}
	if len(page2) != 3 {
		t.Errorf("page2 len = %d, want 3", len(page2))
	}
	if page2[0]["label"] != "row-4" {
		t.Errorf("page2[0].label = %v, want row-4", page2[0]["label"])
	}

	// Last page: offset 9, limit 3 → only 1 row
	lastPage, err := ReadTableData(dbPath, "items", 9, 3)
	if err != nil {
		t.Fatalf("ReadTableData lastPage: %v", err)
	}
	if len(lastPage) != 1 {
		t.Errorf("lastPage len = %d, want 1", len(lastPage))
	}
}

func TestReadTableData_InvalidDBPath(t *testing.T) {
	// Regression test paired with TestReadDatabaseInfo_MissingFile:
	// missing file should surface as an error via openReadOnlyDB's
	// file: URI prefix, not as a "no such table" error from a
	// silently-created empty database.
	_, err := ReadTableData(filepath.Join(t.TempDir(), "nope.db"), "t", 0, 10)
	if err == nil {
		t.Error("expected error for missing DB file, got nil")
	}
}

// TestTableNameWithDoubleQuote validates the quoteSQLiteIdentifier
// security fix. SQLite allows `"` inside a quoted identifier by
// doubling it. If the code used naive interpolation, this table
// name would break out of the identifier and cause a syntax error
// (or worse, SQL injection on a writable connection). Here we verify
// that readDatabaseInfo, getTableRowCount, getTableSample, and
// ReadTableData all handle such a table name correctly end-to-end.
func TestTableNameWithDoubleQuote(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mischief.db")
	// Table name is literally: weird"name
	// In SQL: CREATE TABLE "weird""name" (x INTEGER)
	createTestDB(t, dbPath,
		`CREATE TABLE "weird""name" (x INTEGER, y TEXT)`,
		`INSERT INTO "weird""name" VALUES (1, 'one')`,
		`INSERT INTO "weird""name" VALUES (2, 'two')`,
	)

	info, err := readDatabaseInfo(dbPath)
	if err != nil {
		t.Fatalf("readDatabaseInfo: %v", err)
	}
	if info.Error != "" {
		t.Fatalf("info.Error = %q", info.Error)
	}
	if len(info.Tables) != 1 {
		t.Fatalf("len(Tables) = %d, want 1", len(info.Tables))
	}
	table := info.Tables[0]
	if table.Name != `weird"name` {
		t.Errorf("Table.Name = %q, want %q", table.Name, `weird"name`)
	}
	if table.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", table.RowCount)
	}
	if len(table.Columns) != 2 {
		t.Errorf("len(Columns) = %d, want 2", len(table.Columns))
	}
	if len(table.Sample) != 2 {
		t.Errorf("len(Sample) = %d, want 2", len(table.Sample))
	}

	rows, err := ReadTableData(dbPath, `weird"name`, 0, 10)
	if err != nil {
		t.Fatalf("ReadTableData: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("rows len = %d, want 2", len(rows))
	}
	if rows[0]["y"] != "one" {
		t.Errorf("rows[0].y = %v, want one", rows[0]["y"])
	}
}

func TestGenerateSchema(t *testing.T) {
	tables := []TableInfo{
		{Name: "users", Schema: "CREATE TABLE users (id INTEGER PRIMARY KEY)"},
		{Name: "posts", Schema: "CREATE TABLE posts (id INTEGER PRIMARY KEY)"},
		{Name: "empty", Schema: ""}, // empty schema should be skipped
	}

	got, err := generateSchema(tables)
	if err != nil {
		t.Fatalf("generateSchema: %v", err)
	}
	if !strings.Contains(got, "CREATE TABLE users") {
		t.Error("schema missing users")
	}
	if !strings.Contains(got, "CREATE TABLE posts") {
		t.Error("schema missing posts")
	}
	if strings.Contains(got, "empty") {
		t.Error("schema unexpectedly contains empty-schema table")
	}
	if !strings.HasPrefix(got, "-- SQLite Database Schema") {
		t.Error("schema missing header comment")
	}
}

func TestGetSQLiteVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "version.db")
	createTestDB(t, dbPath, `CREATE TABLE t (x INTEGER)`)

	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	version, err := getSQLiteVersion(db)
	if err != nil {
		t.Fatalf("getSQLiteVersion: %v", err)
	}
	if version == "" {
		t.Error("version is empty")
	}
	// Version format is e.g. "3.43.2" — should start with a digit
	if version[0] < '0' || version[0] > '9' {
		t.Errorf("version %q does not look like a semver", version)
	}
}
