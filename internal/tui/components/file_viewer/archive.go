package file_viewer

import (
	"fmt"
	"sort"
	"strings"

	"simtool/internal/simulator"
	"simtool/internal/ui"
)

// treeNode represents a node in the file tree
type treeNode struct {
	name     string
	isDir    bool
	children map[string]*treeNode
}

// renderArchive renders archive file content as a tree structure
func (fv *FileViewer) renderArchive() string {
	if fv.Content.ArchiveInfo == nil {
		return ui.DetailStyle().Render("Error loading archive")
	}

	var s strings.Builder
	innerWidth := fv.Width - 4 // Account for padding
	archInfo := fv.Content.ArchiveInfo

	// Archive info header
	var infoStr string
	if archInfo.TotalSize > 0 {
		ratio := float64(archInfo.CompressedSize) / float64(archInfo.TotalSize) * 100
		infoStr = fmt.Sprintf("%s Archive • %d files, %d folders • %.1f%% compression",
			archInfo.Format, archInfo.FileCount, archInfo.FolderCount, ratio)
	} else {
		infoStr = fmt.Sprintf("%s Archive • %d files, %d folders",
			archInfo.Format, archInfo.FileCount, archInfo.FolderCount)
	}

	s.WriteString(ui.DetailStyle().Render(infoStr))
	s.WriteString("\n")
	s.WriteString(ui.DetailStyle().Render(strings.Repeat("─", innerWidth)))
	s.WriteString("\n\n")

	// Build and render tree
	tree := buildTreeFromPaths(archInfo.Entries)
	var treeLines []string
	
	// Render root's children
	childNames := make([]string, 0, len(tree.children))
	for name := range tree.children {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)

	for i, childName := range childNames {
		child := tree.children[childName]
		renderTree(child, "", i == len(childNames)-1, &treeLines)
	}

	// Calculate visible range
	headerLines := 4 // Info + separator + padding
	availableLines := fv.Height - headerLines
	
	startIdx := fv.ContentViewport
	endIdx := startIdx + availableLines
	if endIdx > len(treeLines) {
		endIdx = len(treeLines)
	}

	// Display visible tree lines
	linesWritten := 0
	for i := startIdx; i < endIdx && i < len(treeLines); i++ {
		if i > startIdx {
			s.WriteString("\n")
		}
		s.WriteString(treeLines[i])
		linesWritten++
	}

	// Don't pad - ContentBox will handle filling the space

	return s.String()
}

// buildTreeFromPaths builds a tree structure from flat paths
func buildTreeFromPaths(entries []simulator.ArchiveEntry) *treeNode {
	root := &treeNode{
		name:     "",
		isDir:    true,
		children: make(map[string]*treeNode),
	}

	for _, entry := range entries {
		parts := strings.Split(entry.Name, "/")
		current := root

		for i, part := range parts {
			if part == "" {
				continue
			}

			if _, exists := current.children[part]; !exists {
				isDir := i < len(parts)-1 || entry.IsDir
				current.children[part] = &treeNode{
					name:     part,
					isDir:    isDir,
					children: make(map[string]*treeNode),
				}
			}
			current = current.children[part]
		}
	}

	return root
}

// renderTree renders a tree node with proper box drawing characters
func renderTree(node *treeNode, prefix string, isLast bool, lines *[]string) {
	if node.name != "" {
		var line string
		if isLast {
			line = prefix + "└── "
		} else {
			line = prefix + "├── "
		}

		name := node.name
		if node.isDir {
			name = name + "/"
		}
		*lines = append(*lines, line+name)
	}

	// Sort children for consistent output
	childNames := make([]string, 0, len(node.children))
	for name := range node.children {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)

	for i, childName := range childNames {
		child := node.children[childName]
		childPrefix := prefix
		if node.name != "" {
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
		}
		renderTree(child, childPrefix, i == len(childNames)-1, lines)
	}
}