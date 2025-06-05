package file_viewer

import (
	"fmt"
	"path/filepath"

	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// FileViewer renders file content based on file type
type FileViewer struct {
	Width          int
	Height         int
	File           *simulator.FileInfo
	Content        *simulator.FileContent
	ContentViewport int
	ContentOffset   int
	SVGWarning     string
}

// NewFileViewer creates a new file viewer
func NewFileViewer(width, height int) *FileViewer {
	return &FileViewer{
		Width:  width,
		Height: height,
	}
}

// Update updates the viewer data
func (fv *FileViewer) Update(file *simulator.FileInfo, content *simulator.FileContent, viewport, offset int, svgWarning string) {
	fv.File = file
	fv.Content = content
	fv.ContentViewport = viewport
	fv.ContentOffset = offset
	fv.SVGWarning = svgWarning
}

// Render renders the file content based on type
func (fv *FileViewer) Render() string {
	if fv.File == nil || fv.Content == nil {
		return ui.DetailStyle.Render("No file selected")
	}

	switch fv.Content.Type {
	case simulator.FileTypeText:
		return fv.renderText()
	case simulator.FileTypeImage:
		return fv.renderImage()
	case simulator.FileTypeBinary:
		return fv.renderBinary()
	case simulator.FileTypeArchive:
		return fv.renderArchive()
	case simulator.FileTypeDatabase:
		return fv.renderDatabase()
	default:
		return ui.DetailStyle.Render("Unknown file type")
	}
}

// GetTitle returns the title for the file viewer
func (fv *FileViewer) GetTitle() string {
	if fv.File != nil {
		return filepath.Base(fv.File.Path)
	}
	return "File Viewer"
}

// GetFooter returns the footer for the file viewer
func (fv *FileViewer) GetFooter() string {
	footer := "↑/k: scroll up • ↓/j: scroll down • ←/h: back • q: quit"

	// Add scroll indicator
	scrollInfo := fv.getScrollInfo()
	if scrollInfo != "" {
		footer += " " + scrollInfo
	}

	return footer
}

// GetStatus returns the status message for the file viewer
func (fv *FileViewer) GetStatus() string {
	if fv.SVGWarning != "" {
		return ui.StatusStyle.Render(fv.SVGWarning)
	}
	return ""
}

// getScrollInfo returns scroll information based on file type
func (fv *FileViewer) getScrollInfo() string {
	if fv.Content == nil {
		return ""
	}

	var startLine, endLine, totalLines int
	var hasContent bool

	contentHeight := fv.Height // We're already in content dimensions

	switch fv.Content.Type {
	case simulator.FileTypeText:
		if fv.Content.TotalLines > 0 {
			hasContent = true
			startLine = fv.ContentOffset + fv.ContentViewport + 1
			endLine = fv.ContentOffset + fv.ContentViewport + len(fv.Content.Lines)
			totalLines = fv.Content.TotalLines
		}
	case simulator.FileTypeImage:
		if fv.Content.ImageInfo != nil && fv.Content.ImageInfo.Preview != nil {
			hasContent = true
			totalLines = 8 + len(fv.Content.ImageInfo.Preview.Rows)
			visibleLines := contentHeight - 4 // Account for header
			startLine = fv.ContentViewport + 1
			endLine = fv.ContentViewport + visibleLines
			if endLine > totalLines {
				endLine = totalLines
			}
		}
	case simulator.FileTypeBinary:
		if fv.Content.BinaryData != nil && fv.Content.TotalSize > 0 {
			hasContent = true
			totalLines = int((fv.Content.TotalSize + 15) / 16)
			
			chunkStartLine := int(fv.Content.BinaryOffset / 16)
			visibleLines := contentHeight - 4
			startLine = chunkStartLine + fv.ContentViewport + 1
			endLine = startLine + visibleLines - 1
			
			hexLines := simulator.FormatHexDump(fv.Content.BinaryData, fv.Content.BinaryOffset)
			maxEndLine := chunkStartLine + len(hexLines)
			if endLine > maxEndLine {
				endLine = maxEndLine
			}
			if endLine > totalLines {
				endLine = totalLines
			}
		}
	case simulator.FileTypeArchive:
		if fv.Content.ArchiveInfo != nil {
			hasContent = true
			// Count tree lines (this is approximate)
			totalLines = len(fv.Content.ArchiveInfo.Entries) * 2 // Rough estimate
			startLine = fv.ContentViewport + 1
			endLine = startLine + contentHeight - 4
			if endLine > totalLines {
				endLine = totalLines
			}
		}
	case simulator.FileTypeDatabase:
		if fv.Content.DatabaseInfo != nil {
			hasContent = true
			// Each table takes approximately 6-8 lines (header + columns + sample + spacing)
			totalLines = len(fv.Content.DatabaseInfo.Tables) * 8
			startLine = fv.ContentViewport + 1
			endLine = startLine + contentHeight - 4
			if endLine > totalLines {
				endLine = totalLines
			}
		}
	}

	if hasContent {
		// Add scroll indicators with arrows
		canScrollUp := fv.ContentViewport > 0 || fv.ContentOffset > 0
		canScrollDown := endLine < totalLines

		if canScrollUp && canScrollDown {
			return fmt.Sprintf("(%d-%d of %d) ↑↓", startLine, endLine, totalLines)
		} else if canScrollUp {
			return fmt.Sprintf("(%d-%d of %d) ↑", startLine, endLine, totalLines)
		} else if canScrollDown {
			return fmt.Sprintf("(%d-%d of %d) ↓", startLine, endLine, totalLines)
		} else {
			return fmt.Sprintf("(%d-%d of %d)", startLine, endLine, totalLines)
		}
	}

	return ""
}

// renderText is implemented in text.go
// renderImage is implemented in image.go
// renderBinary is implemented in binary.go
// renderArchive is implemented in archive.go