package components

import (
	"strings"
	"testing"

	"github.com/azizuysal/simtool/internal/simulator"
)

func TestNewDatabaseTableContent(t *testing.T) {
	dtc := NewDatabaseTableContent(80, 24)
	if dtc.Width != 80 || dtc.Height != 24 {
		t.Errorf("NewDatabaseTableContent(80,24): got width=%d height=%d", dtc.Width, dtc.Height)
	}
}

func TestDatabaseTableContentGetTitle(t *testing.T) {
	tests := []struct {
		name  string
		table *simulator.TableInfo
		want  string
	}{
		{"no table", nil, "Table Content"},
		{"with table", &simulator.TableInfo{Name: "users"}, "Table: users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtc := NewDatabaseTableContent(80, 24)
			dtc.Update(tt.table, nil, nil, 0, 0, nil)
			if got := dtc.GetTitle(); got != tt.want {
				t.Errorf("GetTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDatabaseTableContentGetFooter(t *testing.T) {
	table := &simulator.TableInfo{Name: "users", RowCount: 100}
	data := []map[string]any{
		{"id": 1, "name": "alice"},
		{"id": 2, "name": "bob"},
		{"id": 3, "name": "carol"},
	}

	dtc := NewDatabaseTableContent(80, 24)
	dtc.Update(table, data, nil, 0, 0, nil)
	got := dtc.GetFooter()

	for _, sub := range []string{"scroll up", "scroll down", "back", "quit", "(1-3 of 100)"} {
		if !strings.Contains(got, sub) {
			t.Errorf("GetFooter() = %q, missing %q", got, sub)
		}
	}
}

func TestDatabaseTableContentRender(t *testing.T) {
	tests := []struct {
		name    string
		table   *simulator.TableInfo
		data    []map[string]any
		wantSub []string
	}{
		{
			name:    "no table selected",
			table:   nil,
			wantSub: []string{"No table selected"},
		},
		{
			name: "empty data",
			table: &simulator.TableInfo{
				Name:     "users",
				RowCount: 0,
				Columns: []simulator.ColumnInfo{
					{Name: "id", Type: "INTEGER", PK: true},
					{Name: "name", Type: "TEXT"},
				},
			},
			data:    nil,
			wantSub: []string{"users", "id", "name", "No data"},
		},
		{
			name: "with rows",
			table: &simulator.TableInfo{
				Name:     "users",
				RowCount: 2,
				Columns: []simulator.ColumnInfo{
					{Name: "id", Type: "INTEGER", PK: true},
					{Name: "name", Type: "TEXT"},
				},
			},
			data: []map[string]any{
				{"id": int64(1), "name": "alice"},
				{"id": int64(2), "name": "bob"},
			},
			wantSub: []string{"users", "id*", "name", "alice", "bob"},
		},
		{
			name: "binary data is sanitized",
			table: &simulator.TableInfo{
				Name:     "blobs",
				RowCount: 1,
				Columns:  []simulator.ColumnInfo{{Name: "data"}},
			},
			data: []map[string]any{
				{"data": "hello\x00world"}, // embedded null byte
			},
			wantSub: []string{"blobs", "□"}, // null byte replaced with box character
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtc := NewDatabaseTableContent(80, 24)
			dtc.Update(tt.table, tt.data, nil, 0, 0, nil)
			got := dtc.Render()
			for _, sub := range tt.wantSub {
				if !strings.Contains(got, sub) {
					t.Errorf("Render() missing %q\n----\n%s", sub, got)
				}
			}
		})
	}
}

func TestSanitizeForDisplay(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii", "hello world", "hello world"},
		{"newline replaced", "line1\nline2", "line1 line2"},
		{"carriage return replaced", "line1\rline2", "line1 line2"},
		{"null byte -> box", "a\x00b", "a□b"},
		{"tab control char -> box", "a\tb", "a□b"},
		{"unicode preserved", "café", "café"},
		{"trimmed", "  hi  ", "hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeForDisplay(tt.in); got != tt.want {
				t.Errorf("sanitizeForDisplay(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
