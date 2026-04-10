package components

import (
	"strings"
	"testing"

	"github.com/azizuysal/simtool/internal/simulator"
)

func TestNewDatabaseTableList(t *testing.T) {
	dtl := NewDatabaseTableList(80, 24)
	if dtl.Width != 80 || dtl.Height != 24 {
		t.Errorf("NewDatabaseTableList(80,24): got width=%d height=%d", dtl.Width, dtl.Height)
	}
}

func TestDatabaseTableListGetTitle(t *testing.T) {
	tests := []struct {
		name   string
		dbFile *simulator.FileInfo
		want   string
	}{
		{"no file", nil, "Database Tables"},
		{"with file", &simulator.FileInfo{Name: "app.db"}, "app.db Tables"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtl := NewDatabaseTableList(80, 24)
			dtl.Update(nil, tt.dbFile, 0, 0, nil)
			if got := dtl.GetTitle(); got != tt.want {
				t.Errorf("GetTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDatabaseTableListGetFooter(t *testing.T) {
	info := &simulator.DatabaseInfo{
		Tables: []simulator.TableInfo{
			{Name: "users", RowCount: 3},
		},
	}

	tests := []struct {
		name    string
		info    *simulator.DatabaseInfo
		cursor  int
		wantSub []string
	}{
		{"no data", nil, 0, []string{"up", "down", "back", "quit"}},
		{"table selected", info, 0, []string{"view table", "back", "quit"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtl := NewDatabaseTableList(80, 24)
			dtl.Update(tt.info, nil, tt.cursor, 0, nil)
			got := dtl.GetFooter()
			for _, sub := range tt.wantSub {
				if !strings.Contains(got, sub) {
					t.Errorf("GetFooter() = %q, missing %q", got, sub)
				}
			}
		})
	}
}

func TestDatabaseTableListRender(t *testing.T) {
	dbFile := &simulator.FileInfo{Name: "app.db"}

	tests := []struct {
		name     string
		info     *simulator.DatabaseInfo
		cursor   int
		wantSub  []string
		dontWant []string
	}{
		{
			name:    "no database",
			info:    nil,
			wantSub: []string{"No database loaded"},
		},
		{
			name: "empty database",
			info: &simulator.DatabaseInfo{
				Format:     "SQLite",
				Version:    "3.43.0",
				TableCount: 0,
				FileSize:   1024,
				Tables:     nil,
			},
			wantSub: []string{"SQLite", "3.43.0", "No tables found"},
		},
		{
			name: "two tables, first selected",
			info: &simulator.DatabaseInfo{
				Format:     "SQLite",
				Version:    "3.43.0",
				TableCount: 2,
				FileSize:   4096,
				Tables: []simulator.TableInfo{
					{
						Name:     "users",
						RowCount: 10,
						Columns: []simulator.ColumnInfo{
							{Name: "id", Type: "INTEGER", PK: true},
							{Name: "email", Type: "TEXT"},
						},
					},
					{
						Name:     "posts",
						RowCount: 20,
						Columns:  []simulator.ColumnInfo{{Name: "id"}, {Name: "title"}},
					},
				},
			},
			cursor:  0,
			wantSub: []string{"app.db", "SQLite", "users", "posts", "▶", "Columns"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtl := NewDatabaseTableList(80, 24)
			dtl.Update(tt.info, dbFile, tt.cursor, 0, nil)
			got := dtl.Render()
			for _, sub := range tt.wantSub {
				if !strings.Contains(got, sub) {
					t.Errorf("Render() missing %q\n----\n%s", sub, got)
				}
			}
			for _, sub := range tt.dontWant {
				if strings.Contains(got, sub) {
					t.Errorf("Render() unexpectedly contains %q", sub)
				}
			}
		})
	}
}
