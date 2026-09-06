package file_viewer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/azizuysal/simtool/internal/simulator"
	"github.com/azizuysal/simtool/internal/ui"
)

// renderText renders text file content with syntax highlighting
func (fv *FileViewer) renderText() string {
	var s strings.Builder
	innerWidth := fv.Width - 6 // Account for the outer border and padding
	if innerWidth < 0 {
		innerWidth = 0
	}

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

	// ContentBox reserves two rows, and the rendered file header uses three.
	headerLines := 5
	visibleLines := fv.Height - headerLines
	if visibleLines < 1 {
		visibleLines = 1
	}

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

		// Line content is truncated by terminal cell width so wide Unicode
		// characters cannot wrap the outer layout.
		line := fv.Content.Lines[i]
		maxLineWidth := innerWidth - maxLineNumWidth - 4
		line = truncateTextLine(line, maxLineWidth)

		// Use detected language if available
		highlightedLine := ""
		if fv.Content.DetectedLang != "" {
			highlightedLine = simulator.GetSyntaxHighlightedLineWithLang(line, fileExt, fv.Content.DetectedLang)
		} else {
			highlightedLine = simulator.GetSyntaxHighlightedLine(line, fileExt)
		}
		s.WriteString(highlightedLine)
		lineCount++
	}

	// Don't pad - ContentBox will handle filling the space

	return s.String()
}

func truncateTextLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	ellipsis := "..."
	if width <= len(ellipsis) {
		ellipsis = ""
	}
	return ansi.Truncate(value, width, ellipsis)
}
