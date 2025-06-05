package tui

import (
	"os/exec"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"simtool/internal/simulator"
)

// ViewState represents the current view
type ViewState int

const (
	SimulatorListView ViewState = iota
	AppListView
	FileListView
	FileViewerView
)

// Model represents the application state
type Model struct {
	// Common state
	viewState     ViewState
	err           error
	height        int
	width         int
	statusMessage string
	fetcher       simulator.Fetcher
	
	// Simulator list state
	simulators    []simulator.Item
	simCursor     int
	simViewport   int
	booting       bool
	
	// App list state
	selectedSim   *simulator.Item
	apps          []simulator.App
	appCursor     int
	appViewport   int
	loadingApps   bool
	
	// File list state
	selectedApp   *simulator.App
	files         []simulator.FileInfo
	fileCursor    int
	fileViewport  int
	loadingFiles  bool
	currentPath   string
	basePath      string              // The app's container path
	breadcrumbs   []string            // Path components from base to current
	cursorMemory  map[string]int      // Remember cursor position for each path
	viewportMemory map[string]int     // Remember viewport position for each path
	
	// File viewer state
	viewingFile   *simulator.FileInfo
	fileContent   *simulator.FileContent
	contentOffset int                 // Line offset for text files, byte offset for binary
	contentViewport int               // Viewport position within file content
	loadingContent bool
	svgWarning    string              // Warning message for SVG files with unsupported features
}

// New creates a new Model with the given fetcher
func New(fetcher simulator.Fetcher) Model {
	return Model{
		fetcher: fetcher,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchSimulatorsCmd(m.fetcher),
		tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

// fetchSimulatorsMsg is sent when simulators are fetched
type fetchSimulatorsMsg struct {
	simulators []simulator.Item
	err        error
}

// fetchSimulatorsCmd fetches simulators asynchronously
func fetchSimulatorsCmd(fetcher simulator.Fetcher) tea.Cmd {
	return func() tea.Msg {
		sims, err := fetcher.Fetch()
		return fetchSimulatorsMsg{simulators: sims, err: err}
	}
}

// bootSimulatorMsg is sent when a simulator boot is attempted
type bootSimulatorMsg struct {
	udid string
	err  error
}

// bootSimulatorCmd boots a simulator asynchronously
func (m Model) bootSimulatorCmd(udid string) tea.Cmd {
	return func() tea.Msg {
		err := m.fetcher.Boot(udid)
		return bootSimulatorMsg{udid: udid, err: err}
	}
}

// fetchAppsMsg is sent when apps are fetched
type fetchAppsMsg struct {
	apps []simulator.App
	err  error
}

// tickMsg is sent periodically to refresh simulator status
type tickMsg time.Time

// fetchAppsCmd fetches apps for a simulator
func (m Model) fetchAppsCmd(sim simulator.Item) tea.Cmd {
	return func() tea.Msg {
		apps, err := simulator.GetAppsForSimulator(sim.UDID, sim.IsRunning())
		return fetchAppsMsg{apps: apps, err: err}
	}
}

// fetchFilesMsg is sent when files are fetched
type fetchFilesMsg struct {
	files []simulator.FileInfo
	err   error
}

// fetchFilesCmd fetches files for an app container
func (m Model) fetchFilesCmd(containerPath string) tea.Cmd {
	return func() tea.Msg {
		files, err := simulator.GetFilesForContainer(containerPath)
		return fetchFilesMsg{files: files, err: err}
	}
}

// openInFinderMsg is sent when attempting to open in Finder
type openInFinderMsg struct {
	err error
}

// openInFinderCmd opens a path in Finder
func (m Model) openInFinderCmd(path string) tea.Cmd {
	return func() tea.Msg {
		// Remove file:// prefix if present
		cleanPath := path
		if len(path) > 7 && path[:7] == "file://" {
			cleanPath = path[7:]
		}
		
		// Use open command to reveal in Finder
		cmd := exec.Command("open", "-R", cleanPath)
		err := cmd.Run()
		return openInFinderMsg{err: err}
	}
}

// fetchFileContentMsg is sent when file content is fetched
type fetchFileContentMsg struct {
	content *simulator.FileContent
	err     error
}

// fetchFileContentCmd fetches the content of a file for viewing
func (m Model) fetchFileContentCmd(path string, offset int) tea.Cmd {
	return func() tea.Msg {
		// For images, pass the actual terminal height so preview is sized correctly
		// For text files, use 500 lines per chunk
		maxLines := 500
		fileType := simulator.DetectFileType(path)
		if fileType == simulator.FileTypeImage {
			// Pass terminal height minus UI overhead
			// Account for: header (3), metadata box (8), spacing (4), preview box borders (2), footer (2)
			maxLines = m.height - 19
			if maxLines < 20 {
				maxLines = 20
			}
		}
		content, err := simulator.ReadFileContent(path, offset, maxLines)
		return fetchFileContentMsg{content: content, err: err}
	}
}