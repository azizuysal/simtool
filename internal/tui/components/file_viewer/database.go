package file_viewer

import (
	"fmt"
	"strings"

	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// renderDatabase renders database file content
func (fv *FileViewer) renderDatabase() string {
	if fv.Content.DatabaseInfo == nil {
		return ui.DetailStyle.Render("Error loading database")
	}

	var s strings.Builder
	innerWidth := fv.Width - 4 // Account for padding
	dbInfo := fv.Content.DatabaseInfo

	// Database info header
	info := fmt.Sprintf("Database file • %s • %d tables • %s",
		dbInfo.Format,
		dbInfo.TableCount,
		simulator.FormatSize(dbInfo.FileSize))
	if dbInfo.Version != "" {
		info = fmt.Sprintf("Database file • %s %s • %d tables • %s",
			dbInfo.Format, dbInfo.Version, dbInfo.TableCount, simulator.FormatSize(dbInfo.FileSize))
	}

	s.WriteString(ui.DetailStyle.Render(info))
	s.WriteString("\n")
	s.WriteString(ui.DetailStyle.Render(strings.Repeat("─", innerWidth)))
	s.WriteString("\n\n")

	// Handle errors
	if dbInfo.Error != "" {
		s.WriteString(ui.ErrorStyle.Render("Error: " + dbInfo.Error))
		return s.String()
	}

	// Render tables with scrolling
	return s.String() + fv.renderDatabaseTables(dbInfo)
}

// renderDatabaseTables renders the database tables with scrolling support
func (fv *FileViewer) renderDatabaseTables(dbInfo *simulator.DatabaseInfo) string {
	if len(dbInfo.Tables) == 0 {
		return ui.DetailStyle.Render("No tables found")
	}

	var s strings.Builder
	innerWidth := fv.Width - 4 // Account for padding
	headerLines := 4 // Info + separator + padding
	availableHeight := fv.Height - headerLines

	// Calculate which tables to show based on viewport
	startIdx := fv.ContentViewport
	linesUsed := 0

	for i := startIdx; i < len(dbInfo.Tables) && linesUsed < availableHeight; i++ {
		table := dbInfo.Tables[i]

		// Add spacing between tables (except for first table)
		if i > startIdx {
			s.WriteString("\n\n")
			linesUsed += 2
			if linesUsed >= availableHeight {
				break
			}
		}

		// Table header with icon
		tableHeader := fmt.Sprintf("🗃️  %s (%d rows)", table.Name, table.RowCount)
		s.WriteString(ui.NameStyle.Render(tableHeader))
		s.WriteString("\n")
		linesUsed++

		// Column info
		if len(table.Columns) > 0 && linesUsed < availableHeight {
			var colStrs []string
			remainingWidth := innerWidth - len("Columns: ")
			currentWidth := 0
			
			for _, col := range table.Columns {
				colStr := col.Name + " " + col.Type
				if col.PK {
					colStr += " (PK)"
				}
				if col.NotNull {
					colStr += " NOT NULL"
				}
				
				// Check if adding this column would exceed width
				separator := ", "
				if len(colStrs) == 0 {
					separator = ""
				}
				
				neededWidth := currentWidth + len(separator) + len(colStr)
				if neededWidth > remainingWidth-3 { // -3 for "..."
					colStrs = append(colStrs, "...")
					break
				}
				
				colStrs = append(colStrs, colStr)
				currentWidth = neededWidth
			}
			
			colInfo := "Columns: " + strings.Join(colStrs, ", ")
			s.WriteString(ui.DetailStyle.Render(colInfo))
			s.WriteString("\n")
			linesUsed++
		}

		// Sample data preview
		if len(table.Sample) > 0 && linesUsed < availableHeight {
			s.WriteString(ui.DetailStyle.Render("Sample data:"))
			s.WriteString("\n")
			linesUsed++

			// Show sample rows
			for j, row := range table.Sample {
				if linesUsed >= availableHeight {
					break
				}

				var values []string
				rowPrefix := fmt.Sprintf("  Row %d: ", j+1)
				remainingWidth := innerWidth - len(rowPrefix)
				currentWidth := 0
				
				for i, col := range table.Columns {
					var valStr string
					if val, ok := row[col.Name]; ok {
						valStr = fmt.Sprintf("%v", val)
						// Truncate long values first
						if len(valStr) > 20 {
							valStr = valStr[:17] + "..."
						}
					} else {
						valStr = "NULL"
					}
					
					separator := " | "
					if i == 0 {
						separator = ""
					}
					
					neededWidth := currentWidth + len(separator) + len(valStr)
					if neededWidth > remainingWidth-3 { // -3 for "..."
						if len(values) > 0 {
							values = append(values, "...")
						}
						break
					}
					
					values = append(values, valStr)
					currentWidth = neededWidth
				}

				rowStr := rowPrefix + strings.Join(values, " | ")

				s.WriteString(ui.DetailStyle.Render(rowStr))
				if j < len(table.Sample)-1 && linesUsed+1 < availableHeight {
					s.WriteString("\n")
					linesUsed++
				}
			}
			linesUsed++
		}

		// Table schema (if space available)
		if table.Schema != "" && linesUsed+2 < availableHeight {
			s.WriteString("\n")
			schemaPreview := table.Schema
			// Show only first line of schema to save space
			if idx := strings.Index(schemaPreview, "\n"); idx > 0 {
				schemaPreview = schemaPreview[:idx] + "..."
			}
			
			schemaPrefix := "Schema: "
			remainingWidth := innerWidth - len(schemaPrefix)
			if len(schemaPreview) > remainingWidth {
				schemaPreview = schemaPreview[:remainingWidth-3] + "..."
			}
			
			s.WriteString(ui.DetailStyle.Copy().Foreground(ui.DetailStyle.GetForeground()).Render(schemaPrefix + schemaPreview))
			linesUsed += 2
		}
	}

	return s.String()
}