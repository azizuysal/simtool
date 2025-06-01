package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FormatHeader creates a centered header
func FormatHeader(text string, width int) string {
	header := HeaderStyle.Render(text)
	headerWidth := lipgloss.Width(header)

	var result strings.Builder
	result.WriteString("\n") // Top margin

	// Center the header
	if width > headerWidth {
		padding := (width - headerWidth) / 2
		result.WriteString(strings.Repeat(" ", padding))
	}
	result.WriteString(header)
	result.WriteString("\n")

	return result.String()
}

// FormatFooter creates a footer aligned with the content
func FormatFooter(text string, contentWidth, totalWidth int) string {
	var result strings.Builder
	
	// Align with content
	if totalWidth > contentWidth {
		leftPadding := (totalWidth - contentWidth) / 2
		result.WriteString(strings.Repeat(" ", leftPadding))
	}
	result.WriteString(FooterStyle.Render(text))
	
	return result.String()
}

// FormatScrollInfo creates scroll indicator text
func FormatScrollInfo(viewport, itemsPerScreen, total int) string {
	endIdx := viewport + itemsPerScreen
	if endIdx > total {
		endIdx = total
	}

	if viewport > 0 && viewport+itemsPerScreen < total {
		return fmt.Sprintf(" (%d-%d of %d) ↑↓", viewport+1, endIdx, total)
	} else if viewport > 0 {
		return fmt.Sprintf(" (%d-%d of %d) ↑", viewport+1, endIdx, total)
	} else if viewport+itemsPerScreen < total {
		return fmt.Sprintf(" (%d-%d of %d) ↓", viewport+1, endIdx, total)
	}
	return ""
}

// PadLine pads a line to the specified width
func PadLine(line string, width int) string {
	currentWidth := lipgloss.Width(line)
	if currentWidth < width {
		return line + strings.Repeat(" ", width-currentWidth)
	}
	return line
}