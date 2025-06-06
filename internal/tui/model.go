package tui

import (
	"os"
	"os/exec"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"simtool/internal/config"
	"simtool/internal/simulator"
)

// ViewState represents the current view
type ViewState int

const (
	SimulatorListView ViewState = iota
	AppListView
	FileListView
	FileViewerView
	DatabaseTableListView
	DatabaseTableContentView
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
	loadingSimulators bool           // Whether simulators are being loaded
	filterActive  bool               // Whether to show only sims with apps
	simSearchMode bool               // Whether search is active in sim list
	simSearchQuery string            // Current search query for sim list
	
	// App list state
	selectedSim   *simulator.Item
	apps          []simulator.App
	appCursor     int
	appViewport   int
	loadingApps   bool
	appSearchMode bool               // Whether search is active in app list
	appSearchQuery string            // Current search query for app list
	
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
	
	// Database state
	viewingDatabase *simulator.FileInfo    // The database file being viewed
	databaseInfo    *simulator.DatabaseInfo // Database metadata and table list
	selectedTable   *simulator.TableInfo   // Currently selected table
	tableData       []map[string]any       // Current page of table data
	tableCursor     int                    // Cursor in table list
	tableViewport   int                    // Viewport in table list
	tableDataOffset int                    // Row offset for table content pagination
	tableDataViewport int                  // Viewport position within table content
	loadingDatabase bool                   // Whether database info is loading
	loadingTableData bool                  // Whether table data is loading
	
	// Theme state
	currentThemeMode string               // Current detected theme mode ("dark" or "light")
}

// New creates a new Model with the given fetcher
func New(fetcher simulator.Fetcher) Model {
	// Get initial theme mode
	isDark := config.DetectTerminalDarkMode()
	themeMode := "light"
	if isDark {
		themeMode = "dark"
	}
	
	return Model{
		fetcher: fetcher,
		loadingSimulators: true,
		currentThemeMode: themeMode,
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

// themeChangedMsg is sent when the terminal theme changes
type themeChangedMsg struct {
	newMode string // "dark" or "light"
}

// fetchAppsCmd fetches apps for a simulator
func (m Model) fetchAppsCmd(sim simulator.Item) tea.Cmd {
	return func() tea.Msg {
		apps, err := simulator.GetAppsForSimulator(sim.UDID, sim.IsRunning())
		return fetchAppsMsg{apps: apps, err: err}
	}
}

// checkThemeChange detects if the terminal theme has changed
// This is called periodically with the tick to enable dynamic theme switching
// when the user changes their terminal's appearance
func (m *Model) checkThemeChange() tea.Cmd {
	// Skip theme detection if explicitly overridden
	if os.Getenv("SIMTOOL_THEME_MODE") != "" {
		return nil
	}
	
	// Detect current theme mode (live, without cache)
	isDark := config.DetectTerminalDarkModeLive()
	newMode := "light"
	if isDark {
		newMode = "dark"
	}
	
	// Debug: log theme detection
	// os.WriteFile("/tmp/theme_debug.log", []byte(fmt.Sprintf("%s: current=%s detected=%s\n", time.Now().Format("15:04:05"), m.currentThemeMode, newMode)), 0644)
	
	// Check if theme has changed
	if newMode != m.currentThemeMode {
		// Theme has changed, return a command to handle it
		return func() tea.Msg {
			return themeChangedMsg{newMode: newMode}
		}
	}
	
	return nil
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
		// For images, pass the actual terminal dimensions so preview is sized correctly
		// For text files, use 500 lines per chunk
		maxLines := 500
		maxWidth := m.width - 6 // Same as contentWidth in view.go
		fileType := simulator.DetectFileType(path)
		if fileType == simulator.FileTypeImage {
			// Pass terminal height minus UI overhead
			// Account for: title (4), footer (4), border (0 - handled by contentHeight)
			maxLines = m.height - 8
			if maxLines < 20 {
				maxLines = 20
			}
		}
		content, err := simulator.ReadFileContent(path, offset, maxLines, maxWidth)
		return fetchFileContentMsg{content: content, err: err}
	}
}

// fetchDatabaseInfoMsg is sent when database info is fetched
type fetchDatabaseInfoMsg struct {
	dbInfo *simulator.DatabaseInfo
	err    error
}

// fetchDatabaseInfoCmd fetches database information
func (m Model) fetchDatabaseInfoCmd(path string) tea.Cmd {
	return func() tea.Msg {
		dbInfo, err := simulator.ReadDatabaseContent(path)
		return fetchDatabaseInfoMsg{dbInfo: dbInfo, err: err}
	}
}

// fetchTableDataMsg is sent when table data is fetched
type fetchTableDataMsg struct {
	data   []map[string]any
	offset int
	err    error
}

// fetchTableDataCmd fetches table data with pagination
func (m Model) fetchTableDataCmd(dbPath, tableName string, offset, limit int) tea.Cmd {
	return func() tea.Msg {
		data, err := simulator.ReadTableData(dbPath, tableName, offset, limit)
		return fetchTableDataMsg{data: data, offset: offset, err: err}
	}
}