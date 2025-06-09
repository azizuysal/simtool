package components

import (
	"fmt"
	"strings"

	"github.com/azizuysal/simtool/internal/config"
	"github.com/azizuysal/simtool/internal/simulator"
	"github.com/azizuysal/simtool/internal/ui"
)

// AllAppsListView renders the combined apps list view
func AllAppsListView(
	allApps []simulator.App,
	cursor int,
	viewport int,
	width, height int,
	searchMode bool,
	searchQuery string,
	loading bool,
	err error,
	keys *config.KeysConfig,
) string {
	contentHeight := height - 8 // Account for title, borders, and footer
	
	// Handle loading state
	if loading {
		content := "" // Empty content during loading
		layout := NewLayout(width, height)
		return layout.Render(
			"All Apps",
			content,
			"Press q to quit",
			ui.LoadingStyle().Render("Loading all apps..."),
		)
	}
	
	// Handle error state
	if err != nil {
		content := fmt.Sprintf("Error loading apps: %v", err)
		layout := NewLayout(width, height)
		return layout.Render(
			"All Apps",
			content,
			"Press q to quit",
			"",
		)
	}
	
	// Filter apps based on search
	filteredApps := filterAllApps(allApps, searchQuery)
	
	// Build title with count
	title := fmt.Sprintf("All Apps (%d", len(filteredApps))
	if searchQuery != "" {
		title += fmt.Sprintf(" of %d)", len(allApps))
	} else {
		title += ")"
	}
	
	// Build status line
	status := ""
	if searchMode {
		searchStatus := fmt.Sprintf("Search: %s", searchQuery)
		if searchQuery == "" {
			searchStatus = "Search: (type to filter)"
		}
		status = ui.SearchStyle().Render(searchStatus)
	}
	
	// Build content
	var content string
	
	// Calculate items per screen (used for footer scroll info)
	itemsPerScreen := (contentHeight - 2) / 3 // Each app takes 2 lines + 1 blank
	if itemsPerScreen < 1 {
		itemsPerScreen = 1
	}
	
	if len(filteredApps) == 0 {
		if searchQuery != "" {
			content = ui.DetailStyle().Render("No apps match your search")
		} else {
			content = ui.DetailStyle().Render("No apps installed on any simulator")
		}
	} else {
		// Create content box for proper rendering
		contentBox := NewContentBox(width-6, contentHeight)
		
		// Adjust cursor bounds
		if cursor >= len(filteredApps) {
			cursor = len(filteredApps) - 1
		}
		if cursor < 0 {
			cursor = 0
		}
		
		startIdx := viewport
		endIdx := startIdx + itemsPerScreen
		if endIdx > len(filteredApps) {
			endIdx = len(filteredApps)
		}
		
		// Build app list content
		var listContent strings.Builder
		innerWidth := width - 10 // Account for padding and borders
		
		for i := startIdx; i < endIdx; i++ {
			app := filteredApps[i]
			
			// Format app details similar to regular app list
			sizeText := simulator.FormatSize(app.Size)
			modTimeText := simulator.FormatModTime(app.ModTime)
			detailText := fmt.Sprintf("%s • v%s • %s • %s", 
				app.BundleID, app.Version, sizeText, app.SimulatorName)
			if modTimeText != "" {
				detailText = fmt.Sprintf("%s • %s", detailText, modTimeText)
			}
			
			if i == cursor {
				// Selected item
				line1 := fmt.Sprintf("▶ %s", app.Name)
				line2 := fmt.Sprintf("  %s", detailText)
				
				// Pad to full width
				line1 = ui.PadLine(line1, innerWidth)
				line2 = ui.PadLine(line2, innerWidth)
				
				listContent.WriteString(ui.SelectedStyle().Render(line1))
				listContent.WriteString("\n")
				listContent.WriteString(ui.SelectedStyle().Render(line2))
			} else {
				// Non-selected item
				listContent.WriteString(ui.ListItemStyle().Copy().Inherit(ui.NameStyle()).Render(app.Name))
				listContent.WriteString("\n")
				listContent.WriteString(ui.ListItemStyle().Copy().Inherit(ui.DetailStyle()).Render(detailText))
			}
			
			if i < endIdx-1 {
				listContent.WriteString("\n\n")
			}
		}
		
		// Render in content box
		content = contentBox.Render("", listContent.String(), false)
	}
	
	// Build footer
	footer := buildAllAppsFooter(searchMode, len(filteredApps), keys, viewport, itemsPerScreen)
	
	layout := NewLayout(width, height)
	return layout.Render(
		title,
		content,
		footer,
		status,
	)
}

// filterAllApps filters apps based on search query
func filterAllApps(apps []simulator.App, query string) []simulator.App {
	if query == "" {
		return apps
	}
	
	query = strings.ToLower(query)
	var filtered []simulator.App
	
	for _, app := range apps {
		if strings.Contains(strings.ToLower(app.Name), query) ||
			strings.Contains(strings.ToLower(app.BundleID), query) ||
			strings.Contains(strings.ToLower(app.Version), query) ||
			strings.Contains(strings.ToLower(app.SimulatorName), query) {
			filtered = append(filtered, app)
		}
	}
	
	return filtered
}

// buildAllAppsFooter builds the footer for all apps view
func buildAllAppsFooter(searchMode bool, appCount int, keys *config.KeysConfig, viewport int, itemsPerScreen int) string {
	if keys == nil {
		return "↑/↓ navigate • enter select • / search • q quit"
	}
	
	var parts []string
	
	if searchMode {
		// Show search mode shortcuts
		parts = append(parts, "Type to search")
		if esc := keys.FormatKeyAction("escape", "cancel"); esc != "" {
			parts = append(parts, esc)
		}
		if up := keys.FormatKeyAction("up", "prev"); up != "" {
			if down := keys.FormatKeyAction("down", "next"); down != "" {
				parts = append(parts, config.FormatKeys(keys.Up)+"/"+config.FormatKeys(keys.Down)+": navigate")
			}
		}
		if right := keys.FormatKeyAction("right", "select"); right != "" {
			enter := config.FormatKeys(keys.Enter)
			if enter != "" {
				parts = append(parts, config.FormatKeys(keys.Right)+"/"+enter+": select")
			} else {
				parts = append(parts, right)
			}
		}
	} else {
		if up := keys.FormatKeyAction("up", "up"); up != "" {
			parts = append(parts, up)
		}
		if down := keys.FormatKeyAction("down", "down"); down != "" {
			parts = append(parts, down)
		}
		if right := keys.FormatKeyAction("right", "files"); right != "" {
			parts = append(parts, right)
		}
		if open := keys.FormatKeyAction("open", "open in Finder"); open != "" {
			parts = append(parts, open)
		}
		if search := keys.FormatKeyAction("search", "search"); search != "" {
			parts = append(parts, search)
		}
		if quit := keys.FormatKeyAction("quit", "quit"); quit != "" {
			parts = append(parts, quit)
		}
	}
	
	footer := strings.Join(parts, " • ")
	
	// Add scroll info
	scrollInfo := ui.FormatScrollInfo(viewport, itemsPerScreen, appCount)
	return footer + scrollInfo
}

// truncateStr truncates a string to a maximum length
func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return "..."
	}
	return s[:max-3] + "..."
}