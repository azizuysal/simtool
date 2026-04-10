package components

import (
	"strings"

	"github.com/azizuysal/simtool/internal/ui"
)

// Component is implemented by view-state components (lists, viewers)
// that produce a title, body, and footer for Layout to compose.
type Component interface {
	Render() string
	GetTitle() string
	GetFooter() string
}

// StatusProvider is an optional extension: components that also emit
// a status-line message implement this. Only FileViewer currently
// does (for SVG rendering warnings).
type StatusProvider interface {
	GetStatus() string
}

// Compile-time assertions that the list components satisfy Component.
// FileViewer lives in the file_viewer sub-package and asserts itself
// there to avoid an import cycle in the other direction.
var (
	_ Component = (*FileList)(nil)
	_ Component = (*DatabaseTableList)(nil)
	_ Component = (*DatabaseTableContent)(nil)
)

// renderHeaderPrefix returns a rendered header block followed by a
// horizontal separator, suitable for prepending to a list body. If
// header is empty, returns "". innerWidth is the body width inside
// any padding, used to size the separator rule.
func renderHeaderPrefix(header string, innerWidth int) string {
	if header == "" {
		return ""
	}
	var s strings.Builder
	s.WriteString(header)
	s.WriteString("\n\n")
	s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
	s.WriteString("\n\n")
	return s.String()
}
