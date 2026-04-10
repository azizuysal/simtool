package file_viewer

import (
	"strings"
	"testing"

	"github.com/azizuysal/simtool/internal/config"
	"github.com/azizuysal/simtool/internal/simulator"
)

// ---------- constructor + basic getters ----------

func TestNewFileViewer(t *testing.T) {
	fv := NewFileViewer(80, 24)
	if fv.Width != 80 || fv.Height != 24 {
		t.Errorf("NewFileViewer(80,24): got width=%d height=%d", fv.Width, fv.Height)
	}
}

func TestFileViewer_GetTitle(t *testing.T) {
	tests := []struct {
		name string
		file *simulator.FileInfo
		want string
	}{
		{"no file", nil, "File Viewer"},
		{"with file", &simulator.FileInfo{Path: "/a/b/readme.txt"}, "readme.txt"},
		{"file at root", &simulator.FileInfo{Path: "config.toml"}, "config.toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fv := NewFileViewer(80, 24)
			fv.Update(tt.file, nil, 0, 0, "", nil)
			if got := fv.GetTitle(); got != tt.want {
				t.Errorf("GetTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileViewer_GetFooter_Defaults(t *testing.T) {
	// Without keys config, GetFooter uses the fallback string.
	fv := NewFileViewer(80, 24)
	fv.Update(&simulator.FileInfo{Path: "/x.txt"}, nil, 0, 0, "", nil)
	got := fv.GetFooter()
	for _, sub := range []string{"scroll up", "scroll down", "back", "quit"} {
		if !strings.Contains(got, sub) {
			t.Errorf("GetFooter() = %q, missing %q", got, sub)
		}
	}
}

func TestFileViewer_GetStatus(t *testing.T) {
	fv := NewFileViewer(80, 24)
	fv.Update(nil, nil, 0, 0, "", nil)
	if got := fv.GetStatus(); got != "" {
		t.Errorf("GetStatus() without warning = %q, want empty", got)
	}

	fv.Update(nil, nil, 0, 0, "SVG rendering limited", nil)
	if got := fv.GetStatus(); !strings.Contains(got, "SVG rendering limited") {
		t.Errorf("GetStatus() = %q, want to contain the warning", got)
	}
}

// ---------- Render dispatcher error paths ----------

func TestFileViewer_Render_NoFile(t *testing.T) {
	fv := NewFileViewer(80, 24)
	fv.Update(nil, nil, 0, 0, "", nil)
	got := fv.Render()
	if !strings.Contains(got, "No file selected") {
		t.Errorf("Render() = %q, want 'No file selected'", got)
	}
}

func TestFileViewer_Render_ContentError(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.txt"}
	content := &simulator.FileContent{Error: errBoom{}}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)
	got := fv.Render()
	if !strings.Contains(got, "Error loading file") {
		t.Errorf("Render() = %q, want 'Error loading file'", got)
	}
}

// errBoom is a minimal error used as a sentinel in tests.
type errBoom struct{}

func (errBoom) Error() string { return "boom" }

// ---------- renderText ----------

func TestRenderText(t *testing.T) {
	file := simulator.FileInfo{Path: "/file.go", Size: 100}
	content := &simulator.FileContent{
		Type:       simulator.FileTypeText,
		Lines:      []string{"package main", "import \"fmt\"", "func main() {}"},
		TotalLines: 3,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	for _, sub := range []string{"Text file", "3 lines", "main", "fmt"} {
		if !strings.Contains(got, sub) {
			t.Errorf("renderText() missing %q\n----\n%s", sub, got)
		}
	}
}

func TestRenderText_PropertyList(t *testing.T) {
	file := simulator.FileInfo{Path: "/Info.plist", Size: 200}
	content := &simulator.FileContent{
		Type:       simulator.FileTypeText,
		Lines:      []string{`<?xml version="1.0"?>`, `<plist version="1.0">`, `</plist>`},
		TotalLines: 3,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	if !strings.Contains(got, "Property list (XML)") {
		t.Errorf("renderText() for .plist should say 'Property list (XML)', got:\n%s", got)
	}
}

func TestRenderText_BinaryPlistConverted(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.plist", Size: 200}
	content := &simulator.FileContent{
		Type:          simulator.FileTypeText,
		Lines:         []string{"<plist>", "</plist>"},
		TotalLines:    2,
		IsBinaryPlist: true,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	if !strings.Contains(got, "Binary plist (converted to XML)") {
		t.Errorf("renderText() should indicate converted binary plist, got:\n%s", got)
	}
}

func TestRenderText_LineNumbers(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.txt", Size: 50}
	content := &simulator.FileContent{
		Type:       simulator.FileTypeText,
		Lines:      []string{"line one", "line two"},
		TotalLines: 2,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	// Line numbers "1" and "2" should appear alongside the content.
	// Use the column separator as a stable marker.
	if !strings.Contains(got, "│") {
		t.Errorf("renderText() should draw line-number column separator\n%s", got)
	}
}

func TestRenderText_ViewportOffset(t *testing.T) {
	// ContentOffset should be added to the displayed line numbers so
	// that a file read in chunks shows the correct absolute numbering.
	file := simulator.FileInfo{Path: "/x.txt", Size: 1000}
	content := &simulator.FileContent{
		Type:       simulator.FileTypeText,
		Lines:      []string{"chunk line", "another"},
		TotalLines: 1000,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 500, "", nil)

	got := fv.Render()
	// The first visible line should have absolute number 501.
	if !strings.Contains(got, "501") {
		t.Errorf("renderText() at offset 500 should show line 501, got:\n%s", got)
	}
}

// ---------- renderBinary ----------

func TestRenderBinary(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.bin", Size: 16}
	content := &simulator.FileContent{
		Type:         simulator.FileTypeBinary,
		BinaryData:   []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, // "Hello"
		BinaryOffset: 0,
		TotalSize:    16,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	for _, sub := range []string{"Binary file", "00000000", "48 65 6c 6c 6f", "Hello"} {
		if !strings.Contains(got, sub) {
			t.Errorf("renderBinary() missing %q\n----\n%s", sub, got)
		}
	}
}

func TestRenderBinary_NilData(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.bin", Size: 0}
	content := &simulator.FileContent{
		Type:       simulator.FileTypeBinary,
		BinaryData: nil,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	// Just the header, no hex body.
	if !strings.Contains(got, "Binary file") {
		t.Errorf("renderBinary() missing header, got:\n%s", got)
	}
}

// ---------- renderImage ----------

func TestRenderImage_NoImageInfo(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.png", Size: 100}
	content := &simulator.FileContent{Type: simulator.FileTypeImage, ImageInfo: nil}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	if !strings.Contains(got, "Error loading image") {
		t.Errorf("renderImage() with nil ImageInfo = %q, want error", got)
	}
}

func TestRenderImage_WithMetadata(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.png", Size: 2048}
	content := &simulator.FileContent{
		Type: simulator.FileTypeImage,
		ImageInfo: &simulator.ImageInfo{
			Format: "PNG",
			Width:  128,
			Height: 64,
			Size:   2048,
			Preview: &simulator.ImagePreview{
				Width:  10,
				Height: 5,
				Rows:   []string{"row0", "row1", "row2", "row3", "row4"},
			},
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	for _, sub := range []string{"Image file", "PNG", "128", "64", "row0", "row4"} {
		if !strings.Contains(got, sub) {
			t.Errorf("renderImage() missing %q\n----\n%s", sub, got)
		}
	}
}

func TestRenderImage_NoPreview(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.png", Size: 100}
	content := &simulator.FileContent{
		Type: simulator.FileTypeImage,
		ImageInfo: &simulator.ImageInfo{
			Format: "PNG",
			Width:  32,
			Height: 32,
			Size:   100,
			// No Preview
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	// Should still render the metadata header.
	if !strings.Contains(got, "Image file") {
		t.Errorf("renderImage() missing header:\n%s", got)
	}
}

// ---------- renderArchive ----------

func TestRenderArchive_NoArchiveInfo(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.zip", Size: 0}
	content := &simulator.FileContent{Type: simulator.FileTypeArchive, ArchiveInfo: nil}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	if !strings.Contains(got, "Error loading archive") {
		t.Errorf("renderArchive() with nil ArchiveInfo = %q, want error", got)
	}
}

func TestRenderArchive_Tree(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.zip", Size: 1024}
	content := &simulator.FileContent{
		Type: simulator.FileTypeArchive,
		ArchiveInfo: &simulator.ArchiveInfo{
			Format:         "ZIP",
			FileCount:      3,
			FolderCount:    1,
			TotalSize:      300,
			CompressedSize: 150,
			Entries: []simulator.ArchiveEntry{
				{Name: "README.md", Size: 100},
				{Name: "src/main.go", Size: 100},
				{Name: "src/util.go", Size: 100},
				{Name: "src/", IsDir: true},
			},
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	// Tree box characters and file/folder names should appear.
	for _, sub := range []string{"ZIP Archive", "README.md", "src", "main.go", "util.go", "├──", "└──"} {
		if !strings.Contains(got, sub) {
			t.Errorf("renderArchive() missing %q\n----\n%s", sub, got)
		}
	}
}

func TestBuildTreeFromPaths(t *testing.T) {
	entries := []simulator.ArchiveEntry{
		{Name: "a.txt"},
		{Name: "dir/b.txt"},
		{Name: "dir/c.txt"},
		{Name: "dir/sub/d.txt"},
	}
	tree := buildTreeFromPaths(entries)

	if len(tree.children) != 2 {
		t.Errorf("root has %d children, want 2 (a.txt, dir)", len(tree.children))
	}
	if _, ok := tree.children["a.txt"]; !ok {
		t.Error("missing a.txt child")
	}
	dir, ok := tree.children["dir"]
	if !ok {
		t.Fatal("missing dir child")
	}
	if !dir.isDir {
		t.Error("dir should be flagged isDir")
	}
	if len(dir.children) != 3 {
		t.Errorf("dir has %d children, want 3 (b.txt, c.txt, sub)", len(dir.children))
	}
	sub, ok := dir.children["sub"]
	if !ok {
		t.Fatal("missing dir/sub child")
	}
	if _, ok := sub.children["d.txt"]; !ok {
		t.Error("missing dir/sub/d.txt leaf")
	}
}

// ---------- renderDatabase ----------

func TestRenderDatabase_NoDatabaseInfo(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.db", Size: 100}
	content := &simulator.FileContent{Type: simulator.FileTypeDatabase, DatabaseInfo: nil}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	if !strings.Contains(got, "Error loading database") {
		t.Errorf("renderDatabase() with nil DatabaseInfo = %q, want error", got)
	}
}

func TestRenderDatabase_WithTables(t *testing.T) {
	file := simulator.FileInfo{Path: "/app.db", Size: 4096}
	content := &simulator.FileContent{
		Type: simulator.FileTypeDatabase,
		DatabaseInfo: &simulator.DatabaseInfo{
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
						{Name: "name", Type: "TEXT", NotNull: true},
					},
					Sample: []map[string]any{
						{"id": int64(1), "name": "alice"},
					},
					Schema: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
				},
				{
					Name:     "posts",
					RowCount: 5,
					Columns:  []simulator.ColumnInfo{{Name: "id"}, {Name: "title"}},
				},
			},
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	for _, sub := range []string{"Database file", "SQLite", "3.43.0", "users", "posts", "Columns:", "Sample data"} {
		if !strings.Contains(got, sub) {
			t.Errorf("renderDatabase() missing %q\n----\n%s", sub, got)
		}
	}
}

func TestRenderDatabase_EmptyTables(t *testing.T) {
	file := simulator.FileInfo{Path: "/empty.db"}
	content := &simulator.FileContent{
		Type: simulator.FileTypeDatabase,
		DatabaseInfo: &simulator.DatabaseInfo{
			Format:     "SQLite",
			TableCount: 0,
			Tables:     nil,
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	if !strings.Contains(got, "No tables found") {
		t.Errorf("renderDatabase() empty = %q, want 'No tables found'", got)
	}
}

func TestRenderDatabase_ErrorInfo(t *testing.T) {
	file := simulator.FileInfo{Path: "/bad.db"}
	content := &simulator.FileContent{
		Type: simulator.FileTypeDatabase,
		DatabaseInfo: &simulator.DatabaseInfo{
			Format: "SQLite",
			Error:  "corrupt file",
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	if !strings.Contains(got, "corrupt file") {
		t.Errorf("renderDatabase() should surface DatabaseInfo.Error, got:\n%s", got)
	}
}

// ---------- Render dispatcher: unknown type ----------

func TestFileViewer_Render_UnknownType(t *testing.T) {
	file := simulator.FileInfo{Path: "/x"}
	// Use an out-of-range FileType value to exercise the default branch.
	content := &simulator.FileContent{Type: simulator.FileType(99)}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.Render()
	if !strings.Contains(got, "Unknown file type") {
		t.Errorf("Render() = %q, want 'Unknown file type'", got)
	}
}

// ---------- GetFooter with configured keys ----------

func TestFileViewer_GetFooter_ConfiguredKeys(t *testing.T) {
	keys := config.DefaultKeys()
	file := simulator.FileInfo{Path: "/x.txt"}
	content := &simulator.FileContent{
		Type:       simulator.FileTypeText,
		Lines:      []string{"a"},
		TotalLines: 1,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", &keys)

	got := fv.GetFooter()
	for _, sub := range []string{"scroll up", "scroll down", "back", "quit"} {
		if !strings.Contains(got, sub) {
			t.Errorf("GetFooter() = %q, missing %q", got, sub)
		}
	}
}

// ---------- scroll info: each file type ----------

func TestFileViewer_ScrollInfo_Text(t *testing.T) {
	// Multiple lines + mid-viewport → footer should contain range with both arrows.
	file := simulator.FileInfo{Path: "/x.txt"}
	content := &simulator.FileContent{
		Type:       simulator.FileTypeText,
		Lines:      make([]string, 20),
		TotalLines: 100,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 5, 50, "", nil)

	got := fv.GetFooter()
	if !strings.Contains(got, "of 100") {
		t.Errorf("GetFooter() scroll info = %q, want to mention 100 total", got)
	}
}

func TestFileViewer_ScrollInfo_Image(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.png"}
	content := &simulator.FileContent{
		Type: simulator.FileTypeImage,
		ImageInfo: &simulator.ImageInfo{
			Preview: &simulator.ImagePreview{Rows: make([]string, 40)},
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 2, 0, "", nil)

	got := fv.GetFooter()
	// renderImageContent produces info + separator + blank + N preview
	// rows, so total lines = 3 + len(Rows) = 43.
	if !strings.Contains(got, "of 43") {
		t.Errorf("GetFooter() scroll info = %q, want to include 'of 43'", got)
	}
}

func TestFileViewer_ScrollInfo_Binary(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.bin"}
	// 160 bytes → 10 hex lines. Total size 800 → 50 total lines.
	content := &simulator.FileContent{
		Type:         simulator.FileTypeBinary,
		BinaryData:   make([]byte, 160),
		BinaryOffset: 0,
		TotalSize:    800,
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.GetFooter()
	// 800 / 16 = 50 total hex lines
	if !strings.Contains(got, "of 50") {
		t.Errorf("GetFooter() scroll info = %q, want to include 'of 50'", got)
	}
}

func TestFileViewer_ScrollInfo_Archive(t *testing.T) {
	// Archive with 3 entries nested under a shared prefix. The tree
	// has 4 unique nodes: "dir", "dir/a.txt", "dir/b.txt", "dir/c.txt".
	file := simulator.FileInfo{Path: "/x.zip"}
	content := &simulator.FileContent{
		Type: simulator.FileTypeArchive,
		ArchiveInfo: &simulator.ArchiveInfo{
			Entries: []simulator.ArchiveEntry{
				{Name: "dir/a.txt"},
				{Name: "dir/b.txt"},
				{Name: "dir/c.txt"},
			},
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.GetFooter()
	if !strings.Contains(got, "of 4") {
		t.Errorf("GetFooter() scroll info = %q, want to include 'of 4'", got)
	}
}

func TestFileViewer_ScrollInfo_Database(t *testing.T) {
	// Two tables, each with 2 columns and 3 sample rows plus a schema.
	// Per table (not counting inter-table spacing): 1 header + 1
	// columns + 1 "Sample data:" + 3 sample + 2 schema = 8 lines.
	// Two tables = 16 lines, plus 2 lines spacing between them = 18.
	file := simulator.FileInfo{Path: "/x.db"}
	tbl := simulator.TableInfo{
		Name:    "users",
		Columns: []simulator.ColumnInfo{{Name: "id"}, {Name: "name"}},
		Sample:  []map[string]any{{}, {}, {}},
		Schema:  "CREATE TABLE users (id INTEGER, name TEXT)",
	}
	content := &simulator.FileContent{
		Type: simulator.FileTypeDatabase,
		DatabaseInfo: &simulator.DatabaseInfo{
			Tables: []simulator.TableInfo{tbl, tbl},
		},
	}
	fv := NewFileViewer(80, 24)
	fv.Update(&file, content, 0, 0, "", nil)

	got := fv.GetFooter()
	if !strings.Contains(got, "of 18") {
		t.Errorf("GetFooter() scroll info = %q, want to include 'of 18'", got)
	}
}

// ---------- count helpers ----------

func TestCountArchiveTreeLines(t *testing.T) {
	tests := []struct {
		name    string
		entries []simulator.ArchiveEntry
		want    int
	}{
		{"nil info", nil, 0},
		{"flat list", []simulator.ArchiveEntry{
			{Name: "a.txt"},
			{Name: "b.txt"},
			{Name: "c.txt"},
		}, 3},
		{"shared prefix", []simulator.ArchiveEntry{
			{Name: "dir/a.txt"},
			{Name: "dir/b.txt"},
		}, 3}, // "dir", "dir/a.txt", "dir/b.txt"
		{"deeply nested", []simulator.ArchiveEntry{
			{Name: "a/b/c/d.txt"},
		}, 4}, // "a", "a/b", "a/b/c", "a/b/c/d.txt"
		{"duplicate paths coalesce", []simulator.ArchiveEntry{
			{Name: "dir/x.txt"},
			{Name: "dir/x.txt"},
		}, 2}, // "dir", "dir/x.txt"
		{"trailing and leading slashes ignored", []simulator.ArchiveEntry{
			{Name: "/dir/x.txt"},
			{Name: "dir/y.txt/"},
		}, 3}, // "dir", "dir/x.txt", "dir/y.txt" — empty parts are skipped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var info *simulator.ArchiveInfo
			if tt.entries != nil {
				info = &simulator.ArchiveInfo{Entries: tt.entries}
			}
			if got := countArchiveTreeLines(info); got != tt.want {
				t.Errorf("countArchiveTreeLines() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountDatabaseLines(t *testing.T) {
	tests := []struct {
		name string
		info *simulator.DatabaseInfo
		want int
	}{
		{"nil info", nil, 0},
		{"empty tables", &simulator.DatabaseInfo{}, 0},
		{
			name: "single bare table (header only)",
			info: &simulator.DatabaseInfo{
				Tables: []simulator.TableInfo{{Name: "t"}},
			},
			want: 1,
		},
		{
			name: "single table with columns + sample + schema",
			info: &simulator.DatabaseInfo{
				Tables: []simulator.TableInfo{{
					Name:    "users",
					Columns: []simulator.ColumnInfo{{Name: "id"}},
					Sample:  []map[string]any{{}, {}},
					Schema:  "CREATE TABLE users (id INTEGER)",
				}},
			},
			// 1 header + 1 cols + 1 sample header + 2 sample rows + 2 schema = 7
			want: 7,
		},
		{
			name: "two tables with spacing",
			info: &simulator.DatabaseInfo{
				Tables: []simulator.TableInfo{
					{Name: "a"}, // 1 header
					{Name: "b"}, // 1 header + 2 spacing before
				},
			},
			want: 1 + 2 + 1, // 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countDatabaseLines(tt.info); got != tt.want {
				t.Errorf("countDatabaseLines() = %d, want %d", got, tt.want)
			}
		})
	}
}
