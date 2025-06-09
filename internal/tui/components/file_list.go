package components

import (
	"fmt"
	"strings"

	"github.com/azizuysal/simtool/internal/config"
	"github.com/azizuysal/simtool/internal/simulator"
	"github.com/azizuysal/simtool/internal/ui"
)

// FileList renders the file list view
type FileList struct {
	Width       int
	Height      int
	Files       []simulator.FileInfo
	Cursor      int
	Viewport    int
	App         *simulator.App
	Breadcrumbs []string
	Keys        *config.KeysConfig
}

// NewFileList creates a new file list renderer
func NewFileList(width, height int) *FileList {
	return &FileList{
		Width:  width,
		Height: height,
	}
}

// Update updates the list data
func (fl *FileList) Update(files []simulator.FileInfo, cursor, viewport int, app *simulator.App, breadcrumbs []string, keys *config.KeysConfig) {
	fl.Files = files
	fl.Cursor = cursor
	fl.Viewport = viewport
	fl.App = app
	fl.Breadcrumbs = breadcrumbs
	fl.Keys = keys
}

// Render renders the file list content
func (fl *FileList) Render() string {
	// Build header content
	header := fl.buildHeader()
	
	// Calculate available space for file list
	headerLines := strings.Count(header, "\n") + 4 // header + separator + padding
	availableHeight := fl.Height - headerLines
	
	// Calculate how many complete items we can show (each item = 3 lines)
	itemsPerScreen := availableHeight / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	
	// Ensure we don't show partial items by adjusting availableHeight
	availableHeight = itemsPerScreen * 3
	
	startIdx := fl.Viewport
	endIdx := fl.Viewport + itemsPerScreen
	if endIdx > len(fl.Files) {
		endIdx = len(fl.Files)
	}

	// Build content with header
	content := fl.renderWithHeader(header, startIdx, endIdx, availableHeight)
	
	return content
}

// GetTitle returns the title for the file list
func (fl *FileList) GetTitle() string {
	if fl.App != nil {
		return fmt.Sprintf("%s Files", fl.App.Name)
	}
	return "Files"
}

// GetFooter returns the footer for the file list
func (fl *FileList) GetFooter() string {
	if fl.Keys == nil {
		// Fallback to default if keys not set
		footer := "↑/k: up • ↓/j: down"
		if len(fl.Files) > 0 && fl.Cursor < len(fl.Files) {
			if fl.Files[fl.Cursor].IsDirectory {
				footer += " • →/l: enter"
			} else {
				footer += " • →/l: view file"
			}
			footer += " • space: open in Finder"
		}
		footer += " • ←/h: back • q: quit"

		// Add scroll info
		// Calculate actual header lines for this specific render
		header := fl.buildHeader()
		headerLines := strings.Count(header, "\n") + 4 // header + separator + padding
		availableHeight := fl.Height - headerLines
		itemsPerScreen := availableHeight / 3
		if itemsPerScreen < 1 {
			itemsPerScreen = 1
		}
		scrollInfo := ui.FormatScrollInfo(fl.Viewport, itemsPerScreen, len(fl.Files))
		return footer + scrollInfo
	}
	
	// Build footer from configured keys
	var parts []string
	
	if up := fl.Keys.FormatKeyAction("up", "up"); up != "" {
		parts = append(parts, up)
	}
	if down := fl.Keys.FormatKeyAction("down", "down"); down != "" {
		parts = append(parts, down)
	}
	
	if len(fl.Files) > 0 && fl.Cursor < len(fl.Files) {
		if fl.Files[fl.Cursor].IsDirectory {
			if right := fl.Keys.FormatKeyAction("right", "enter"); right != "" {
				parts = append(parts, right)
			}
		} else {
			if right := fl.Keys.FormatKeyAction("right", "view file"); right != "" {
				parts = append(parts, right)
			}
		}
		if open := fl.Keys.FormatKeyAction("open", "open in Finder"); open != "" {
			parts = append(parts, open)
		}
	}
	
	if left := fl.Keys.FormatKeyAction("left", "back"); left != "" {
		parts = append(parts, left)
	}
	if quit := fl.Keys.FormatKeyAction("quit", "quit"); quit != "" {
		parts = append(parts, quit)
	}
	
	footer := strings.Join(parts, " • ")
	
	// Add scroll info
	// Calculate actual header lines for this specific render
	header := fl.buildHeader()
	headerLines := strings.Count(header, "\n") + 4 // header + separator + padding
	availableHeight := fl.Height - headerLines
	itemsPerScreen := availableHeight / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	scrollInfo := ui.FormatScrollInfo(fl.Viewport, itemsPerScreen, len(fl.Files))
	return footer + scrollInfo
}

// buildHeader builds the header content for the file list
func (fl *FileList) buildHeader() string {
	if fl.App == nil {
		return ""
	}

	var s strings.Builder

	// App info
	s.WriteString(ui.NameStyle().Render(fl.App.Name))
	s.WriteString("\n")
	
	appDetails := fmt.Sprintf("%s • v%s • %s", fl.App.BundleID, fl.App.Version, simulator.FormatSize(fl.App.Size))
	if fl.App.Version == "" {
		appDetails = fmt.Sprintf("%s • %s", fl.App.BundleID, simulator.FormatSize(fl.App.Size))
	}
	s.WriteString(ui.DetailStyle().Render(appDetails))

	// Breadcrumbs if not at root
	if len(fl.Breadcrumbs) > 0 {
		s.WriteString("\n\n")
		breadcrumbPath := strings.Join(fl.Breadcrumbs, "/") + "/"
		s.WriteString(ui.FolderStyle().Render(breadcrumbPath))
	}

	return s.String()
}

// calculateItemsPerScreen calculates how many items fit on screen
func (fl *FileList) calculateItemsPerScreen(availableHeight int) int {
	// Each item takes 3 lines (name + details + blank line)
	itemsPerScreen := availableHeight / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	return itemsPerScreen
}

// renderWithHeader renders the file list with header
func (fl *FileList) renderWithHeader(header string, startIdx, endIdx int, availableHeight int) string {
	var s strings.Builder
	innerWidth := fl.Width - 4 // Account for padding

	// Add header
	if header != "" {
		s.WriteString(header)
		s.WriteString("\n\n")
		s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
		s.WriteString("\n\n")
	}

	// Count lines used so far
	linesUsed := 0

	// Render file list
	if len(fl.Files) == 0 {
		s.WriteString(ui.DetailStyle().Render("No files in folder"))
		linesUsed = 1
	} else {
		for i := startIdx; i < endIdx && linesUsed < availableHeight; i++ {
			file := fl.Files[i]

			// Add spacing between items (except for first item)
			if i > startIdx {
				s.WriteString("\n\n")
				linesUsed++ // Empty line between items
			}

			// Format file name with directory indicator
			fileName := file.Name
			if file.IsDirectory {
				fileName = fileName + "/"
			}

			// Format file details
			sizeText := simulator.FormatSize(file.Size)
			createdText := simulator.FormatFileDate(file.CreatedAt)
			modifiedText := simulator.FormatFileDate(file.ModifiedAt)
			detailText := fmt.Sprintf("%s • Created %s • Modified %s", sizeText, createdText, modifiedText)

			if i == fl.Cursor {
				// Selected item
				line1 := fmt.Sprintf("▶ %s", fileName)
				line2 := fmt.Sprintf("  %s", detailText)

				// Pad to full width
				line1 = ui.PadLine(line1, innerWidth)
				line2 = ui.PadLine(line2, innerWidth)

				s.WriteString(ui.SelectedStyle().Render(line1))
				s.WriteString("\n")
				s.WriteString(ui.SelectedStyle().Render(line2))
			} else {
				// Non-selected item
				if file.IsDirectory {
					s.WriteString(ui.ListItemStyle().Copy().Inherit(ui.FolderStyle()).Render(fileName))
				} else {
					s.WriteString(ui.ListItemStyle().Copy().Inherit(ui.NameStyle()).Render(fileName))
				}
				s.WriteString("\n")
				s.WriteString(ui.ListItemStyle().Copy().Inherit(ui.DetailStyle()).Render(detailText))
			}
			linesUsed += 2 // Each item uses 2 lines
		}
	}

	return s.String()
}