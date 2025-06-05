package file_viewer

import (
	"fmt"
	"strings"

	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// renderBinary renders binary file content as hex dump
func (fv *FileViewer) renderBinary() string {
	var s strings.Builder
	innerWidth := fv.Width - 4 // Account for padding

	// File info header
	info := fmt.Sprintf("Binary file • %s", simulator.FormatSize(fv.File.Size))
	s.WriteString(ui.DetailStyle().Render(info))
	s.WriteString("\n")
	s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
	s.WriteString("\n\n")

	// Display hex dump
	if fv.Content.BinaryData != nil {
		// Format hex dump
		hexLines := simulator.FormatHexDump(fv.Content.BinaryData, fv.Content.BinaryOffset)

		// Calculate visible range
		headerLines := 4 // Info + separator + padding
		visibleLines := fv.Height - headerLines
		
		startLine := fv.ContentViewport
		endLine := startLine + visibleLines
		if endLine > len(hexLines) {
			endLine = len(hexLines)
		}

		// Render visible hex lines
		lineCount := 0
		for i := startLine; i < endLine && i < len(hexLines); i++ {
			if lineCount > 0 {
				s.WriteString("\n")
			}
			s.WriteString(ui.DetailStyle().Render(hexLines[i]))
			lineCount++
		}

		// Don't pad - ContentBox will handle filling the space
	}

	return s.String()
}