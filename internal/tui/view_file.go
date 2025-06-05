package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// treeNode represents a node in the file tree
type treeNode struct {
	name     string
	isDir    bool
	children map[string]*treeNode
}

// buildTreeFromPaths builds a tree structure from flat paths
func buildTreeFromPaths(entries []simulator.ArchiveEntry) *treeNode {
	root := &treeNode{
		name:     "",
		isDir:    true,
		children: make(map[string]*treeNode),
	}
	
	for _, entry := range entries {
		parts := strings.Split(entry.Name, "/")
		current := root
		
		for i, part := range parts {
			if part == "" {
				continue
			}
			
			if _, exists := current.children[part]; !exists {
				isDir := i < len(parts)-1 || entry.IsDir
				current.children[part] = &treeNode{
					name:     part,
					isDir:    isDir,
					children: make(map[string]*treeNode),
				}
			}
			current = current.children[part]
		}
	}
	
	return root
}

// renderTree renders a tree node with proper box drawing characters
func renderTree(node *treeNode, prefix string, isLast bool, lines *[]string) {
	if node.name != "" {
		var line string
		if isLast {
			line = prefix + "└── "
		} else {
			line = prefix + "├── "
		}
		
		name := node.name
		if node.isDir {
			name = name + "/"
		}
		*lines = append(*lines, line + name)
	}
	
	// Sort children for consistent output
	childNames := make([]string, 0, len(node.children))
	for name := range node.children {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)
	
	for i, childName := range childNames {
		child := node.children[childName]
		childPrefix := prefix
		if node.name != "" {
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
		}
		renderTree(child, childPrefix, i == len(childNames)-1, lines)
	}
}

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
		// Center the footer text
		footerText := "Please wait..."
		if m.width > lipgloss.Width(footerText) {
			leftPadding := (m.width - lipgloss.Width(footerText)) / 2
			s.WriteString(strings.Repeat(" ", leftPadding))
		}
		s.WriteString(ui.FooterStyle.Render(footerText))
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
		// Center the footer text
		footerText := "←/h: back • q: quit"
		if m.width > lipgloss.Width(footerText) {
			leftPadding := (m.width - lipgloss.Width(footerText)) / 2
			s.WriteString(strings.Repeat(" ", leftPadding))
		}
		s.WriteString(ui.FooterStyle.Render(footerText))
		return s.String()
	}

	// SPECIAL CASE: Handle images separately to preserve ANSI codes
	if m.fileContent.Type == simulator.FileTypeImage && 
		m.fileContent.ImageInfo != nil && 
		m.fileContent.ImageInfo.Preview != nil && 
		len(m.fileContent.ImageInfo.Preview.Rows) > 0 {
		
		// Metadata box
		metaContent := fmt.Sprintf("Image Information\n\nFormat: %s\nDimensions: %d × %d pixels\nFile size: %s",
			m.fileContent.ImageInfo.Format,
			m.fileContent.ImageInfo.Width, 
			m.fileContent.ImageInfo.Height,
			simulator.FormatSize(m.fileContent.ImageInfo.Size))
		
		// Calculate content width
		contentWidth := m.width - 6
		if contentWidth < 50 {
			contentWidth = 50
		}
		
		metaBorder := ui.BorderStyle.Width(contentWidth).Render(metaContent)
		s.WriteString(m.centerContent(metaBorder))
		s.WriteString("\n\n") // Add spacing between boxes
		
		// Build preview content in a box
		var previewContent strings.Builder
		
		// Count lines used so far
		currentContent := s.String()
		linesUsed := strings.Count(currentContent, "\n") + 1
		
		// Calculate remaining space for preview box
		// Reserve lines for: border (2), footer (2), spacing (2)
		availableForPreview := m.height - linesUsed - 6
		if availableForPreview < 10 {
			availableForPreview = 10 // Minimum preview size
		}
		
		// Calculate inner dimensions for the preview
		innerWidth := contentWidth - 4 // Account for border padding
		innerHeight := availableForPreview - 1 // Account for top/bottom border (adjusted for images)
		
		// Add preview rows with padding
		previewRows := m.fileContent.ImageInfo.Preview.Rows
		maxRows := len(previewRows)
		if maxRows > innerHeight {
			maxRows = innerHeight
		}
		
		// Add top padding if preview is smaller than available space
		topPadding := (innerHeight - maxRows) / 2
		for i := 0; i < topPadding; i++ {
			previewContent.WriteString("\n")
		}
		
		// Add preview rows with left padding to center them
		for i := 0; i < maxRows; i++ {
			row := previewRows[i]
			// Calculate padding for centering (preview width should be less than innerWidth)
			previewWidth := m.fileContent.ImageInfo.Preview.Width
			leftPadding := 0
			if innerWidth > previewWidth {
				leftPadding = (innerWidth - previewWidth) / 2
			}
			if leftPadding > 0 {
				previewContent.WriteString(strings.Repeat(" ", leftPadding))
			}
			previewContent.WriteString(row)
			if i < maxRows-1 || i+topPadding < innerHeight-1 {
				previewContent.WriteString("\n")
			}
		}
		
		// Add bottom padding
		currentLines := topPadding + maxRows
		for i := currentLines; i < innerHeight; i++ {
			previewContent.WriteString("\n")
		}
		
		// Apply border to preview
		previewBorder := ui.BorderStyle.Width(contentWidth).Render(previewContent.String())
		s.WriteString(m.centerContent(previewBorder))
		
		// Display SVG warning if present
		if m.svgWarning != "" {
			warningStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")). // Orange color
				Align(lipgloss.Center)
			
			warningText := warningStyle.Width(contentWidth).Render(m.svgWarning)
			centeredWarning := m.centerContent(warningText)
			s.WriteString("\n")
			s.WriteString(centeredWarning)
			s.WriteString("\n")
		} else {
			s.WriteString("\n\n")
		}
		
		// Center the footer text
		footerText := "←/h: back • q: quit"
		if m.width > lipgloss.Width(footerText) {
			leftPadding := (m.width - lipgloss.Width(footerText)) / 2
			s.WriteString(strings.Repeat(" ", leftPadding))
		}
		s.WriteString(ui.FooterStyle.Render(footerText))
		
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
			// Apply syntax highlighting
			fileExt := filepath.Ext(m.viewingFile.Path)
			highlightedLine := simulator.GetSyntaxHighlightedLine(line, fileExt)
			listContent.WriteString(highlightedLine)
			lineCount++
		}
		
	case simulator.FileTypeImage:
		// This case should not be reached due to early return above
		// But include it for completeness
		listContent.WriteString("Image file")
		
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
			
	case simulator.FileTypeArchive:
		// Show archive contents
		if m.fileContent.ArchiveInfo != nil {
			archInfo := m.fileContent.ArchiveInfo
			
			// Build info string with file/folder counts and compression ratio
			var infoStr string
			if archInfo.TotalSize > 0 {
				ratio := float64(archInfo.CompressedSize) / float64(archInfo.TotalSize) * 100
				infoStr = fmt.Sprintf("%s Archive • %d files, %d folders • %.1f%% compression",
					archInfo.Format, archInfo.FileCount, archInfo.FolderCount, ratio)
			} else {
				infoStr = fmt.Sprintf("%s Archive • %d files, %d folders",
					archInfo.Format, archInfo.FileCount, archInfo.FolderCount)
			}
			
			listContent.WriteString(ui.DetailStyle.Render(infoStr))
			listContent.WriteString("\n")
			listContent.WriteString(ui.DetailStyle.Render(strings.Repeat("─", innerWidth)))
			listContent.WriteString("\n\n")
			
			// Build tree structure
			tree := buildTreeFromPaths(archInfo.Entries)
			
			// Render tree to lines
			var treeLines []string
			// Render root's children directly
			childNames := make([]string, 0, len(tree.children))
			for name := range tree.children {
				childNames = append(childNames, name)
			}
			sort.Strings(childNames)
			
			for i, childName := range childNames {
				child := tree.children[childName]
				renderTree(child, "", i == len(childNames)-1, &treeLines)
			}
			
			// Calculate visible range
			headerLines := 3 // Info + separator + blank line
			// Reserve one line for padding above the title
			availableLines := contentHeight - headerLines - 1
			startIdx := m.contentViewport
			endIdx := startIdx + availableLines
			
			if endIdx > len(treeLines) {
				endIdx = len(treeLines)
			}
			
			// Display visible tree lines
			linesWritten := 0
			for i := startIdx; i < endIdx; i++ {
				if i < len(treeLines) {
					if i > startIdx {
						listContent.WriteString("\n")
					}
					listContent.WriteString(treeLines[i])
					linesWritten++
				}
			}
			
			// If we didn't fill all available lines (e.g., at the bottom of the list),
			// add empty lines to maintain consistent height
			for i := linesWritten; i < availableLines; i++ {
				listContent.WriteString("\n")
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
		// Subtract 1 to leave room for proper spacing
		for i := currentLineCount; i < contentHeight - 1; i++ {
			paddedBuilder.WriteString("\n")
		}
		content = paddedBuilder.String()
	}

	// Apply border and center
	var borderedList string
	// Special handling for images to preserve ANSI codes
	if m.fileContent.Type == simulator.FileTypeImage && 
		m.fileContent.ImageInfo != nil && 
		m.fileContent.ImageInfo.Preview != nil &&
		len(m.fileContent.ImageInfo.Preview.Rows) > 0 {
		// For images, use a simpler border without padding to preserve ANSI codes
		simpleBorder := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(contentWidth)
		borderedList = simpleBorder.Render(content)
		s.WriteString(m.centerContent(borderedList))
	} else {
		// Use normal border for text and binary files
		borderedList = ui.BorderStyle.Width(contentWidth).Render(content)
		s.WriteString(m.centerContent(borderedList))
	}

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
	
	// Center the footer text
	if m.width > lipgloss.Width(footerText) {
		leftPadding := (m.width - lipgloss.Width(footerText)) / 2
		s.WriteString(strings.Repeat(" ", leftPadding))
	}
	s.WriteString(ui.FooterStyle.Render(footerText))

	return s.String()
}