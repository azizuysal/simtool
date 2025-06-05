package components

import (
	"fmt"
	"strings"

	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// DatabaseTableList renders the database table list view
type DatabaseTableList struct {
	Width        int
	Height       int
	DatabaseInfo *simulator.DatabaseInfo
	DatabaseFile *simulator.FileInfo
	Cursor       int
	Viewport     int
}

// NewDatabaseTableList creates a new database table list renderer
func NewDatabaseTableList(width, height int) *DatabaseTableList {
	return &DatabaseTableList{
		Width:  width,
		Height: height,
	}
}

// Update updates the list data
func (dtl *DatabaseTableList) Update(dbInfo *simulator.DatabaseInfo, dbFile *simulator.FileInfo, cursor, viewport int) {
	dtl.DatabaseInfo = dbInfo
	dtl.DatabaseFile = dbFile
	dtl.Cursor = cursor
	dtl.Viewport = viewport
}

// Render renders the database table list content
func (dtl *DatabaseTableList) Render() string {
	if dtl.DatabaseInfo == nil {
		return ui.DetailStyle().Render("No database loaded")
	}

	// Build header content
	header := dtl.buildHeader()
	
	// Calculate available space for table list
	headerLines := strings.Count(header, "\n") + 4 // header + separator + padding
	availableHeight := dtl.Height - headerLines
	
	// Calculate how many complete items we can show (each item = 3 lines)
	itemsPerScreen := availableHeight / 3
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	
	startIdx := dtl.Viewport
	endIdx := dtl.Viewport + itemsPerScreen
	if endIdx > len(dtl.DatabaseInfo.Tables) {
		endIdx = len(dtl.DatabaseInfo.Tables)
	}

	// Build content with header
	content := dtl.renderWithHeader(header, startIdx, endIdx)
	
	return content
}

// GetTitle returns the title for the database table list
func (dtl *DatabaseTableList) GetTitle() string {
	if dtl.DatabaseFile != nil {
		return fmt.Sprintf("%s Tables", dtl.DatabaseFile.Name)
	}
	return "Database Tables"
}

// GetFooter returns the footer for the database table list
func (dtl *DatabaseTableList) GetFooter() string {
	footer := "↑/k: up • ↓/j: down"
	if dtl.DatabaseInfo != nil && len(dtl.DatabaseInfo.Tables) > 0 && dtl.Cursor < len(dtl.DatabaseInfo.Tables) {
		footer += " • →/l: view table"
	}
	footer += " • ←/h: back • q: quit"

	// Add scroll info
	if dtl.DatabaseInfo != nil && len(dtl.DatabaseInfo.Tables) > 0 {
		headerLines := 6 // Approximate header lines
		availableHeight := dtl.Height - headerLines
		itemsPerScreen := availableHeight / 3
		if itemsPerScreen < 1 {
			itemsPerScreen = 1
		}
		scrollInfo := ui.FormatScrollInfo(dtl.Viewport, itemsPerScreen, len(dtl.DatabaseInfo.Tables))
		return footer + scrollInfo
	}
	
	return footer
}

// buildHeader builds the header content for the database table list
func (dtl *DatabaseTableList) buildHeader() string {
	if dtl.DatabaseInfo == nil {
		return ""
	}

	var s strings.Builder

	// Database info
	dbName := "Database"
	if dtl.DatabaseFile != nil {
		dbName = dtl.DatabaseFile.Name
	}
	s.WriteString(ui.NameStyle().Render(dbName))
	s.WriteString("\n")
	
	dbDetails := fmt.Sprintf("%s • %d tables • %s", 
		dtl.DatabaseInfo.Format, 
		dtl.DatabaseInfo.TableCount, 
		simulator.FormatSize(dtl.DatabaseInfo.FileSize))
	if dtl.DatabaseInfo.Version != "" {
		dbDetails = fmt.Sprintf("%s %s • %d tables • %s", 
			dtl.DatabaseInfo.Format, 
			dtl.DatabaseInfo.Version,
			dtl.DatabaseInfo.TableCount, 
			simulator.FormatSize(dtl.DatabaseInfo.FileSize))
	}
	s.WriteString(ui.DetailStyle().Render(dbDetails))

	return s.String()
}

// renderWithHeader renders the table list with header
func (dtl *DatabaseTableList) renderWithHeader(header string, startIdx, endIdx int) string {
	var s strings.Builder
	innerWidth := dtl.Width - 4 // Account for padding

	// Add header
	if header != "" {
		s.WriteString(header)
		s.WriteString("\n\n")
		s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
		s.WriteString("\n\n")
	}

	// Render table list
	if len(dtl.DatabaseInfo.Tables) == 0 {
		s.WriteString(ui.DetailStyle().Render("No tables found"))
	} else {
		for i := startIdx; i < endIdx && i < len(dtl.DatabaseInfo.Tables); i++ {
			table := dtl.DatabaseInfo.Tables[i]

			// Add spacing between items (except for first item)
			if i > startIdx {
				s.WriteString("\n\n")
			}

			// Format table name (no icon)
			tableName := table.Name

			// Format column details (no row count)
			var colNames []string
			for _, col := range table.Columns {
				colNames = append(colNames, col.Name)
			}
			
			colInfo := fmt.Sprintf("Columns: %s", strings.Join(colNames, ", "))

			if i == dtl.Cursor {
				// Selected item
				line1 := fmt.Sprintf("▶ %s", tableName)
				line2Prefix := "  "
				
				// Truncate colInfo to fit within the line, accounting for prefix
				maxColInfoLen := innerWidth - len(line2Prefix)
				if len(colInfo) > maxColInfoLen {
					colInfo = colInfo[:maxColInfoLen-3] + "..."
				}
				line2 := fmt.Sprintf("%s%s", line2Prefix, colInfo)

				// Pad to full width
				line1 = ui.PadLine(line1, innerWidth)
				line2 = ui.PadLine(line2, innerWidth)

				s.WriteString(ui.SelectedStyle().Render(line1))
				s.WriteString("\n")
				s.WriteString(ui.SelectedStyle().Render(line2))
			} else {
				// Non-selected item
				// Truncate colInfo to fit within innerWidth
				if len(colInfo) > innerWidth {
					colInfo = colInfo[:innerWidth-3] + "..."
				}
				
				s.WriteString(ui.ListItemStyle().Copy().Inherit(ui.NameStyle()).Render(tableName))
				s.WriteString("\n")
				s.WriteString(ui.ListItemStyle().Copy().Inherit(ui.DetailStyle()).Render(colInfo))
			}
		}
	}

	return s.String()
}