package components

import (
	"fmt"
	"strings"
	"unicode"

	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// DatabaseTableContent renders individual table content view
type DatabaseTableContent struct {
	Width           int
	Height          int
	Table           *simulator.TableInfo
	TableData       []map[string]any
	DatabaseFile    *simulator.FileInfo
	Viewport        int
	DataOffset      int
}

// NewDatabaseTableContent creates a new database table content renderer
func NewDatabaseTableContent(width, height int) *DatabaseTableContent {
	return &DatabaseTableContent{
		Width:  width,
		Height: height,
	}
}

// Update updates the table data
func (dtc *DatabaseTableContent) Update(table *simulator.TableInfo, data []map[string]any, dbFile *simulator.FileInfo, viewport, dataOffset int) {
	dtc.Table = table
	dtc.TableData = data
	dtc.DatabaseFile = dbFile
	dtc.Viewport = viewport
	dtc.DataOffset = dataOffset
}

// Render renders the table content
func (dtc *DatabaseTableContent) Render() string {
	if dtc.Table == nil {
		return ui.DetailStyle().Render("No table selected")
	}

	// Build header content
	header := dtc.buildHeader()
	
	// Calculate available space for table data
	headerLines := strings.Count(header, "\n") + 4 // header + separator + padding
	availableHeight := dtc.Height - headerLines
	
	// Build content with header
	content := dtc.renderWithHeader(header, availableHeight)
	
	return content
}

// GetTitle returns the title for the table content view
func (dtc *DatabaseTableContent) GetTitle() string {
	if dtc.Table != nil {
		return fmt.Sprintf("Table: %s", dtc.Table.Name)
	}
	return "Table Content"
}

// GetFooter returns the footer for the table content view
func (dtc *DatabaseTableContent) GetFooter() string {
	footer := "↑/k: scroll up • ↓/j: scroll down • ←/h: back • q: quit"
	
	// Add scroll info for data rows
	if dtc.Table != nil && dtc.Table.RowCount > 0 {
		startRow := dtc.DataOffset + 1
		endRow := dtc.DataOffset + len(dtc.TableData)
		totalRows := int(dtc.Table.RowCount)
		
		if endRow > totalRows {
			endRow = totalRows
		}
		
		scrollInfo := fmt.Sprintf(" (%d-%d of %d)", startRow, endRow, totalRows)
		footer += scrollInfo
	}
	
	return footer
}

// buildHeader builds the header content for the table
func (dtc *DatabaseTableContent) buildHeader() string {
	if dtc.Table == nil {
		return ""
	}

	var s strings.Builder

	// Table info
	s.WriteString(ui.NameStyle().Render(dtc.Table.Name))
	s.WriteString("\n")
	
	tableDetails := fmt.Sprintf("%d rows • %d columns", dtc.Table.RowCount, len(dtc.Table.Columns))
	s.WriteString(ui.DetailStyle().Render(tableDetails))

	return s.String()
}

// renderWithHeader renders the table content with header
func (dtc *DatabaseTableContent) renderWithHeader(header string, availableHeight int) string {
	var s strings.Builder
	innerWidth := dtc.Width - 4 // Account for padding

	// Add header
	if header != "" {
		s.WriteString(header)
		s.WriteString("\n\n")
		s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
		s.WriteString("\n\n")
	}

	// Calculate column widths first to align delimiters
	var columnWidths []int
	visibleColumns := 0
	totalUsedWidth := 0
	
	if len(dtc.Table.Columns) > 0 {
		// Reserve space for " | ..." if we won't show all columns
		reservedSpace := 0
		if len(dtc.Table.Columns) > 1 {
			reservedSpace = 6 // " | ..."
		}
		
		// First pass: calculate minimum column widths based on headers and sample data
		for i, col := range dtc.Table.Columns {
			colHeader := col.Name
			if col.PK {
				colHeader += "*"
			}
			
			// Start with header width (rune count)
			minWidth := len([]rune(colHeader))
			
			// Check ALL loaded rows to get accurate data width
			// This ensures we calculate based on the actual data we'll display
			for j := 0; j < len(dtc.TableData); j++ {
				row := dtc.TableData[j]
				var valStr string
				if val, ok := row[col.Name]; ok {
					valStr = fmt.Sprintf("%v", val)
					// Sanitize the value for display
					valStr = sanitizeForDisplay(valStr)
				} else {
					valStr = "NULL"
				}
				// Use rune count for width calculation to handle multi-byte chars
				runeCount := len([]rune(valStr))
				if runeCount > minWidth {
					minWidth = runeCount
				}
			}
			
			
			// Check if we can fit this column
			separatorWidth := 0
			if i > 0 {
				separatorWidth = 3 // " | "
			}
			
			// Check if we need to reserve space for "..."
			effectiveWidth := innerWidth
			if i < len(dtc.Table.Columns) - 1 {
				// Not the last column, so we might need "..."
				effectiveWidth = innerWidth - reservedSpace
			}
			
			if totalUsedWidth + separatorWidth + minWidth <= effectiveWidth {
				// Column fits entirely
				columnWidths = append(columnWidths, minWidth)
				totalUsedWidth += separatorWidth + minWidth
				visibleColumns++
			} else {
				// Column doesn't fit entirely, but add it partially if there's enough space
				remainingSpace := effectiveWidth - totalUsedWidth - separatorWidth
				if remainingSpace >= 10 { // Only add if we have at least 10 chars for readability
					columnWidths = append(columnWidths, remainingSpace)
					totalUsedWidth += separatorWidth + remainingSpace
					visibleColumns++
				}
				break
			}
		}
		
		// Render column headers with calculated widths
		var headerParts []string
		for i := 0; i < visibleColumns; i++ {
			col := dtc.Table.Columns[i]
			colHeader := col.Name
			if col.PK {
				colHeader += "*"
			}
			
			// Pad header to column width based on rune count
			padded := colHeader
			runeCount := len([]rune(padded))
			if runeCount < columnWidths[i] {
				padded += strings.Repeat(" ", columnWidths[i]-runeCount)
			}
			headerParts = append(headerParts, padded)
		}
		
		headerStr := strings.Join(headerParts, " | ")
		if visibleColumns < len(dtc.Table.Columns) {
			headerStr += " | ..."
		}
		
		s.WriteString(ui.NameStyle().Render(headerStr))
		s.WriteString("\n")
		s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
		s.WriteString("\n")
		availableHeight -= 3 // Account for column headers
	}

	// Show table data
	if len(dtc.TableData) == 0 {
		s.WriteString(ui.DetailStyle().Render("No data"))
	} else {
		linesUsed := 0
		startIdx := dtc.Viewport
		
		for i := startIdx; i < len(dtc.TableData) && linesUsed < availableHeight; i++ {
			row := dtc.TableData[i]
			
			if i > startIdx {
				s.WriteString("\n")
				linesUsed++
				if linesUsed >= availableHeight {
					break
				}
			}
			
			// Build row data with aligned columns
			var rowParts []string
			for j := 0; j < visibleColumns; j++ {
				col := dtc.Table.Columns[j]
				var valStr string
				if val, ok := row[col.Name]; ok {
					valStr = fmt.Sprintf("%v", val)
					// Sanitize the value for display
					valStr = sanitizeForDisplay(valStr)
				} else {
					valStr = "NULL"
				}
				
				// Truncate if needed to fit column width
				if len(valStr) > columnWidths[j] {
					// Use rune-aware truncation to handle multi-byte characters
					runes := []rune(valStr)
					if len(runes) > columnWidths[j]-3 {
						valStr = string(runes[:columnWidths[j]-3]) + "..."
					} else {
						valStr = valStr + "..."
					}
				}
				
				// Pad to column width based on rune count
				runeCount := len([]rune(valStr))
				if runeCount < columnWidths[j] {
					valStr += strings.Repeat(" ", columnWidths[j]-runeCount)
				}
				
				rowParts = append(rowParts, valStr)
			}
			
			rowStr := strings.Join(rowParts, " | ")
			// Add ... if we have more columns
			if visibleColumns < len(dtc.Table.Columns) {
				rowStr += " | ..."
			}
			
			s.WriteString(ui.DetailStyle().Render(rowStr))
			linesUsed++
		}
	}

	return s.String()
}

// sanitizeForDisplay cleans string values for terminal display
func sanitizeForDisplay(s string) string {
	// Remove newlines and carriage returns
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	
	// Replace non-printable characters
	var result strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) || r == ' ' {
			result.WriteRune(r)
		} else {
			// Replace non-printable with a box character
			result.WriteString("□")
		}
	}
	
	// Don't collapse spaces for binary data - it affects column width calculation
	return strings.TrimSpace(result.String())
}