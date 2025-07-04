package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/azizuysal/simtool/internal/ui"
)

// Layout represents the main TUI layout structure
type Layout struct {
	Width  int
	Height int
}

// NewLayout creates a new layout with the given dimensions
func NewLayout(width, height int) *Layout {
	return &Layout{
		Width:  width,
		Height: height,
	}
}

// Render renders the complete layout with title, content, and footer
func (l *Layout) Render(title, content, footer, status string) string {
	var s strings.Builder

	// Calculate heights more precisely
	titleSection := l.renderTitle(title)
	footerSection := l.renderFooter(footer, status)
	
	// Use lipgloss to measure actual rendered height
	titleHeight := lipgloss.Height(titleSection)
	footerHeight := lipgloss.Height(footerSection)
	
	// Calculate remaining height for content
	contentHeight := l.Height - titleHeight - footerHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Render sections
	s.WriteString(titleSection)
	s.WriteString(l.renderContent(content, contentHeight))
	s.WriteString(footerSection)

	return s.String()
}

// renderTitle renders the centered title with padding
func (l *Layout) renderTitle(title string) string {
	var s strings.Builder
	
	// Top padding line
	s.WriteString("\n")
	
	// Use the theme-based header style
	headerStyle := ui.HeaderStyle().MarginBottom(0)
	
	header := headerStyle.Render(title)
	headerWidth := lipgloss.Width(header)
	
	// Center the title
	if l.Width > headerWidth {
		padding := (l.Width - headerWidth) / 2
		s.WriteString(strings.Repeat(" ", padding))
	}
	s.WriteString(header)
	s.WriteString("\n")
	
	// Bottom padding line
	s.WriteString("\n")
	
	return s.String()
}

// renderContent renders the content box with rounded corners and padding
func (l *Layout) renderContent(content string, height int) string {
	// Calculate content box width
	contentWidth := l.Width - 6 // Leave some margin on sides
	if contentWidth < 50 {
		contentWidth = 50
	}

	// Don't set explicit height on BorderStyle, let content determine it
	// But ensure content fills the available space
	contentLines := strings.Split(content, "\n")
	currentLines := len(contentLines)
	
	// Pad content to fill available height (accounting for border)
	targetLines := height - 2 // -2 for top and bottom border
	if currentLines < targetLines && targetLines > 0 {
		for i := currentLines; i < targetLines; i++ {
			content += "\n"
		}
	}

	// Apply border with rounded corners
	borderedContent := ui.BorderStyle().
		Width(contentWidth).
		Render(content)

	// Center the content box
	return l.centerContent(borderedContent)
}

// renderFooter renders the footer with optional status message
func (l *Layout) renderFooter(footer, status string) string {
	var s strings.Builder

	// Empty line for spacing
	s.WriteString("\n")

	// Status line or another empty line
	if status != "" {
		// Center the status message
		if l.Width > lipgloss.Width(status) {
			leftPadding := (l.Width - lipgloss.Width(status)) / 2
			s.WriteString(strings.Repeat(" ", leftPadding))
		}
		s.WriteString(status)
		s.WriteString("\n")
	} else {
		// Empty line when no status
		s.WriteString("\n")
	}

	// Footer key legend
	styledFooter := ui.FooterStyle().Render(footer)
	if l.Width > lipgloss.Width(styledFooter) {
		leftPadding := (l.Width - lipgloss.Width(styledFooter)) / 2
		s.WriteString(strings.Repeat(" ", leftPadding))
	}
	s.WriteString(styledFooter)

	// Bottom padding line
	s.WriteString("\n")

	return s.String()
}

// centerContent centers content horizontally
func (l *Layout) centerContent(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Find the actual content width (might vary per line due to ANSI codes)
	maxWidth := 0
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > maxWidth {
			maxWidth = w
		}
	}

	if l.Width <= maxWidth {
		return content
	}

	var result strings.Builder
	leftPadding := (l.Width - maxWidth) / 2
	paddingStr := strings.Repeat(" ", leftPadding)

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(paddingStr)
		result.WriteString(line)
	}

	return result.String()
}

// ContentBox represents a content area with optional header
type ContentBox struct {
	Width  int
	Height int
}

// NewContentBox creates a new content box
func NewContentBox(width, height int) *ContentBox {
	return &ContentBox{
		Width:  width,
		Height: height,
	}
}

// Render renders content with optional header section
func (cb *ContentBox) Render(header, content string, hasHeader bool) string {
	innerWidth := cb.Width - 4  // Account for border padding
	innerHeight := cb.Height - 2 // Account for border padding

	var s strings.Builder

	if hasHeader && header != "" {
		// Render header
		s.WriteString(header)
		s.WriteString("\n\n")
		
		// Separator line
		s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
		s.WriteString("\n\n")

		// Adjust height for content
		headerLines := strings.Count(header, "\n") + 4 // header + separators
		contentHeight := innerHeight - headerLines
		
		// Ensure content fills remaining space
		contentLines := strings.Split(content, "\n")
		for i := 0; i < contentHeight && i < len(contentLines); i++ {
			if i > 0 {
				s.WriteString("\n")
			}
			s.WriteString(contentLines[i])
		}
		
		// Pad remaining space
		currentLines := len(contentLines)
		if currentLines < contentHeight {
			for i := currentLines; i < contentHeight; i++ {
				s.WriteString("\n")
			}
		}
	} else {
		// No header, use full height for content
		contentLines := strings.Split(content, "\n")
		for i := 0; i < innerHeight && i < len(contentLines); i++ {
			if i > 0 {
				s.WriteString("\n")
			}
			s.WriteString(contentLines[i])
		}
		
		// Pad remaining space
		currentLines := len(contentLines)
		if currentLines < innerHeight {
			for i := currentLines; i < innerHeight; i++ {
				s.WriteString("\n")
			}
		}
	}

	return s.String()
}