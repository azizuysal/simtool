package simulator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlistSyntaxHighlighting(t *testing.T) {
	// Test that .plist files get XML syntax highlighting
	lexer := getLexerForExtension(".plist")
	if lexer == nil {
		t.Error("Expected XML lexer for .plist files, got nil")
	}
	
	// Test syntax highlighting for a simple XML line
	line := `<key>CFBundleIdentifier</key>`
	highlighted := GetSyntaxHighlightedLine(line, ".plist")
	
	// Should contain ANSI escape codes if highlighting worked
	if highlighted == line {
		t.Error("Expected syntax highlighting to add ANSI codes")
	}
}

func TestReadBinaryPlist(t *testing.T) {
	// Skip if plutil is not available
	if _, err := exec.LookPath("plutil"); err != nil {
		t.Skip("plutil not found in PATH, skipping binary plist test")
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "plist-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple XML plist first
	xmlPlist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>TestKey</key>
	<string>TestValue</string>
	<key>Number</key>
	<integer>42</integer>
</dict>
</plist>`

	xmlPath := filepath.Join(tmpDir, "test.plist")
	if err := os.WriteFile(xmlPath, []byte(xmlPlist), 0644); err != nil {
		t.Fatal(err)
	}

	// Convert to binary plist
	binaryPath := filepath.Join(tmpDir, "test-binary.plist")
	cmd := exec.Command("plutil", "-convert", "binary1", "-o", binaryPath, xmlPath)
	if err := cmd.Run(); err != nil {
		t.Fatal("Failed to create binary plist:", err)
	}

	// Test reading the binary plist
	lines, totalLines, err := readBinaryPlist(binaryPath, 0, 100)
	if err != nil {
		t.Fatal("Failed to read binary plist:", err)
	}

	// Verify we got some XML content back
	if totalLines == 0 {
		t.Error("No lines returned from binary plist")
	}

	// Check that the content contains expected XML elements
	content := strings.Join(lines, "\n")
	if !strings.Contains(content, "<?xml") {
		t.Error("Output doesn't contain XML declaration")
	}
	if !strings.Contains(content, "TestKey") {
		t.Error("Output doesn't contain expected key")
	}
	if !strings.Contains(content, "TestValue") {
		t.Error("Output doesn't contain expected value")
	}
}

func TestDetectFileTypeWithPlist(t *testing.T) {
	// Skip if plutil is not available
	if _, err := exec.LookPath("plutil"); err != nil {
		t.Skip("plutil not found in PATH, skipping plist detection test")
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "plist-detect-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create XML plist
	xmlPlist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Test</key>
	<string>Value</string>
</dict>
</plist>`

	xmlPath := filepath.Join(tmpDir, "test.plist")
	if err := os.WriteFile(xmlPath, []byte(xmlPlist), 0644); err != nil {
		t.Fatal(err)
	}

	// Test XML plist detection
	fileType := DetectFileType(xmlPath)
	if fileType != FileTypeText {
		t.Errorf("XML plist should be detected as text, got %v", fileType)
	}

	// Create binary plist
	binaryPath := filepath.Join(tmpDir, "test-binary.plist")
	cmd := exec.Command("plutil", "-convert", "binary1", "-o", binaryPath, xmlPath)
	if err := cmd.Run(); err != nil {
		t.Fatal("Failed to create binary plist:", err)
	}

	// Test binary plist detection - should still be text because of .plist extension
	fileType = DetectFileType(binaryPath)
	if fileType != FileTypeText {
		t.Errorf("Binary plist with .plist extension should be detected as text, got %v", fileType)
	}
}

func TestReadFileContentWithBinaryPlist(t *testing.T) {
	// Skip if plutil is not available
	if _, err := exec.LookPath("plutil"); err != nil {
		t.Skip("plutil not found in PATH, skipping plist content test")
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "plist-content-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create XML plist
	xmlPlist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AppName</key>
	<string>TestApp</string>
	<key>Version</key>
	<string>1.0.0</string>
</dict>
</plist>`

	xmlPath := filepath.Join(tmpDir, "Info.plist")
	if err := os.WriteFile(xmlPath, []byte(xmlPlist), 0644); err != nil {
		t.Fatal(err)
	}

	// Create binary plist
	binaryPath := filepath.Join(tmpDir, "Info-binary.plist")
	cmd := exec.Command("plutil", "-convert", "binary1", "-o", binaryPath, xmlPath)
	if err := cmd.Run(); err != nil {
		t.Fatal("Failed to create binary plist:", err)
	}

	// Test reading binary plist content
	content, err := ReadFileContent(binaryPath, 0, 50, 100)
	if err != nil {
		t.Fatal("Failed to read binary plist content:", err)
	}

	// Verify the content
	if content.Type != FileTypeText {
		t.Errorf("Expected FileTypeText, got %v", content.Type)
	}

	if !content.IsBinaryPlist {
		t.Error("IsBinaryPlist flag should be true")
	}

	// Check content contains expected values
	fullContent := strings.Join(content.Lines, "\n")
	if !strings.Contains(fullContent, "AppName") {
		t.Error("Content doesn't contain expected AppName key")
	}
	if !strings.Contains(fullContent, "TestApp") {
		t.Error("Content doesn't contain expected TestApp value")
	}
}