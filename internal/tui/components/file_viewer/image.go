package file_viewer

import (
	"fmt"
	"strings"

	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// renderImage renders image file content with preview
func (fv *FileViewer) renderImage() string {
	if fv.Content.ImageInfo == nil {
		return ui.DetailStyle().Render("Error loading image")
	}

	return fv.renderImageContent()
}

// renderImageContent renders the image content for the content box
func (fv *FileViewer) renderImageContent() string {
	var s strings.Builder
	innerWidth := fv.Width - 4 // Account for content box padding
	
	// Image info header (similar to text and binary files)
	info := fmt.Sprintf("Image file • %s • %d × %d • %s",
		fv.Content.ImageInfo.Format,
		fv.Content.ImageInfo.Width,
		fv.Content.ImageInfo.Height,
		simulator.FormatSize(fv.Content.ImageInfo.Size))
	s.WriteString(ui.DetailStyle().Render(info))
	s.WriteString("\n")
	s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
	s.WriteString("\n\n")
	
	// Count lines used by header
	headerLines := 4 // Info + separator + padding
	
	// Build preview content if available
	if fv.Content.ImageInfo.Preview != nil && len(fv.Content.ImageInfo.Preview.Rows) > 0 {
		// Calculate remaining space for preview
		remainingHeight := fv.Height - headerLines
		if remainingHeight < 5 {
			remainingHeight = 5
		}
		
		previewRows := fv.Content.ImageInfo.Preview.Rows
		
		// Calculate how many rows we can show with viewport scrolling
		startRow := fv.ContentViewport
		maxRows := len(previewRows) - startRow
		if maxRows > remainingHeight {
			maxRows = remainingHeight
		}
		
		// Add preview rows
		lineCount := 0
		for i := 0; i < maxRows && startRow+i < len(previewRows); i++ {
			if lineCount > 0 {
				s.WriteString("\n")
			}
			row := previewRows[startRow+i]
			
			// Center each row if needed
			previewWidth := fv.Content.ImageInfo.Preview.Width
			if innerWidth > previewWidth {
				leftPadding := (innerWidth - previewWidth) / 2
				s.WriteString(strings.Repeat(" ", leftPadding))
			}
			s.WriteString(row)
			lineCount++
		}
	}
	
	// Don't pad - ContentBox will handle filling the space
	
	return s.String()
}

