package simulator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetFilesForContainer(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	
	// Create some test files and directories
	testDir := filepath.Join(tmpDir, "TestDir")
	os.Mkdir(testDir, 0755)
	
	testFile1 := filepath.Join(tmpDir, "file1.txt")
	os.WriteFile(testFile1, []byte("test content"), 0644)
	
	testFile2 := filepath.Join(tmpDir, "file2.json")
	os.WriteFile(testFile2, []byte("{}"), 0644)
	
	// Create a file in subdirectory
	testFile3 := filepath.Join(testDir, "file3.txt")
	os.WriteFile(testFile3, []byte("nested file"), 0644)
	
	tests := []struct {
		name          string
		containerPath string
		wantErr       bool
		checkResults  func(t *testing.T, files []FileInfo)
	}{
		{
			name:          "valid directory",
			containerPath: tmpDir,
			wantErr:       false,
			checkResults: func(t *testing.T, files []FileInfo) {
				if len(files) != 3 { // TestDir, file1.txt, file2.json
					t.Errorf("Expected 3 files, got %d", len(files))
				}
				
				// Check that directories come first
				if !files[0].IsDirectory {
					t.Error("Expected first item to be a directory")
				}
				
				// Check file names
				foundDir := false
				foundFile1 := false
				foundFile2 := false
				for _, f := range files {
					switch f.Name {
					case "TestDir":
						foundDir = true
						if !f.IsDirectory {
							t.Error("TestDir should be a directory")
						}
					case "file1.txt":
						foundFile1 = true
						if f.IsDirectory {
							t.Error("file1.txt should not be a directory")
						}
						if f.Size == 0 {
							t.Error("file1.txt should have non-zero size")
						}
					case "file2.json":
						foundFile2 = true
					}
				}
				
				if !foundDir || !foundFile1 || !foundFile2 {
					t.Error("Not all expected files found")
				}
			},
		},
		{
			name:          "with file:// prefix",
			containerPath: "file://" + tmpDir,
			wantErr:       false,
			checkResults: func(t *testing.T, files []FileInfo) {
				if len(files) == 0 {
					t.Error("Should handle file:// prefix")
				}
			},
		},
		{
			name:          "non-existent directory",
			containerPath: "/non/existent/path",
			wantErr:       true,
		},
		{
			name:          "empty directory",
			containerPath: testDir,
			wantErr:       false,
			checkResults: func(t *testing.T, files []FileInfo) {
				if len(files) != 1 { // Only file3.txt
					t.Errorf("Expected 1 file in TestDir, got %d", len(files))
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := GetFilesForContainer(tt.containerPath)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFilesForContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.checkResults != nil && !tt.wantErr {
				tt.checkResults(t, files)
			}
		})
	}
}

func TestCalculateDirSize(t *testing.T) {
	// Create a temporary directory with known file sizes
	tmpDir := t.TempDir()
	
	// Create files with specific sizes
	file1 := filepath.Join(tmpDir, "file1.txt")
	os.WriteFile(file1, make([]byte, 1024), 0644) // 1KB
	
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	
	file2 := filepath.Join(subDir, "file2.txt")
	os.WriteFile(file2, make([]byte, 2048), 0644) // 2KB
	
	size := calculateDirSize(tmpDir)
	expectedSize := int64(3072) // 1KB + 2KB
	
	if size != expectedSize {
		t.Errorf("calculateDirSize() = %d, want %d", size, expectedSize)
	}
	
	// Test non-existent directory
	size = calculateDirSize("/non/existent/path")
	if size != 0 {
		t.Errorf("calculateDirSize() for non-existent path = %d, want 0", size)
	}
}

func TestFormatFileDateEdgeCases(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		time     time.Time
		validate func(string) bool
	}{
		{
			name: "exactly 7 days ago",
			time: now.AddDate(0, 0, -7),
			validate: func(s string) bool {
				// Should show full date, not day of week
				return s != "" && !strings.Contains(s, ":")
			},
		},
		{
			name: "future date",
			time: now.AddDate(0, 0, 1),
			validate: func(s string) bool {
				// Future dates should still format correctly
				return s != ""
			},
		},
		{
			name: "different year in past",
			time: time.Date(2019, 12, 31, 23, 59, 59, 0, time.UTC),
			validate: func(s string) bool {
				return s == "Dec 31, 2019"
			},
		},
		{
			name: "within last week",
			time: now.AddDate(0, 0, -3),
			validate: func(s string) bool {
				// Could show either day with time OR month day
				return s != "" && len(s) > 3
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFileDate(tt.time)
			if !tt.validate(result) {
				t.Errorf("FormatFileDate() = %v, validation failed", result)
			}
		})
	}
}