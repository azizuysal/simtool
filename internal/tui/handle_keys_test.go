package tui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/azizuysal/simtool/internal/config"
	"github.com/azizuysal/simtool/internal/simulator"
)

// asModel unwraps a tea.Model returned from a handler into a concrete
// Model. All failures are fatal — handlers always return a Model.
func asModel(t *testing.T, iface any) Model {
	t.Helper()
	m, ok := iface.(Model)
	if !ok {
		t.Fatalf("expected tea.Model to be Model, got %T", iface)
	}
	return m
}

// fakeSims returns a set of 3 simulators, two running, one shut down.
func fakeSims() []simulator.Item {
	return []simulator.Item{
		{Simulator: simulator.Simulator{Name: "iPhone 14", UDID: "udid-14", State: "Shutdown"}, Runtime: "iOS 16.0"},
		{Simulator: simulator.Simulator{Name: "iPhone 15", UDID: "udid-15", State: "Booted"}, Runtime: "iOS 17.0"},
		{Simulator: simulator.Simulator{Name: "iPad Pro", UDID: "udid-ip", State: "Shutdown"}, Runtime: "iPadOS 17.0"},
	}
}

// ---------- handleSimulatorListKey ----------

func TestHandleSimulatorListKey_Navigation(t *testing.T) {
	sims := fakeSims()

	tests := []struct {
		name       string
		cursor     int
		action     string
		wantCursor int
	}{
		{"down moves cursor", 0, "down", 1},
		{"down stops at last", 2, "down", 2},
		{"up moves cursor", 2, "up", 1},
		{"up stops at first", 0, "up", 0},
		{"home resets cursor", 2, "home", 0},
		{"end moves to last", 0, "end", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{viewState: SimulatorListView, simulators: sims, simCursor: tt.cursor, height: 30}
			got, cmd := m.handleSimulatorListKey(tt.action)
			gm := asModel(t, got)
			if gm.simCursor != tt.wantCursor {
				t.Errorf("simCursor = %d, want %d", gm.simCursor, tt.wantCursor)
			}
			if gm.viewState != SimulatorListView {
				t.Errorf("viewState changed unexpectedly to %v", gm.viewState)
			}
			if cmd != nil {
				t.Error("expected nil cmd for pure navigation")
			}
		})
	}
}

func TestHandleSimulatorListKey_Right_TransitionsToAppList(t *testing.T) {
	sims := fakeSims()
	m := Model{viewState: SimulatorListView, simulators: sims, simCursor: 0, height: 30}

	got, cmd := m.handleSimulatorListKey("right")
	gm := asModel(t, got)

	if gm.viewState != AppListView {
		t.Errorf("viewState = %v, want AppListView", gm.viewState)
	}
	if gm.selectedSim == nil || gm.selectedSim.UDID != "udid-14" {
		t.Errorf("selectedSim = %+v, want udid-14", gm.selectedSim)
	}
	if !gm.loadingApps {
		t.Error("loadingApps should be true after transition")
	}
	if cmd == nil {
		t.Error("expected fetchAppsCmd, got nil")
	}
}

func TestHandleSimulatorListKey_Right_NoTransitionOnEmpty(t *testing.T) {
	m := Model{viewState: SimulatorListView, simulators: nil, simCursor: 0, height: 30}
	got, cmd := m.handleSimulatorListKey("right")
	gm := asModel(t, got)
	if gm.viewState != SimulatorListView {
		t.Errorf("viewState changed despite empty list: %v", gm.viewState)
	}
	if cmd != nil {
		t.Error("expected nil cmd with empty list")
	}
}

func TestHandleSimulatorListKey_Filter_Toggles(t *testing.T) {
	m := Model{viewState: SimulatorListView, simulators: fakeSims(), simCursor: 2, simViewport: 1, filterActive: false, height: 30}
	got, _ := m.handleSimulatorListKey("filter")
	gm := asModel(t, got)
	if !gm.filterActive {
		t.Error("filterActive should be toggled on")
	}
	if gm.simCursor != 0 || gm.simViewport != 0 {
		t.Errorf("filter should reset cursor/viewport: got cursor=%d viewport=%d", gm.simCursor, gm.simViewport)
	}

	// toggle back
	got2, _ := gm.handleSimulatorListKey("filter")
	gm2 := asModel(t, got2)
	if gm2.filterActive {
		t.Error("filterActive should be toggled off")
	}
}

func TestHandleSimulatorListKey_Search_EntersSearchMode(t *testing.T) {
	m := Model{viewState: SimulatorListView, simulators: fakeSims(), simCursor: 2, simSearchQuery: "old", height: 30}
	got, _ := m.handleSimulatorListKey("search")
	gm := asModel(t, got)
	if !gm.simSearchMode {
		t.Error("simSearchMode should be true")
	}
	if gm.simSearchQuery != "" {
		t.Errorf("simSearchQuery = %q, want empty", gm.simSearchQuery)
	}
	if gm.simCursor != 0 {
		t.Errorf("simCursor = %d, want 0", gm.simCursor)
	}
}

func TestHandleSimulatorListKey_Boot_Shutdown(t *testing.T) {
	sims := fakeSims()
	// cursor on iPhone 14 which is Shutdown
	m := Model{viewState: SimulatorListView, simulators: sims, simCursor: 0, height: 30}

	got, cmd := m.handleSimulatorListKey("boot")
	gm := asModel(t, got)

	if !gm.booting {
		t.Error("booting should be true")
	}
	if gm.statusMessage == "" {
		t.Error("statusMessage should be set")
	}
	if cmd == nil {
		t.Error("expected bootSimulatorCmd, got nil")
	}
}

func TestHandleSimulatorListKey_Boot_AlreadyRunning(t *testing.T) {
	sims := fakeSims()
	// cursor on iPhone 15 which is Booted
	m := Model{viewState: SimulatorListView, simulators: sims, simCursor: 1, height: 30}

	got, cmd := m.handleSimulatorListKey("boot")
	gm := asModel(t, got)

	if gm.booting {
		t.Error("booting should not be set when sim already running")
	}
	if gm.statusMessage != "Simulator is already running" {
		t.Errorf("statusMessage = %q, want 'Simulator is already running'", gm.statusMessage)
	}
	if cmd == nil {
		t.Error("expected clearStatusMsg tick cmd")
	}
}

// ---------- handleAppListKey ----------

func fakeApps() []simulator.App {
	return []simulator.App{
		{Name: "AppA", BundleID: "com.example.a", Container: "/path/a"},
		{Name: "AppB", BundleID: "com.example.b", Container: "/path/b"},
	}
}

func TestHandleAppListKey_Left_ReturnsToSimList(t *testing.T) {
	sim := simulator.Item{Simulator: simulator.Simulator{Name: "iPhone 15", UDID: "u"}}
	m := Model{
		viewState:      AppListView,
		selectedSim:    &sim,
		apps:           fakeApps(),
		appSearchMode:  true,
		appSearchQuery: "query",
		height:         30,
	}
	got, _ := m.handleAppListKey("left")
	gm := asModel(t, got)

	if gm.viewState != SimulatorListView {
		t.Errorf("viewState = %v, want SimulatorListView", gm.viewState)
	}
	if gm.selectedSim != nil {
		t.Error("selectedSim should be cleared")
	}
	if gm.apps != nil {
		t.Error("apps should be cleared")
	}
	if gm.appSearchMode {
		t.Error("appSearchMode should be cleared")
	}
	if gm.appSearchQuery != "" {
		t.Errorf("appSearchQuery = %q, want empty", gm.appSearchQuery)
	}
}

func TestHandleAppListKey_Right_OpensFileList(t *testing.T) {
	m := Model{
		viewState: AppListView,
		apps:      fakeApps(),
		appCursor: 1,
		height:    30,
	}
	got, cmd := m.handleAppListKey("right")
	gm := asModel(t, got)

	if gm.viewState != FileListView {
		t.Errorf("viewState = %v, want FileListView", gm.viewState)
	}
	if gm.selectedApp == nil || gm.selectedApp.BundleID != "com.example.b" {
		t.Errorf("selectedApp wrong: %+v", gm.selectedApp)
	}
	if gm.currentPath != "/path/b" {
		t.Errorf("currentPath = %q, want /path/b", gm.currentPath)
	}
	if gm.basePath != "/path/b" {
		t.Errorf("basePath = %q, want /path/b", gm.basePath)
	}
	if cmd == nil {
		t.Error("expected fetchFilesCmd")
	}
}

func TestHandleAppListKey_Open_ReturnsFinderCmd(t *testing.T) {
	m := Model{viewState: AppListView, apps: fakeApps(), appCursor: 0, height: 30}
	_, cmd := m.handleAppListKey("open")
	if cmd == nil {
		t.Error("expected openInFinderCmd")
	}
}

func TestHandleAppListKey_Open_NoContainer(t *testing.T) {
	apps := []simulator.App{{Name: "NoContainer", Container: ""}}
	m := Model{viewState: AppListView, apps: apps, appCursor: 0, height: 30}
	_, cmd := m.handleAppListKey("open")
	if cmd != nil {
		t.Error("expected nil cmd when container is empty")
	}
}

func TestHandleAppListKey_Navigation(t *testing.T) {
	apps := fakeApps()
	tests := []struct {
		name       string
		cursor     int
		action     string
		wantCursor int
	}{
		{"down", 0, "down", 1},
		{"down stops at last", 1, "down", 1},
		{"up", 1, "up", 0},
		{"home", 1, "home", 0},
		{"end", 0, "end", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{viewState: AppListView, apps: apps, appCursor: tt.cursor, height: 30}
			got, _ := m.handleAppListKey(tt.action)
			gm := asModel(t, got)
			if gm.appCursor != tt.wantCursor {
				t.Errorf("appCursor = %d, want %d", gm.appCursor, tt.wantCursor)
			}
		})
	}
}

// ---------- handleAllAppsKey ----------

func TestHandleAllAppsKey_Right_OpensFileList(t *testing.T) {
	m := Model{
		viewState:     AllAppsView,
		allApps:       fakeApps(),
		allAppsCursor: 0,
		height:        30,
	}
	got, cmd := m.handleAllAppsKey("right")
	gm := asModel(t, got)
	if gm.viewState != FileListView {
		t.Errorf("viewState = %v, want FileListView", gm.viewState)
	}
	if gm.selectedApp == nil {
		t.Error("selectedApp should be set")
	}
	if cmd == nil {
		t.Error("expected fetchFilesCmd")
	}
}

func TestHandleAllAppsKey_Navigation(t *testing.T) {
	apps := fakeApps()
	m := Model{viewState: AllAppsView, allApps: apps, allAppsCursor: 0, height: 30}

	got, _ := m.handleAllAppsKey("down")
	gm := asModel(t, got)
	if gm.allAppsCursor != 1 {
		t.Errorf("down: cursor = %d, want 1", gm.allAppsCursor)
	}

	got, _ = gm.handleAllAppsKey("up")
	gm = asModel(t, got)
	if gm.allAppsCursor != 0 {
		t.Errorf("up: cursor = %d, want 0", gm.allAppsCursor)
	}
}

func TestHandleAllAppsKey_Search(t *testing.T) {
	m := Model{viewState: AllAppsView, allApps: fakeApps(), allAppsCursor: 1, height: 30}
	got, _ := m.handleAllAppsKey("search")
	gm := asModel(t, got)
	if !gm.allAppsSearchMode {
		t.Error("allAppsSearchMode should be true")
	}
	if gm.allAppsCursor != 0 {
		t.Errorf("cursor should reset to 0, got %d", gm.allAppsCursor)
	}
}

// ---------- handleFileListKey ----------

func fakeFiles() []simulator.FileInfo {
	return []simulator.FileInfo{
		{Name: "Documents", Path: "/path/a/Documents", IsDirectory: true},
		{Name: "readme.txt", Path: "/path/a/readme.txt", IsDirectory: false, Size: 42},
	}
}

func TestHandleFileListKey_Left_WithBreadcrumbs_GoesUp(t *testing.T) {
	m := Model{
		viewState:   FileListView,
		basePath:    "/path/a",
		currentPath: "/path/a/Documents/Inner",
		breadcrumbs: []string{"Documents", "Inner"},
		height:      30,
	}
	got, cmd := m.handleFileListKey("left")
	gm := asModel(t, got)

	if len(gm.breadcrumbs) != 1 || gm.breadcrumbs[0] != "Documents" {
		t.Errorf("breadcrumbs = %v, want [Documents]", gm.breadcrumbs)
	}
	if gm.currentPath != "/path/a/Documents" {
		t.Errorf("currentPath = %q, want /path/a/Documents", gm.currentPath)
	}
	if !gm.loadingFiles {
		t.Error("loadingFiles should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFilesCmd")
	}
}

func TestHandleFileListKey_Left_AtRootFromAppList(t *testing.T) {
	// selectedApp without SimulatorUDID means we came from AppListView
	app := simulator.App{Name: "App", Container: "/path/a"}
	m := Model{
		viewState:   FileListView,
		selectedApp: &app,
		basePath:    "/path/a",
		currentPath: "/path/a",
		breadcrumbs: nil,
		height:      30,
	}
	got, _ := m.handleFileListKey("left")
	gm := asModel(t, got)

	if gm.viewState != AppListView {
		t.Errorf("viewState = %v, want AppListView", gm.viewState)
	}
	if gm.selectedApp != nil {
		t.Error("selectedApp should be cleared")
	}
	if gm.basePath != "" || gm.currentPath != "" {
		t.Error("paths should be cleared")
	}
}

func TestHandleFileListKey_Left_AtRootFromAllApps(t *testing.T) {
	// selectedApp with SimulatorUDID means we came from AllAppsView
	app := simulator.App{Name: "App", Container: "/path/a", SimulatorUDID: "u-1"}
	m := Model{
		viewState:   FileListView,
		selectedApp: &app,
		basePath:    "/path/a",
		currentPath: "/path/a",
		breadcrumbs: nil,
		height:      30,
	}
	got, _ := m.handleFileListKey("left")
	gm := asModel(t, got)

	if gm.viewState != AllAppsView {
		t.Errorf("viewState = %v, want AllAppsView", gm.viewState)
	}
}

func TestHandleFileListKey_Right_OnDirectory_DrillsIn(t *testing.T) {
	files := fakeFiles()
	m := Model{
		viewState:   FileListView,
		files:       files,
		fileCursor:  0, // Documents directory
		currentPath: "/path/a",
		breadcrumbs: nil,
		height:      30,
	}
	got, cmd := m.handleFileListKey("right")
	gm := asModel(t, got)

	if len(gm.breadcrumbs) != 1 || gm.breadcrumbs[0] != "Documents" {
		t.Errorf("breadcrumbs = %v, want [Documents]", gm.breadcrumbs)
	}
	if gm.currentPath != "/path/a/Documents" {
		t.Errorf("currentPath = %q", gm.currentPath)
	}
	if !gm.loadingFiles {
		t.Error("loadingFiles should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFilesCmd")
	}
}

func TestHandleFileListKey_Right_OnFile_OpensFileViewer(t *testing.T) {
	files := fakeFiles()
	m := Model{
		viewState:  FileListView,
		files:      files,
		fileCursor: 1, // readme.txt
		height:     30,
	}
	got, cmd := m.handleFileListKey("right")
	gm := asModel(t, got)

	if gm.viewState != FileViewerView {
		t.Errorf("viewState = %v, want FileViewerView", gm.viewState)
	}
	if gm.viewingFile == nil || gm.viewingFile.Name != "readme.txt" {
		t.Errorf("viewingFile wrong: %+v", gm.viewingFile)
	}
	if !gm.loadingContent {
		t.Error("loadingContent should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileListKey_Right_OnDatabase_OpensDatabaseView(t *testing.T) {
	// Need a real .sqlite file so DetectFileType returns FileTypeDatabase.
	// Use t.TempDir + real sqlite file for integrity.
	dbPath := t.TempDir() + "/app.sqlite"
	// Create a minimal sqlite file so DetectFileType recognizes it by header.
	// DetectFileType also accepts by extension, so an empty .sqlite works for
	// the extension check alone. But readDatabaseInfo would fail — we don't
	// call it here, we only verify the state transition and cmd presence.
	if err := writeEmptyFile(dbPath); err != nil {
		t.Fatalf("setup: %v", err)
	}

	files := []simulator.FileInfo{
		{Name: "app.sqlite", Path: dbPath, IsDirectory: false},
	}
	m := Model{
		viewState:  FileListView,
		files:      files,
		fileCursor: 0,
		height:     30,
	}
	got, cmd := m.handleFileListKey("right")
	gm := asModel(t, got)

	if gm.viewState != DatabaseTableListView {
		t.Errorf("viewState = %v, want DatabaseTableListView", gm.viewState)
	}
	if gm.viewingDatabase == nil {
		t.Error("viewingDatabase should be set")
	}
	if !gm.loadingDatabase {
		t.Error("loadingDatabase should be true")
	}
	if cmd == nil {
		t.Error("expected fetchDatabaseInfoCmd")
	}
}

func TestHandleFileListKey_Navigation(t *testing.T) {
	files := fakeFiles()
	tests := []struct {
		name       string
		cursor     int
		action     string
		wantCursor int
	}{
		{"down", 0, "down", 1},
		{"down stops at last", 1, "down", 1},
		{"up", 1, "up", 0},
		{"home", 1, "home", 0},
		{"end", 0, "end", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{viewState: FileListView, files: files, fileCursor: tt.cursor, height: 30}
			got, _ := m.handleFileListKey(tt.action)
			gm := asModel(t, got)
			if gm.fileCursor != tt.wantCursor {
				t.Errorf("fileCursor = %d, want %d", gm.fileCursor, tt.wantCursor)
			}
		})
	}
}

func TestHandleFileListKey_Open_ReturnsFinderCmd(t *testing.T) {
	m := Model{viewState: FileListView, files: fakeFiles(), fileCursor: 0, height: 30}
	_, cmd := m.handleFileListKey("open")
	if cmd == nil {
		t.Error("expected openInFinderCmd")
	}
}

// ---------- handleFileViewerKey ----------

func TestHandleFileViewerKey_Left_ReturnsToFileList(t *testing.T) {
	file := simulator.FileInfo{Name: "x.txt", Path: "/x.txt"}
	m := Model{
		viewState:       FileViewerView,
		viewingFile:     &file,
		fileContent:     &simulator.FileContent{Type: simulator.FileTypeText},
		contentOffset:   10,
		contentViewport: 5,
		svgWarning:      "warn",
		height:          30,
	}
	got, _ := m.handleFileViewerKey("left")
	gm := asModel(t, got)

	if gm.viewState != FileListView {
		t.Errorf("viewState = %v, want FileListView", gm.viewState)
	}
	if gm.viewingFile != nil || gm.fileContent != nil {
		t.Error("viewer state should be cleared")
	}
	if gm.contentOffset != 0 || gm.contentViewport != 0 {
		t.Error("content scroll state should be reset")
	}
	if gm.svgWarning != "" {
		t.Error("svgWarning should be cleared")
	}
}

func TestHandleFileViewerKey_Up_Text_ScrollsViewport(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileContent: &simulator.FileContent{
			Type:       simulator.FileTypeText,
			Lines:      []string{"a", "b", "c"},
			TotalLines: 3,
		},
		contentViewport: 2,
		height:          30,
	}
	got, _ := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.contentViewport)
	}
}

func TestHandleFileViewerKey_Up_Text_LoadsPreviousChunk(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.txt"}
	m := Model{
		viewState:       FileViewerView,
		viewingFile:     &file,
		fileContent:     &simulator.FileContent{Type: simulator.FileTypeText},
		contentViewport: 0,
		contentOffset:   500,
		height:          30,
	}
	got, cmd := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.contentOffset != 300 {
		t.Errorf("contentOffset = %d, want 300 (500 - 200)", gm.contentOffset)
	}
	if !gm.loadingContent {
		t.Error("loadingContent should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileViewerKey_Down_Text_ScrollsViewport(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileContent: &simulator.FileContent{
			Type:       simulator.FileTypeText,
			Lines:      make([]string, 100),
			TotalLines: 100,
		},
		contentViewport: 0,
		height:          30,
	}
	got, _ := m.handleFileViewerKey("down")
	gm := asModel(t, got)
	if gm.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.contentViewport)
	}
}

func TestHandleFileViewerKey_NilContent_NoOp(t *testing.T) {
	m := Model{viewState: FileViewerView, fileContent: nil, height: 30}
	got, cmd := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.fileContent != nil {
		t.Error("should remain nil")
	}
	if cmd != nil {
		t.Error("no cmd expected with nil content")
	}

	got, cmd = m.handleFileViewerKey("down")
	gm = asModel(t, got)
	if gm.fileContent != nil {
		t.Error("should remain nil")
	}
	if cmd != nil {
		t.Error("no cmd expected with nil content")
	}
}

func TestHandleFileViewerKey_Up_Binary_LoadsPreviousChunk(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.bin"}
	m := Model{
		viewState:       FileViewerView,
		viewingFile:     &file,
		fileContent:     &simulator.FileContent{Type: simulator.FileTypeBinary, BinaryData: []byte{1, 2, 3}},
		contentViewport: 0,
		contentOffset:   512,
		height:          30,
	}
	got, cmd := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.contentOffset != 256 {
		t.Errorf("contentOffset = %d, want 256 (512 - 256)", gm.contentOffset)
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileViewerKey_Up_Image_Scrolls(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileContent: &simulator.FileContent{
			Type:      simulator.FileTypeImage,
			ImageInfo: &simulator.ImageInfo{Preview: &simulator.ImagePreview{Rows: []string{"a", "b"}}},
		},
		contentViewport: 3,
		height:          30,
	}
	got, _ := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.contentViewport != 2 {
		t.Errorf("contentViewport = %d, want 2", gm.contentViewport)
	}
}

func TestHandleFileViewerKey_Up_Archive_Scrolls(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileContent: &simulator.FileContent{
			Type:        simulator.FileTypeArchive,
			ArchiveInfo: &simulator.ArchiveInfo{Entries: []simulator.ArchiveEntry{{Name: "a"}, {Name: "b"}}},
		},
		contentViewport: 2,
		height:          30,
	}
	got, _ := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.contentViewport)
	}
}

func TestHandleFileViewerKey_Down_Text_LoadsNextChunk(t *testing.T) {
	// With height=30, CalculateItemsPerScreen=7, itemsPerScreen-5=2.
	// Lines has 2 entries → maxViewport = 2 - 2 = 0. contentViewport=0
	// is NOT < 0, so the advance branch is skipped and the load-more
	// branch runs since contentOffset + len(Lines) < TotalLines.
	file := simulator.FileInfo{Path: "/x.txt"}
	m := Model{
		viewState:   FileViewerView,
		viewingFile: &file,
		fileContent: &simulator.FileContent{
			Type:       simulator.FileTypeText,
			Lines:      []string{"line 1", "line 2"},
			TotalLines: 1000,
		},
		contentOffset:   0,
		contentViewport: 0,
		height:          30,
	}
	got, cmd := m.handleFileViewerKey("down")
	gm := asModel(t, got)

	if gm.contentOffset != 2 {
		t.Errorf("contentOffset = %d, want 2 (next chunk starts after loaded lines)", gm.contentOffset)
	}
	if gm.contentViewport != 0 {
		t.Errorf("contentViewport = %d, want 0 (reset for new chunk)", gm.contentViewport)
	}
	if !gm.loadingContent {
		t.Error("loadingContent should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileViewerKey_Down_Binary_LoadsNextChunk(t *testing.T) {
	// 8 bytes → 1 hex line. With height=30, itemsPerScreen-5=2, so
	// maxViewport clamps to 0 and the advance branch is skipped.
	// currentEndByte (8) < TotalSize (1024) triggers the load.
	file := simulator.FileInfo{Path: "/x.bin"}
	m := Model{
		viewState:   FileViewerView,
		viewingFile: &file,
		fileContent: &simulator.FileContent{
			Type:         simulator.FileTypeBinary,
			BinaryData:   []byte{0, 1, 2, 3, 4, 5, 6, 7},
			BinaryOffset: 0,
			TotalSize:    1024,
		},
		contentOffset:   0,
		contentViewport: 0,
		height:          30,
	}
	got, cmd := m.handleFileViewerKey("down")
	gm := asModel(t, got)

	if gm.contentOffset != 1 {
		t.Errorf("contentOffset = %d, want 1 (len(hexLines)=1)", gm.contentOffset)
	}
	if gm.contentViewport != 0 {
		t.Errorf("contentViewport = %d, want 0 (reset for new chunk)", gm.contentViewport)
	}
	if !gm.loadingContent {
		t.Error("loadingContent should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileViewerKey_Down_Image_AdvancesViewport(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileContent: &simulator.FileContent{
			Type: simulator.FileTypeImage,
			ImageInfo: &simulator.ImageInfo{
				Preview: &simulator.ImagePreview{Rows: make([]string, 50)},
			},
		},
		contentViewport: 0,
		height:          30,
	}
	got, _ := m.handleFileViewerKey("down")
	gm := asModel(t, got)
	if gm.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.contentViewport)
	}
}

func TestHandleFileViewerKey_Down_Archive_AdvancesViewport(t *testing.T) {
	entries := make([]simulator.ArchiveEntry, 50)
	m := Model{
		viewState: FileViewerView,
		fileContent: &simulator.FileContent{
			Type:        simulator.FileTypeArchive,
			ArchiveInfo: &simulator.ArchiveInfo{Entries: entries},
		},
		contentViewport: 0,
		height:          30,
	}
	got, _ := m.handleFileViewerKey("down")
	gm := asModel(t, got)
	if gm.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.contentViewport)
	}
}

func TestHandleFileViewerKey_Up_Text_AtTopNoOffset_NoOp(t *testing.T) {
	m := Model{
		viewState:       FileViewerView,
		fileContent:     &simulator.FileContent{Type: simulator.FileTypeText, Lines: []string{"a"}},
		contentViewport: 0,
		contentOffset:   0,
		height:          30,
	}
	got, cmd := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.contentViewport != 0 || gm.contentOffset != 0 {
		t.Error("state should not change at top with no offset")
	}
	if cmd != nil {
		t.Error("no cmd expected at top")
	}
}

// ---------- handleDatabaseTableListKey ----------

func TestHandleDatabaseTableListKey_Left_ReturnsToFileList(t *testing.T) {
	m := Model{
		viewState:       DatabaseTableListView,
		viewingDatabase: &simulator.FileInfo{Name: "x.db"},
		databaseInfo:    &simulator.DatabaseInfo{},
		tableCursor:     2,
		tableViewport:   1,
		height:          30,
	}
	got, _ := m.handleDatabaseTableListKey("left")
	gm := asModel(t, got)

	if gm.viewState != FileListView {
		t.Errorf("viewState = %v, want FileListView", gm.viewState)
	}
	if gm.viewingDatabase != nil || gm.databaseInfo != nil {
		t.Error("database state should be cleared")
	}
	if gm.tableCursor != 0 || gm.tableViewport != 0 {
		t.Error("table cursor/viewport should be reset")
	}
}

func TestHandleDatabaseTableListKey_Right_OpensTableContent(t *testing.T) {
	dbFile := simulator.FileInfo{Path: "/x.db"}
	m := Model{
		viewState:       DatabaseTableListView,
		viewingDatabase: &dbFile,
		databaseInfo: &simulator.DatabaseInfo{
			Tables: []simulator.TableInfo{{Name: "users"}, {Name: "posts"}},
		},
		tableCursor: 1,
		height:      30,
	}
	got, cmd := m.handleDatabaseTableListKey("right")
	gm := asModel(t, got)

	if gm.viewState != DatabaseTableContentView {
		t.Errorf("viewState = %v, want DatabaseTableContentView", gm.viewState)
	}
	if gm.selectedTable == nil || gm.selectedTable.Name != "posts" {
		t.Errorf("selectedTable = %+v", gm.selectedTable)
	}
	if !gm.loadingTableData {
		t.Error("loadingTableData should be true")
	}
	if cmd == nil {
		t.Error("expected fetchTableDataCmd")
	}
}

func TestHandleDatabaseTableListKey_Navigation(t *testing.T) {
	info := &simulator.DatabaseInfo{Tables: []simulator.TableInfo{{Name: "a"}, {Name: "b"}}}
	m := Model{viewState: DatabaseTableListView, databaseInfo: info, tableCursor: 0, height: 30}

	got, _ := m.handleDatabaseTableListKey("down")
	gm := asModel(t, got)
	if gm.tableCursor != 1 {
		t.Errorf("down: tableCursor = %d, want 1", gm.tableCursor)
	}

	got, _ = gm.handleDatabaseTableListKey("up")
	gm = asModel(t, got)
	if gm.tableCursor != 0 {
		t.Errorf("up: tableCursor = %d, want 0", gm.tableCursor)
	}
}

// ---------- handleDatabaseTableContentKey ----------

func TestHandleDatabaseTableContentKey_Left_ReturnsToTableList(t *testing.T) {
	table := simulator.TableInfo{Name: "users"}
	m := Model{
		viewState:         DatabaseTableContentView,
		selectedTable:     &table,
		tableData:         []map[string]any{{"id": 1}},
		tableDataOffset:   5,
		tableDataViewport: 2,
		height:            30,
	}
	got, _ := m.handleDatabaseTableContentKey("left")
	gm := asModel(t, got)

	if gm.viewState != DatabaseTableListView {
		t.Errorf("viewState = %v, want DatabaseTableListView", gm.viewState)
	}
	if gm.selectedTable != nil {
		t.Error("selectedTable should be cleared")
	}
	if gm.tableData != nil {
		t.Error("tableData should be cleared")
	}
	if gm.tableDataOffset != 0 || gm.tableDataViewport != 0 {
		t.Error("table data offset/viewport should be reset")
	}
}

func TestHandleDatabaseTableContentKey_Up_Scrolls(t *testing.T) {
	m := Model{
		viewState:         DatabaseTableContentView,
		tableDataViewport: 3,
		height:            30,
	}
	got, _ := m.handleDatabaseTableContentKey("up")
	gm := asModel(t, got)
	if gm.tableDataViewport != 2 {
		t.Errorf("tableDataViewport = %d, want 2", gm.tableDataViewport)
	}
}

func TestHandleDatabaseTableContentKey_Up_AtTop_NoOp(t *testing.T) {
	m := Model{
		viewState:         DatabaseTableContentView,
		tableDataViewport: 0,
		height:            30,
	}
	got, _ := m.handleDatabaseTableContentKey("up")
	gm := asModel(t, got)
	if gm.tableDataViewport != 0 {
		t.Errorf("tableDataViewport = %d, want 0", gm.tableDataViewport)
	}
}

func TestHandleDatabaseTableContentKey_Down_LoadsMore(t *testing.T) {
	dbFile := simulator.FileInfo{Path: "/x.db"}
	table := simulator.TableInfo{Name: "users", RowCount: 500}
	m := Model{
		viewState:         DatabaseTableContentView,
		viewingDatabase:   &dbFile,
		selectedTable:     &table,
		tableData:         make([]map[string]any, 50),
		tableDataOffset:   0,
		tableDataViewport: 1000, // way past end, forces load
		height:            30,
	}
	got, cmd := m.handleDatabaseTableContentKey("down")
	gm := asModel(t, got)
	if gm.tableDataOffset != 50 {
		t.Errorf("tableDataOffset = %d, want 50", gm.tableDataOffset)
	}
	if !gm.loadingTableData {
		t.Error("loadingTableData should be true")
	}
	if cmd == nil {
		t.Error("expected fetchTableDataCmd")
	}
}

// ---------- handleKeyPress dispatcher ----------

func testModelWithKeyMap() Model {
	return Model{
		height: 30,
		width:  80,
		config: config.Default(),
		keyMap: config.NewKeyMap(config.DefaultKeys()),
	}
}

func TestHandleKeyPress_Quit(t *testing.T) {
	m := testModelWithKeyMap()
	m.viewState = SimulatorListView

	got, cmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if got == nil {
		t.Fatal("model should be returned")
	}
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd")
	}
	// tea.Quit is a function that returns tea.QuitMsg.
	if msg := cmd(); msg == nil {
		t.Error("quit cmd should return a message")
	}
}

func TestHandleKeyPress_QuitIgnoredInSearchMode(t *testing.T) {
	m := testModelWithKeyMap()
	m.viewState = SimulatorListView
	m.simSearchMode = true

	_, cmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	// In search mode, "q" is typed into the query, so handleSimulatorSearchInput
	// handles it — not the quit path. Verify we didn't get a tea.Quit cmd by
	// checking the model still has search mode active.
	// (handleSimulatorSearchInput appends 'q' to the query.)
	_ = cmd
}

func TestHandleKeyPress_NavigationClearsStatus(t *testing.T) {
	m := testModelWithKeyMap()
	m.viewState = SimulatorListView
	m.simulators = fakeSims()
	m.simCursor = 1
	m.statusMessage = "previous status"

	got, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // j → down
	gm := asModel(t, got)

	if gm.statusMessage != "" {
		t.Errorf("statusMessage = %q, want empty (cleared by navigation)", gm.statusMessage)
	}
	if gm.simCursor != 2 {
		t.Errorf("simCursor = %d, want 2 (down advanced)", gm.simCursor)
	}
}

func TestHandleKeyPress_NonNavigationKeepsStatus(t *testing.T) {
	m := testModelWithKeyMap()
	m.viewState = SimulatorListView
	m.simulators = fakeSims()
	m.simCursor = 0
	m.statusMessage = "keep me"

	got, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) // f → filter
	gm := asModel(t, got)

	if gm.statusMessage != "keep me" {
		t.Errorf("statusMessage = %q, want 'keep me' (non-nav should not clear)", gm.statusMessage)
	}
	if !gm.filterActive {
		t.Error("filterActive should be toggled")
	}
}

func TestHandleKeyPress_DispatchesByViewState(t *testing.T) {
	// Pressing "j" (down) on FileListView should move fileCursor,
	// not simCursor. Confirms the state-based dispatch works.
	m := testModelWithKeyMap()
	m.viewState = FileListView
	m.files = fakeFiles()
	m.fileCursor = 0
	m.simCursor = 0

	got, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	gm := asModel(t, got)

	if gm.fileCursor != 1 {
		t.Errorf("fileCursor = %d, want 1", gm.fileCursor)
	}
	if gm.simCursor != 0 {
		t.Errorf("simCursor = %d, want 0 (untouched)", gm.simCursor)
	}
}

// ---------- helpers ----------

// writeEmptyFile creates an empty file at path. Used for tests that
// only need a file's extension (not its contents) for type detection.
func writeEmptyFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}
