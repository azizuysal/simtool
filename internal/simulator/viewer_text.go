package simulator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// textScanBufferSize caps the maximum line length the bufio scanner
	// will accept (1 MiB). Minified sources can legitimately have very
	// long lines that would otherwise cause a "token too long" error.
	textScanBufferSize = 1024 * 1024
	// maxDisplayLineLength truncates displayed lines to keep rendering
	// responsive on pathologically long lines (e.g. minified JS).
	maxDisplayLineLength = 2000
)

// readTextFile reads a text file with pagination support
// Returns lines, totalLines, isBinaryPlist, error
func readTextFile(path string, startLine, maxLines int) ([]string, int, bool, error) {
	isBinaryPlist := false

	// Check if it's a plist file
	if strings.HasSuffix(strings.ToLower(path), ".plist") {
		// Check if it's a binary plist by reading first few bytes
		file, err := os.Open(path)
		if err != nil {
			return nil, 0, false, err
		}

		magic := make([]byte, 6)
		n, err := file.Read(magic)
		_ = file.Close()

		if err == nil && n >= 6 && string(magic) == "bplist" {
			// It's a binary plist, convert it to XML for viewing
			lines, total, err := readBinaryPlist(path, startLine, maxLines)
			return lines, total, true, err
		}
		// Otherwise, fall through to read as normal text file
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, false, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	var lines []string
	currentLine := 0
	totalLines := 0

	// Set a larger buffer size for very long lines (common in minified files)
	buf := make([]byte, textScanBufferSize)
	scanner.Buffer(buf, textScanBufferSize)

	for scanner.Scan() {
		totalLines++
		if currentLine >= startLine && len(lines) < maxLines {
			line := scanner.Text()
			// Truncate very long lines for display
			if len(line) > maxDisplayLineLength {
				line = line[:maxDisplayLineLength] + "..."
			}
			lines = append(lines, line)
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return lines, totalLines, isBinaryPlist, err
	}

	return lines, totalLines, isBinaryPlist, nil
}

// readBinaryPlist converts a binary plist to XML and reads it
func readBinaryPlist(path string, startLine, maxLines int) ([]string, int, error) {
	// Use plutil to convert binary plist to XML
	cmd := exec.Command("plutil", "-convert", "xml1", "-o", "-", path)
	output, err := cmd.Output()
	if err != nil {
		// If conversion fails, return an error message
		return []string{fmt.Sprintf("Error converting binary plist: %v", err)}, 1, nil
	}

	// Split the XML output into lines
	allLines := strings.Split(string(output), "\n")
	totalLines := len(allLines)

	// Handle empty output
	if totalLines == 0 {
		return []string{}, 0, nil
	}

	// Apply pagination
	endLine := startLine + maxLines
	if endLine > totalLines {
		endLine = totalLines
	}

	lines := make([]string, 0, endLine-startLine)
	for i := startLine; i < endLine; i++ {
		line := allLines[i]
		// Truncate very long lines for display
		if len(line) > maxDisplayLineLength {
			line = line[:maxDisplayLineLength] + "..."
		}
		lines = append(lines, line)
	}

	return lines, totalLines, nil
}
