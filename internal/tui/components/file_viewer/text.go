package file_viewer

import (
	"fmt"
	"path/filepath"
	"strings"

	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// renderText renders text file content with syntax highlighting
func (fv *FileViewer) renderText() string {
	var s strings.Builder
	innerWidth := fv.Width - 4 // Account for content box padding

	// File info header
	fileType := "Text file"
	if fv.Content.IsBinaryPlist {
		fileType = "Binary plist (converted to XML)"
	} else if strings.HasSuffix(strings.ToLower(fv.File.Path), ".plist") {
		fileType = "Property list (XML)"
	}
	
	info := fmt.Sprintf("%s • %d lines • %s",
		fileType,
		fv.Content.TotalLines,
		simulator.FormatSize(fv.File.Size))
	s.WriteString(ui.DetailStyle().Render(info))
	s.WriteString("\n")
	s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
	s.WriteString("\n\n")

	// Calculate visible lines
	// Don't subtract border height as we're already in content dimensions
	headerLines := 4 // Info + separator + padding
	visibleLines := fv.Height - headerLines
	
	startLine := fv.ContentViewport
	endLine := startLine + visibleLines
	if endLine > len(fv.Content.Lines) {
		endLine = len(fv.Content.Lines)
	}

	// Render lines with syntax highlighting
	maxLineNumWidth := len(fmt.Sprintf("%d", fv.ContentOffset+endLine))
	fileExt := filepath.Ext(fv.File.Path)

	lineCount := 0
	for i := startLine; i < endLine && i < len(fv.Content.Lines); i++ {
		if lineCount > 0 {
			s.WriteString("\n")
		}
		
		// Line number
		lineNum := fv.ContentOffset + i + 1
		lineNumStr := fmt.Sprintf("%*d", maxLineNumWidth, lineNum)
		s.WriteString(ui.DetailStyle().Render(lineNumStr + " │ "))

		// Line content with syntax highlighting
		line := fv.Content.Lines[i]
		maxLineWidth := innerWidth - maxLineNumWidth - 4
		if len(line) > maxLineWidth {
			line = line[:maxLineWidth-3] + "..."
		}
		
		highlightedLine := simulator.GetSyntaxHighlightedLine(line, fileExt)
		s.WriteString(highlightedLine)
		lineCount++
	}

	// Don't pad - ContentBox will handle filling the space

	return s.String()
}