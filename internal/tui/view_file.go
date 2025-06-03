package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// viewFileContent renders the file viewer
func (m Model) viewFileContent() string {
	var s strings.Builder
	
	// Always render header, even when loading
	if m.viewingFile != nil {
		headerText := filepath.Base(m.viewingFile.Path)
		s.WriteString(ui.FormatHeader(headerText, m.width))
	} else {
		s.WriteString(ui.FormatHeader("File Viewer", m.width))
	}
	
	if m.loadingContent {
		// Render a properly formatted loading message
		contentWidth := m.width - 6
		if contentWidth < 50 {
			contentWidth = 50
		}
		loadingMsg := "Loading file..."
		borderedContent := ui.BorderStyle.Width(contentWidth).Render(loadingMsg)
		s.WriteString(m.centerContent(borderedContent))
		s.WriteString("\n\n")
		s.WriteString(ui.FormatFooter("Please wait...", contentWidth, m.width))
		return s.String()
	}
	
	if m.viewingFile == nil || m.fileContent == nil {
		// Render a properly formatted error message
		contentWidth := m.width - 6
		if contentWidth < 50 {
			contentWidth = 50
		}
		errorMsg := "No file selected"
		borderedContent := ui.BorderStyle.Width(contentWidth).Render(errorMsg)
		s.WriteString(m.centerContent(borderedContent))
		s.WriteString("\n\n")
		s.WriteString(ui.FormatFooter("←/h: back • q: quit", contentWidth, m.width))
		return s.String()
	}

	// Calculate content area dimensions
	contentHeight := m.height - 8 // Header, footer, borders
	contentWidth := m.width - 6
	if contentWidth < 50 {
		contentWidth = 50
	}

	// Build content based on file type
	var listContent strings.Builder
	innerWidth := contentWidth - 4

	switch m.fileContent.Type {
	case simulator.FileTypeText:
		// Show file info
		info := fmt.Sprintf("Text file • %d lines • %s", 
			m.fileContent.TotalLines, 
			simulator.FormatSize(m.viewingFile.Size))
		listContent.WriteString(ui.DetailStyle.Render(info))
		listContent.WriteString("\n")
		listContent.WriteString(ui.DetailStyle.Render(strings.Repeat("─", innerWidth)))
		listContent.WriteString("\n\n")
		
		// Display text content with line numbers
		visibleLines := contentHeight - 4 // Account for info header
		startLine := m.contentViewport
		endLine := startLine + visibleLines
		if endLine > len(m.fileContent.Lines) {
			endLine = len(m.fileContent.Lines)
		}
		
		maxLineNumWidth := len(fmt.Sprintf("%d", m.contentOffset + endLine))
		
		lineCount := 0
		for i := startLine; i < endLine && i < len(m.fileContent.Lines); i++ {
			if lineCount > 0 {
				listContent.WriteString("\n")
			}
			lineNum := m.contentOffset + i + 1
			lineNumStr := fmt.Sprintf("%*d", maxLineNumWidth, lineNum)
			listContent.WriteString(ui.DetailStyle.Render(lineNumStr + " │ "))
			
			// Truncate very long lines for display
			line := m.fileContent.Lines[i]
			if len(line) > innerWidth - maxLineNumWidth - 4 {
				line = line[:innerWidth-maxLineNumWidth-7] + "..."
			}
			listContent.WriteString(line)
			lineCount++
		}
		
	case simulator.FileTypeImage:
		// Show image metadata
		if m.fileContent.ImageInfo != nil {
			info := m.fileContent.ImageInfo
			listContent.WriteString(ui.NameStyle.Render("Image Information"))
			listContent.WriteString("\n\n")
			listContent.WriteString(fmt.Sprintf("Format: %s\n", info.Format))
			listContent.WriteString(fmt.Sprintf("Dimensions: %d × %d pixels\n", info.Width, info.Height))
			listContent.WriteString(fmt.Sprintf("File size: %s", simulator.FormatSize(info.Size)))
			
			// Show preview if available
			if info.Preview != nil && len(info.Preview.Rows) > 0 {
				listContent.WriteString("\n\n")
				listContent.WriteString(ui.DetailStyle.Render("Preview:"))
				listContent.WriteString("\n\n")
				
				// Add the preview rows
				for i, row := range info.Preview.Rows {
					listContent.WriteString(row)
					if i < len(info.Preview.Rows)-1 {
						listContent.WriteString("\n")
					}
				}
			} else {
				listContent.WriteString("\n\n")
				listContent.WriteString(ui.DetailStyle.Render("(Preview generation failed)"))
			}
		}
		
	case simulator.FileTypeBinary:
		// Show hex dump
		info := fmt.Sprintf("Binary file • %s", simulator.FormatSize(m.viewingFile.Size))
		listContent.WriteString(ui.DetailStyle.Render(info))
		listContent.WriteString("\n")
		listContent.WriteString(ui.DetailStyle.Render(strings.Repeat("─", innerWidth)))
		listContent.WriteString("\n\n")
		
		// Display hex dump with viewport
		if m.fileContent.BinaryData != nil {
			// Use the actual file offset from the loaded chunk
			hexLines := simulator.FormatHexDump(m.fileContent.BinaryData, m.fileContent.BinaryOffset)
			
			// Calculate visible range for hex dump
			visibleLines := contentHeight - 4 // Account for info header
			startLine := m.contentViewport
			endLine := startLine + visibleLines
			if endLine > len(hexLines) {
				endLine = len(hexLines)
			}
			
			// Only show visible lines
			lineCount := 0
			for i := startLine; i < endLine && i < len(hexLines); i++ {
				if lineCount > 0 {
					listContent.WriteString("\n")
				}
				listContent.WriteString(ui.DetailStyle.Copy().Foreground(lipgloss.Color("245")).Render(hexLines[i]))
				lineCount++
			}
		}
	}
	
	// For file content, we need to ensure consistent height without adding extra padding at the bottom
	content := listContent.String()
	lines := strings.Split(content, "\n")
	currentLineCount := len(lines)
	
	// For images, we need to handle viewport scrolling if the preview is too large
	if m.fileContent.Type == simulator.FileTypeImage && currentLineCount > contentHeight {
		// Apply viewport scrolling for large image previews
		startLine := m.contentViewport
		endLine := startLine + contentHeight
		if endLine > currentLineCount {
			endLine = currentLineCount
		}
		
		// Extract only the visible lines
		visibleLines := lines[startLine:endLine]
		content = strings.Join(visibleLines, "\n")
		currentLineCount = len(visibleLines)
	}
	
	// Only pad if we have fewer lines than the content area can hold
	if currentLineCount < contentHeight {
		var paddedBuilder strings.Builder
		paddedBuilder.WriteString(content)
		// Add empty lines to reach the desired height
		// Subtract 1 to avoid extra blank line at the bottom
		for i := currentLineCount; i < contentHeight - 1; i++ {
			paddedBuilder.WriteString("\n")
		}
		content = paddedBuilder.String()
	}

	// Apply border and center
	borderedList := ui.BorderStyle.Width(contentWidth).Render(content)
	s.WriteString(m.centerContent(borderedList))

	s.WriteString("\n\n")

	// Footer
	footerText := "↑/k: scroll up • ↓/j: scroll down • ←/h: back • q: quit"
	
	// Add scroll indicator with arrows
	if m.fileContent != nil {
		var startLine, endLine, totalLines int
		var hasContent bool
		
		switch m.fileContent.Type {
		case simulator.FileTypeText:
			if m.fileContent.TotalLines > 0 {
				hasContent = true
				startLine = m.contentOffset + m.contentViewport + 1
				endLine = m.contentOffset + m.contentViewport + len(m.fileContent.Lines)
				totalLines = m.fileContent.TotalLines
			}
		case simulator.FileTypeImage:
			if m.fileContent.ImageInfo != nil && m.fileContent.ImageInfo.Preview != nil {
				hasContent = true
				// Calculate total lines (metadata + preview)
				totalLines = 8 + len(m.fileContent.ImageInfo.Preview.Rows)
				visibleLines := contentHeight - 4
				startLine = m.contentViewport + 1
				endLine = m.contentViewport + visibleLines
				if endLine > totalLines {
					endLine = totalLines
				}
			}
		case simulator.FileTypeBinary:
			if m.fileContent.BinaryData != nil {
				if m.fileContent.TotalSize > 0 {
					hasContent = true
					// Calculate total lines based on file size (16 bytes per line)
					totalLines = int((m.fileContent.TotalSize + 15) / 16)
					
					// Calculate the absolute line position in the file
					// BinaryOffset tells us where this chunk starts in the file
					chunkStartLine := int(m.fileContent.BinaryOffset / 16)
					
					// The actual lines we're showing
					visibleLines := contentHeight - 4
					startLine = chunkStartLine + m.contentViewport + 1
					endLine = startLine + visibleLines - 1
					
					// Don't exceed the actual data we have
					hexLines := simulator.FormatHexDump(m.fileContent.BinaryData, m.fileContent.BinaryOffset)
					maxEndLine := chunkStartLine + len(hexLines)
					if endLine > maxEndLine {
						endLine = maxEndLine
					}
					
					// Don't exceed total lines in file
					if endLine > totalLines {
						endLine = totalLines
					}
				}
			}
		}
		
		if hasContent {
			// Add scroll indicators with arrows
			if m.contentViewport > 0 && endLine < totalLines {
				footerText += fmt.Sprintf(" (%d-%d of %d) ↑↓", startLine, endLine, totalLines)
			} else if m.contentViewport > 0 {
				footerText += fmt.Sprintf(" (%d-%d of %d) ↑", startLine, endLine, totalLines)
			} else if endLine < totalLines {
				footerText += fmt.Sprintf(" (%d-%d of %d) ↓", startLine, endLine, totalLines)
			} else {
				footerText += fmt.Sprintf(" (%d-%d of %d)", startLine, endLine, totalLines)
			}
		}
	}
	
	s.WriteString(ui.FormatFooter(footerText,
		lipgloss.Width(strings.Split(borderedList, "\n")[0]), m.width))

	return s.String()
}