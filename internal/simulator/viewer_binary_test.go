package simulator

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReadBinaryFile_FromStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.bin")
	payload := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	if err := os.WriteFile(path, payload, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := readBinaryFile(path, 0, 8)
	if err != nil {
		t.Fatalf("readBinaryFile: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("got = % x, want % x", got, payload)
	}
}

func TestReadBinaryFile_WithOffset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.bin")
	payload := []byte("ABCDEFGHIJ")
	if err := os.WriteFile(path, payload, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := readBinaryFile(path, 4, 3)
	if err != nil {
		t.Fatalf("readBinaryFile: %v", err)
	}
	if string(got) != "EFG" {
		t.Errorf("got = %q, want %q", string(got), "EFG")
	}
}

func TestReadBinaryFile_TruncatesAtEOF(t *testing.T) {
	// Request more bytes than remain past the offset — should return
	// only what was actually read (no zero-padding, no error).
	path := filepath.Join(t.TempDir(), "data.bin")
	payload := []byte("hello")
	if err := os.WriteFile(path, payload, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := readBinaryFile(path, 2, 100)
	if err != nil {
		t.Fatalf("readBinaryFile: %v", err)
	}
	if string(got) != "llo" {
		t.Errorf("got = %q, want %q", string(got), "llo")
	}
}

func TestReadBinaryFile_OffsetPastEOF(t *testing.T) {
	// Seeking past the end of a regular file succeeds, but the subsequent
	// read returns 0 bytes and io.EOF. The helper should swallow the EOF
	// and return an empty slice.
	path := filepath.Join(t.TempDir(), "small.bin")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := readBinaryFile(path, 100, 16)
	if err != nil {
		t.Fatalf("readBinaryFile: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

func TestReadBinaryFile_NonexistentPath(t *testing.T) {
	_, err := readBinaryFile(filepath.Join(t.TempDir(), "missing"), 0, 8)
	if err == nil {
		t.Fatal("readBinaryFile on missing file returned nil error, want error")
	}
}

func TestReadBinaryFile_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatalf("write empty: %v", err)
	}

	got, err := readBinaryFile(path, 0, 16)
	if err != nil {
		t.Fatalf("readBinaryFile: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}
