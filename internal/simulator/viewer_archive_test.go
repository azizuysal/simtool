package simulator

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTestZip writes a zip with the given entries to path and returns the path.
// An entry whose Name ends with "/" is written as a directory entry.
type zipEntry struct {
	Name    string
	Content string
	Mod     time.Time
}

func writeTestZip(t *testing.T, path string, entries []zipEntry) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			t.Fatalf("close zip: %v", cerr)
		}
	}()

	zw := zip.NewWriter(f)
	for _, e := range entries {
		header := &zip.FileHeader{
			Name:     e.Name,
			Method:   zip.Deflate,
			Modified: e.Mod,
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatalf("zip CreateHeader %q: %v", e.Name, err)
		}
		if _, err := w.Write([]byte(e.Content)); err != nil {
			t.Fatalf("zip write %q: %v", e.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
}

func TestReadArchiveInfo_FilesAndDirs(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "test.zip")
	mod := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	writeTestZip(t, zipPath, []zipEntry{
		{Name: "dir1/", Mod: mod},
		{Name: "dir1/file1.txt", Content: "hello world", Mod: mod},
		{Name: "dir1/file2.txt", Content: "second", Mod: mod},
		{Name: "root.txt", Content: "r", Mod: mod},
	})

	info, err := readArchiveInfo(zipPath)
	if err != nil {
		t.Fatalf("readArchiveInfo: %v", err)
	}
	if info.Format != "ZIP" {
		t.Errorf("Format = %q, want ZIP", info.Format)
	}
	if info.FileCount != 3 {
		t.Errorf("FileCount = %d, want 3", info.FileCount)
	}
	if info.FolderCount != 1 {
		t.Errorf("FolderCount = %d, want 1", info.FolderCount)
	}
	wantTotal := int64(len("hello world") + len("second") + len("r"))
	if info.TotalSize != wantTotal {
		t.Errorf("TotalSize = %d, want %d", info.TotalSize, wantTotal)
	}
	if info.CompressedSize <= 0 {
		t.Errorf("CompressedSize = %d, want > 0", info.CompressedSize)
	}
	if len(info.Entries) != 4 {
		t.Fatalf("len(Entries) = %d, want 4", len(info.Entries))
	}

	// Verify entry fields for the directory and one file.
	var gotDir, gotFile *ArchiveEntry
	for i := range info.Entries {
		e := &info.Entries[i]
		switch e.Name {
		case "dir1/":
			gotDir = e
		case "dir1/file1.txt":
			gotFile = e
		}
	}
	if gotDir == nil {
		t.Fatal("directory entry dir1/ missing")
	}
	if !gotDir.IsDir {
		t.Errorf("dir1/ IsDir = false, want true")
	}
	if gotFile == nil {
		t.Fatal("file entry dir1/file1.txt missing")
	}
	if gotFile.IsDir {
		t.Error("file1.txt IsDir = true, want false")
	}
	if gotFile.Size != int64(len("hello world")) {
		t.Errorf("file1.txt Size = %d, want %d", gotFile.Size, len("hello world"))
	}
	if gotFile.CompressedSize <= 0 {
		t.Errorf("file1.txt CompressedSize = %d, want > 0", gotFile.CompressedSize)
	}
	if !gotFile.ModTime.Equal(mod) {
		t.Errorf("file1.txt ModTime = %v, want %v", gotFile.ModTime, mod)
	}
}

func TestReadArchiveInfo_EmptyZip(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "empty.zip")
	writeTestZip(t, zipPath, nil)

	info, err := readArchiveInfo(zipPath)
	if err != nil {
		t.Fatalf("readArchiveInfo: %v", err)
	}
	if info.FileCount != 0 || info.FolderCount != 0 {
		t.Errorf("FileCount=%d FolderCount=%d, want 0/0", info.FileCount, info.FolderCount)
	}
	if len(info.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(info.Entries))
	}
	if info.TotalSize != 0 || info.CompressedSize != 0 {
		t.Errorf("sizes non-zero: TotalSize=%d CompressedSize=%d", info.TotalSize, info.CompressedSize)
	}
}

func TestReadArchiveInfo_NotAZipFile(t *testing.T) {
	notZip := filepath.Join(t.TempDir(), "plain.txt")
	if err := os.WriteFile(notZip, bytes.Repeat([]byte("x"), 128), 0600); err != nil {
		t.Fatalf("write non-zip: %v", err)
	}

	if _, err := readArchiveInfo(notZip); err == nil {
		t.Fatal("readArchiveInfo on non-zip returned nil error, want error")
	}
}

func TestReadArchiveInfo_NonexistentFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.zip")
	if _, err := readArchiveInfo(missing); err == nil {
		t.Fatal("readArchiveInfo on missing file returned nil error, want error")
	}
}
