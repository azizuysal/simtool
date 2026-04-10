package components

import (
	"strings"
	"testing"
	"time"

	"github.com/azizuysal/simtool/internal/simulator"
)

func TestNewFileList(t *testing.T) {
	fl := NewFileList(80, 24)
	if fl.Width != 80 || fl.Height != 24 {
		t.Errorf("NewFileList(80,24): got width=%d height=%d", fl.Width, fl.Height)
	}
}

func TestFileListGetTitle(t *testing.T) {
	tests := []struct {
		name string
		app  *simulator.App
		want string
	}{
		{"no app", nil, "Files"},
		{"with app", &simulator.App{Name: "MyApp"}, "MyApp Files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl := NewFileList(80, 24)
			fl.Update(nil, 0, 0, tt.app, nil, nil)
			if got := fl.GetTitle(); got != tt.want {
				t.Errorf("GetTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileListGetFooter(t *testing.T) {
	app := &simulator.App{Name: "MyApp"}
	files := []simulator.FileInfo{
		{Name: "dir", IsDirectory: true},
		{Name: "file.txt", IsDirectory: false},
	}

	tests := []struct {
		name    string
		files   []simulator.FileInfo
		cursor  int
		wantSub []string // substrings expected in footer
	}{
		{
			name:    "empty list",
			files:   nil,
			cursor:  0,
			wantSub: []string{"up", "down", "back", "quit"},
		},
		{
			name:    "directory selected",
			files:   files,
			cursor:  0,
			wantSub: []string{"enter", "open in Finder", "back"},
		},
		{
			name:    "file selected",
			files:   files,
			cursor:  1,
			wantSub: []string{"view file", "open in Finder", "back"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl := NewFileList(80, 24)
			fl.Update(tt.files, tt.cursor, 0, app, nil, nil)
			got := fl.GetFooter()
			for _, sub := range tt.wantSub {
				if !strings.Contains(got, sub) {
					t.Errorf("GetFooter() = %q, missing %q", got, sub)
				}
			}
		})
	}
}

func TestFileListRender(t *testing.T) {
	now := time.Now()
	app := &simulator.App{
		Name:     "MyApp",
		BundleID: "com.example.myapp",
		Version:  "1.0",
		Size:     1024,
	}

	tests := []struct {
		name        string
		files       []simulator.FileInfo
		cursor      int
		breadcrumbs []string
		wantSub     []string
		dontWant    []string
	}{
		{
			name:     "empty folder",
			files:    nil,
			wantSub:  []string{"MyApp", "com.example.myapp", "No files in folder"},
			dontWant: []string{"▶"},
		},
		{
			name: "single file selected",
			files: []simulator.FileInfo{
				{Name: "readme.txt", IsDirectory: false, Size: 42, CreatedAt: now, ModifiedAt: now},
			},
			cursor:  0,
			wantSub: []string{"MyApp", "readme.txt", "▶"},
		},
		{
			name: "directory with breadcrumbs",
			files: []simulator.FileInfo{
				{Name: "sub", IsDirectory: true, CreatedAt: now, ModifiedAt: now},
			},
			cursor:      0,
			breadcrumbs: []string{"Documents", "Inner"},
			wantSub:     []string{"Documents/Inner/", "sub"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl := NewFileList(80, 24)
			fl.Update(tt.files, tt.cursor, 0, app, tt.breadcrumbs, nil)
			got := fl.Render()
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

func TestFileListRenderWithoutApp(t *testing.T) {
	// When App is nil, buildHeader returns "" and renderHeaderPrefix
	// should emit nothing (no header, no separator).
	fl := NewFileList(80, 24)
	fl.Update(nil, 0, 0, nil, nil, nil)
	got := fl.Render()
	// No app info and no files -> "No files in folder" shown, no header rule
	if !strings.Contains(got, "No files in folder") {
		t.Errorf("Render() missing empty-state message: %q", got)
	}
	if strings.Contains(got, "─") {
		t.Errorf("Render() unexpectedly contains separator rule without a header")
	}
}
