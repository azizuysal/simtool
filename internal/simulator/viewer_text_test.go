package simulator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTextFile_FullFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.txt")
	body := "line one\nline two\nline three\n"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	lines, total, isBinPlist, err := readTextFile(path, 0, 100)
	if err != nil {
		t.Fatalf("readTextFile: %v", err)
	}
	if isBinPlist {
		t.Error("isBinaryPlist = true, want false for plain .txt")
	}
	if total != 3 {
		t.Errorf("totalLines = %d, want 3", total)
	}
	if len(lines) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(lines))
	}
	want := []string{"line one", "line two", "line three"}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], w)
		}
	}
}

func TestReadTextFile_Pagination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "many.txt")
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString("line\n")
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// Skip first 10 lines, take 5 — expect 5 returned, total=50.
	lines, total, _, err := readTextFile(path, 10, 5)
	if err != nil {
		t.Fatalf("readTextFile: %v", err)
	}
	if total != 50 {
		t.Errorf("totalLines = %d, want 50", total)
	}
	if len(lines) != 5 {
		t.Errorf("len(lines) = %d, want 5", len(lines))
	}
}

func TestReadTextFile_LongLineTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "long.txt")
	// 3000 'a' characters — longer than maxDisplayLineLength (2000).
	long := strings.Repeat("a", 3000)
	if err := os.WriteFile(path, []byte(long+"\nshort\n"), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	lines, total, _, err := readTextFile(path, 0, 10)
	if err != nil {
		t.Fatalf("readTextFile: %v", err)
	}
	if total != 2 {
		t.Errorf("totalLines = %d, want 2", total)
	}
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}
	// First line truncated at maxDisplayLineLength (2000) with "..." suffix.
	if !strings.HasSuffix(lines[0], "...") {
		t.Errorf("lines[0] not truncated; suffix = %q", lines[0][len(lines[0])-min(6, len(lines[0])):])
	}
	if len(lines[0]) != maxDisplayLineLength+len("...") {
		t.Errorf("len(lines[0]) = %d, want %d", len(lines[0]), maxDisplayLineLength+len("..."))
	}
	if lines[1] != "short" {
		t.Errorf("lines[1] = %q, want 'short'", lines[1])
	}
}

func TestReadTextFile_NonexistentFile(t *testing.T) {
	_, _, _, err := readTextFile(filepath.Join(t.TempDir(), "missing.txt"), 0, 10)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReadTextFile_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	lines, total, _, err := readTextFile(path, 0, 10)
	if err != nil {
		t.Fatalf("readTextFile: %v", err)
	}
	if total != 0 {
		t.Errorf("totalLines = %d, want 0", total)
	}
	if len(lines) != 0 {
		t.Errorf("len(lines) = %d, want 0", len(lines))
	}
}

func TestReadTextFile_PlistXMLReadsAsText(t *testing.T) {
	// A .plist file whose magic bytes are NOT "bplist" should fall
	// through to the plain text path (isBinaryPlist stays false).
	path := filepath.Join(t.TempDir(), "xml.plist")
	body := `<?xml version="1.0" encoding="UTF-8"?>` + "\n<plist></plist>\n"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	lines, total, isBinPlist, err := readTextFile(path, 0, 10)
	if err != nil {
		t.Fatalf("readTextFile: %v", err)
	}
	if isBinPlist {
		t.Error("isBinaryPlist = true, want false for XML plist")
	}
	if total != 2 {
		t.Errorf("totalLines = %d, want 2", total)
	}
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "<?xml") {
		t.Errorf("unexpected lines: %+v", lines)
	}
}

func TestReadTextFile_BinaryPlistSniffed(t *testing.T) {
	// A .plist file whose first six bytes are "bplist" must be routed
	// through the binary-plist path (isBinaryPlist = true). Build a
	// real binary plist with plutil so the conversion actually succeeds
	// and we can assert on the XML output.
	if _, err := exec.LookPath("plutil"); err != nil {
		t.Skip("plutil not found; skipping binary-plist sniff test")
	}

	tmp := t.TempDir()
	xmlPath := filepath.Join(tmp, "seed.plist")
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict><key>MarkerKey</key><string>MarkerValue</string></dict>
</plist>`
	if err := os.WriteFile(xmlPath, []byte(xml), 0600); err != nil {
		t.Fatalf("write xml seed: %v", err)
	}

	bplistPath := filepath.Join(tmp, "real.plist")
	if err := exec.Command("plutil", "-convert", "binary1", "-o", bplistPath, xmlPath).Run(); err != nil {
		t.Fatalf("plutil convert to binary1: %v", err)
	}

	// Sanity check: ensure the produced file really starts with "bplist"
	// so we're exercising the sniffing branch.
	head, err := os.ReadFile(bplistPath)
	if err != nil {
		t.Fatalf("read bplist: %v", err)
	}
	if len(head) < 6 || string(head[:6]) != "bplist" {
		t.Fatalf("fixture is not a binary plist; header=%q", string(head[:min(6, len(head))]))
	}

	lines, total, isBinPlist, err := readTextFile(bplistPath, 0, 100)
	if err != nil {
		t.Fatalf("readTextFile: %v", err)
	}
	if !isBinPlist {
		t.Error("isBinaryPlist = false, want true")
	}
	if total == 0 {
		t.Fatal("totalLines = 0, want > 0")
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "MarkerKey") || !strings.Contains(joined, "MarkerValue") {
		t.Errorf("converted XML missing expected markers; got %q", joined)
	}
}
