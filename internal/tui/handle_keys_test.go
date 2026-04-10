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
			m := Model{
				viewState: SimulatorListView,
				simList:   simListState{simulators: sims, cursor: tt.cursor},
				height:    30,
			}
			got, cmd := m.handleSimulatorListKey(tt.action)
			gm := asModel(t, got)
			if gm.simList.cursor != tt.wantCursor {
				t.Errorf("simList.cursor = %d, want %d", gm.simList.cursor, tt.wantCursor)
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
	m := Model{
		viewState: SimulatorListView,
		simList:   simListState{simulators: sims, cursor: 0},
		height:    30,
	}

	got, cmd := m.handleSimulatorListKey("right")
	gm := asModel(t, got)

	if gm.viewState != AppListView {
		t.Errorf("viewState = %v, want AppListView", gm.viewState)
	}
	if gm.appList.selectedSim == nil || gm.appList.selectedSim.UDID != "udid-14" {
		t.Errorf("appList.selectedSim = %+v, want udid-14", gm.appList.selectedSim)
	}
	if !gm.appList.loading {
		t.Error("appList.loading should be true after transition")
	}
	if cmd == nil {
		t.Error("expected fetchAppsCmd, got nil")
	}
}

func TestHandleSimulatorListKey_Right_NoTransitionOnEmpty(t *testing.T) {
	m := Model{
		viewState: SimulatorListView,
		simList:   simListState{simulators: nil, cursor: 0},
		height:    30,
	}
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
	m := Model{
		viewState: SimulatorListView,
		simList: simListState{
			simulators: fakeSims(),
			cursor:     2,
			viewport:   1,
		},
		height: 30,
	}
	got, _ := m.handleSimulatorListKey("filter")
	gm := asModel(t, got)
	if !gm.simList.filterActive {
		t.Error("simList.filterActive should be toggled on")
	}
	if gm.simList.cursor != 0 || gm.simList.viewport != 0 {
		t.Errorf("filter should reset cursor/viewport: got cursor=%d viewport=%d", gm.simList.cursor, gm.simList.viewport)
	}

	// toggle back
	got2, _ := gm.handleSimulatorListKey("filter")
	gm2 := asModel(t, got2)
	if gm2.simList.filterActive {
		t.Error("simList.filterActive should be toggled off")
	}
}

func TestHandleSimulatorListKey_Search_EntersSearchMode(t *testing.T) {
	m := Model{
		viewState: SimulatorListView,
		simList: simListState{
			simulators:  fakeSims(),
			cursor:      2,
			searchQuery: "old",
		},
		height: 30,
	}
	got, _ := m.handleSimulatorListKey("search")
	gm := asModel(t, got)
	if !gm.simList.searchMode {
		t.Error("simList.searchMode should be true")
	}
	if gm.simList.searchQuery != "" {
		t.Errorf("simList.searchQuery = %q, want empty", gm.simList.searchQuery)
	}
	if gm.simList.cursor != 0 {
		t.Errorf("simList.cursor = %d, want 0", gm.simList.cursor)
	}
}

func TestHandleSimulatorListKey_Boot_Shutdown(t *testing.T) {
	sims := fakeSims()
	// cursor on iPhone 14 which is Shutdown
	m := Model{
		viewState: SimulatorListView,
		simList:   simListState{simulators: sims, cursor: 0},
		height:    30,
	}

	got, cmd := m.handleSimulatorListKey("boot")
	gm := asModel(t, got)

	if !gm.simList.booting {
		t.Error("simList.booting should be true")
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
	m := Model{
		viewState: SimulatorListView,
		simList:   simListState{simulators: sims, cursor: 1},
		height:    30,
	}

	got, cmd := m.handleSimulatorListKey("boot")
	gm := asModel(t, got)

	if gm.simList.booting {
		t.Error("simList.booting should not be set when sim already running")
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
		viewState: AppListView,
		appList: appListState{
			selectedSim: &sim,
			apps:        fakeApps(),
			searchMode:  true,
			searchQuery: "query",
		},
		height: 30,
	}
	got, _ := m.handleAppListKey("left")
	gm := asModel(t, got)

	if gm.viewState != SimulatorListView {
		t.Errorf("viewState = %v, want SimulatorListView", gm.viewState)
	}
	if gm.appList.selectedSim != nil {
		t.Error("appList.selectedSim should be cleared")
	}
	if gm.appList.apps != nil {
		t.Error("appList.apps should be cleared")
	}
	if gm.appList.searchMode {
		t.Error("appList.searchMode should be cleared")
	}
	if gm.appList.searchQuery != "" {
		t.Errorf("appList.searchQuery = %q, want empty", gm.appList.searchQuery)
	}
}

func TestHandleAppListKey_Right_OpensFileList(t *testing.T) {
	m := Model{
		viewState: AppListView,
		appList:   appListState{apps: fakeApps(), cursor: 1},
		height:    30,
	}
	got, cmd := m.handleAppListKey("right")
	gm := asModel(t, got)

	if gm.viewState != FileListView {
		t.Errorf("viewState = %v, want FileListView", gm.viewState)
	}
	if gm.fileList.selectedApp == nil || gm.fileList.selectedApp.BundleID != "com.example.b" {
		t.Errorf("fileList.selectedApp wrong: %+v", gm.fileList.selectedApp)
	}
	if gm.fileList.currentPath != "/path/b" {
		t.Errorf("fileList.currentPath = %q, want /path/b", gm.fileList.currentPath)
	}
	if gm.fileList.basePath != "/path/b" {
		t.Errorf("fileList.basePath = %q, want /path/b", gm.fileList.basePath)
	}
	if cmd == nil {
		t.Error("expected fetchFilesCmd")
	}
}

func TestHandleAppListKey_Open_ReturnsFinderCmd(t *testing.T) {
	m := Model{
		viewState: AppListView,
		appList:   appListState{apps: fakeApps(), cursor: 0},
		height:    30,
	}
	_, cmd := m.handleAppListKey("open")
	if cmd == nil {
		t.Error("expected openInFinderCmd")
	}
}

func TestHandleAppListKey_Open_NoContainer(t *testing.T) {
	apps := []simulator.App{{Name: "NoContainer", Container: ""}}
	m := Model{
		viewState: AppListView,
		appList:   appListState{apps: apps, cursor: 0},
		height:    30,
	}
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
			m := Model{
				viewState: AppListView,
				appList:   appListState{apps: apps, cursor: tt.cursor},
				height:    30,
			}
			got, _ := m.handleAppListKey(tt.action)
			gm := asModel(t, got)
			if gm.appList.cursor != tt.wantCursor {
				t.Errorf("appList.cursor = %d, want %d", gm.appList.cursor, tt.wantCursor)
			}
		})
	}
}

// ---------- handleAllAppsKey ----------

func TestHandleAllAppsKey_Right_OpensFileList(t *testing.T) {
	m := Model{
		viewState: AllAppsView,
		allApps:   allAppsState{apps: fakeApps(), cursor: 0},
		height:    30,
	}
	got, cmd := m.handleAllAppsKey("right")
	gm := asModel(t, got)
	if gm.viewState != FileListView {
		t.Errorf("viewState = %v, want FileListView", gm.viewState)
	}
	if gm.fileList.selectedApp == nil {
		t.Error("fileList.selectedApp should be set")
	}
	if cmd == nil {
		t.Error("expected fetchFilesCmd")
	}
}

func TestHandleAllAppsKey_Navigation(t *testing.T) {
	apps := fakeApps()
	m := Model{
		viewState: AllAppsView,
		allApps:   allAppsState{apps: apps, cursor: 0},
		height:    30,
	}

	got, _ := m.handleAllAppsKey("down")
	gm := asModel(t, got)
	if gm.allApps.cursor != 1 {
		t.Errorf("down: cursor = %d, want 1", gm.allApps.cursor)
	}

	got, _ = gm.handleAllAppsKey("up")
	gm = asModel(t, got)
	if gm.allApps.cursor != 0 {
		t.Errorf("up: cursor = %d, want 0", gm.allApps.cursor)
	}
}

func TestHandleAllAppsKey_Search(t *testing.T) {
	m := Model{
		viewState: AllAppsView,
		allApps:   allAppsState{apps: fakeApps(), cursor: 1},
		height:    30,
	}
	got, _ := m.handleAllAppsKey("search")
	gm := asModel(t, got)
	if !gm.allApps.searchMode {
		t.Error("allApps.searchMode should be true")
	}
	if gm.allApps.cursor != 0 {
		t.Errorf("cursor should reset to 0, got %d", gm.allApps.cursor)
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
		viewState: FileListView,
		fileList: fileListState{
			basePath:    "/path/a",
			currentPath: "/path/a/Documents/Inner",
			breadcrumbs: []string{"Documents", "Inner"},
		},
		height: 30,
	}
	got, cmd := m.handleFileListKey("left")
	gm := asModel(t, got)

	if len(gm.fileList.breadcrumbs) != 1 || gm.fileList.breadcrumbs[0] != "Documents" {
		t.Errorf("fileList.breadcrumbs = %v, want [Documents]", gm.fileList.breadcrumbs)
	}
	if gm.fileList.currentPath != "/path/a/Documents" {
		t.Errorf("fileList.currentPath = %q, want /path/a/Documents", gm.fileList.currentPath)
	}
	if !gm.fileList.loading {
		t.Error("fileList.loading should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFilesCmd")
	}
}

func TestHandleFileListKey_Left_AtRootFromAppList(t *testing.T) {
	// selectedApp without SimulatorUDID means we came from AppListView
	app := simulator.App{Name: "App", Container: "/path/a"}
	m := Model{
		viewState: FileListView,
		fileList: fileListState{
			selectedApp: &app,
			basePath:    "/path/a",
			currentPath: "/path/a",
			breadcrumbs: nil,
		},
		height: 30,
	}
	got, _ := m.handleFileListKey("left")
	gm := asModel(t, got)

	if gm.viewState != AppListView {
		t.Errorf("viewState = %v, want AppListView", gm.viewState)
	}
	if gm.fileList.selectedApp != nil {
		t.Error("fileList.selectedApp should be cleared")
	}
	if gm.fileList.basePath != "" || gm.fileList.currentPath != "" {
		t.Error("paths should be cleared")
	}
}

func TestHandleFileListKey_Left_AtRootFromAllApps(t *testing.T) {
	// selectedApp with SimulatorUDID means we came from AllAppsView
	app := simulator.App{Name: "App", Container: "/path/a", SimulatorUDID: "u-1"}
	m := Model{
		viewState: FileListView,
		fileList: fileListState{
			selectedApp: &app,
			basePath:    "/path/a",
			currentPath: "/path/a",
			breadcrumbs: nil,
		},
		height: 30,
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
		viewState: FileListView,
		fileList: fileListState{
			files:       files,
			cursor:      0, // Documents directory
			currentPath: "/path/a",
			breadcrumbs: nil,
		},
		height: 30,
	}
	got, cmd := m.handleFileListKey("right")
	gm := asModel(t, got)

	if len(gm.fileList.breadcrumbs) != 1 || gm.fileList.breadcrumbs[0] != "Documents" {
		t.Errorf("fileList.breadcrumbs = %v, want [Documents]", gm.fileList.breadcrumbs)
	}
	if gm.fileList.currentPath != "/path/a/Documents" {
		t.Errorf("fileList.currentPath = %q", gm.fileList.currentPath)
	}
	if !gm.fileList.loading {
		t.Error("fileList.loading should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFilesCmd")
	}
}

func TestHandleFileListKey_Right_OnFile_OpensFileViewer(t *testing.T) {
	files := fakeFiles()
	m := Model{
		viewState: FileListView,
		fileList:  fileListState{files: files, cursor: 1}, // readme.txt
		height:    30,
	}
	got, cmd := m.handleFileListKey("right")
	gm := asModel(t, got)

	if gm.viewState != FileViewerView {
		t.Errorf("viewState = %v, want FileViewerView", gm.viewState)
	}
	if gm.fileViewer.file == nil || gm.fileViewer.file.Name != "readme.txt" {
		t.Errorf("fileViewer.file wrong: %+v", gm.fileViewer.file)
	}
	if !gm.fileViewer.loading {
		t.Error("fileViewer.loading should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileListKey_Right_OnDatabase_OpensDatabaseView(t *testing.T) {
	// Need a real .sqlite file so DetectFileType returns FileTypeDatabase.
	dbPath := t.TempDir() + "/app.sqlite"
	if err := writeEmptyFile(dbPath); err != nil {
		t.Fatalf("setup: %v", err)
	}

	files := []simulator.FileInfo{
		{Name: "app.sqlite", Path: dbPath, IsDirectory: false},
	}
	m := Model{
		viewState: FileListView,
		fileList:  fileListState{files: files, cursor: 0},
		height:    30,
	}
	got, cmd := m.handleFileListKey("right")
	gm := asModel(t, got)

	if gm.viewState != DatabaseTableListView {
		t.Errorf("viewState = %v, want DatabaseTableListView", gm.viewState)
	}
	if gm.dbTables.file == nil {
		t.Error("dbTables.file should be set")
	}
	if !gm.dbTables.loading {
		t.Error("dbTables.loading should be true")
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
			m := Model{
				viewState: FileListView,
				fileList:  fileListState{files: files, cursor: tt.cursor},
				height:    30,
			}
			got, _ := m.handleFileListKey(tt.action)
			gm := asModel(t, got)
			if gm.fileList.cursor != tt.wantCursor {
				t.Errorf("fileList.cursor = %d, want %d", gm.fileList.cursor, tt.wantCursor)
			}
		})
	}
}

func TestHandleFileListKey_Open_ReturnsFinderCmd(t *testing.T) {
	m := Model{
		viewState: FileListView,
		fileList:  fileListState{files: fakeFiles(), cursor: 0},
		height:    30,
	}
	_, cmd := m.handleFileListKey("open")
	if cmd == nil {
		t.Error("expected openInFinderCmd")
	}
}

// ---------- handleFileViewerKey ----------

func TestHandleFileViewerKey_Left_ReturnsToFileList(t *testing.T) {
	file := simulator.FileInfo{Name: "x.txt", Path: "/x.txt"}
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			file:            &file,
			content:         &simulator.FileContent{Type: simulator.FileTypeText},
			contentOffset:   10,
			contentViewport: 5,
			svgWarning:      "warn",
		},
		height: 30,
	}
	got, _ := m.handleFileViewerKey("left")
	gm := asModel(t, got)

	if gm.viewState != FileListView {
		t.Errorf("viewState = %v, want FileListView", gm.viewState)
	}
	if gm.fileViewer.file != nil || gm.fileViewer.content != nil {
		t.Error("viewer state should be cleared")
	}
	if gm.fileViewer.contentOffset != 0 || gm.fileViewer.contentViewport != 0 {
		t.Error("content scroll state should be reset")
	}
	if gm.fileViewer.svgWarning != "" {
		t.Error("svgWarning should be cleared")
	}
}

func TestHandleFileViewerKey_Up_Text_ScrollsViewport(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			content: &simulator.FileContent{
				Type:       simulator.FileTypeText,
				Lines:      []string{"a", "b", "c"},
				TotalLines: 3,
			},
			contentViewport: 2,
		},
		height: 30,
	}
	got, _ := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.fileViewer.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.fileViewer.contentViewport)
	}
}

func TestHandleFileViewerKey_Up_Text_LoadsPreviousChunk(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.txt"}
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			file:            &file,
			content:         &simulator.FileContent{Type: simulator.FileTypeText},
			contentViewport: 0,
			contentOffset:   500,
		},
		height: 30,
	}
	got, cmd := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.fileViewer.contentOffset != 300 {
		t.Errorf("contentOffset = %d, want 300 (500 - 200)", gm.fileViewer.contentOffset)
	}
	if !gm.fileViewer.loading {
		t.Error("fileViewer.loading should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileViewerKey_Down_Text_ScrollsViewport(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			content: &simulator.FileContent{
				Type:       simulator.FileTypeText,
				Lines:      make([]string, 100),
				TotalLines: 100,
			},
			contentViewport: 0,
		},
		height: 30,
	}
	got, _ := m.handleFileViewerKey("down")
	gm := asModel(t, got)
	if gm.fileViewer.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.fileViewer.contentViewport)
	}
}

func TestHandleFileViewerKey_NilContent_NoOp(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		// fileViewer zero value — content is nil
		height: 30,
	}
	got, cmd := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.fileViewer.content != nil {
		t.Error("should remain nil")
	}
	if cmd != nil {
		t.Error("no cmd expected with nil content")
	}

	got, cmd = m.handleFileViewerKey("down")
	gm = asModel(t, got)
	if gm.fileViewer.content != nil {
		t.Error("should remain nil")
	}
	if cmd != nil {
		t.Error("no cmd expected with nil content")
	}
}

func TestHandleFileViewerKey_Up_Binary_LoadsPreviousChunk(t *testing.T) {
	file := simulator.FileInfo{Path: "/x.bin"}
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			file:            &file,
			content:         &simulator.FileContent{Type: simulator.FileTypeBinary, BinaryData: []byte{1, 2, 3}},
			contentViewport: 0,
			contentOffset:   512,
		},
		height: 30,
	}
	got, cmd := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.fileViewer.contentOffset != 256 {
		t.Errorf("contentOffset = %d, want 256 (512 - 256)", gm.fileViewer.contentOffset)
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileViewerKey_Up_Image_Scrolls(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			content: &simulator.FileContent{
				Type:      simulator.FileTypeImage,
				ImageInfo: &simulator.ImageInfo{Preview: &simulator.ImagePreview{Rows: []string{"a", "b"}}},
			},
			contentViewport: 3,
		},
		height: 30,
	}
	got, _ := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.fileViewer.contentViewport != 2 {
		t.Errorf("contentViewport = %d, want 2", gm.fileViewer.contentViewport)
	}
}

func TestHandleFileViewerKey_Up_Archive_Scrolls(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			content: &simulator.FileContent{
				Type:        simulator.FileTypeArchive,
				ArchiveInfo: &simulator.ArchiveInfo{Entries: []simulator.ArchiveEntry{{Name: "a"}, {Name: "b"}}},
			},
			contentViewport: 2,
		},
		height: 30,
	}
	got, _ := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.fileViewer.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.fileViewer.contentViewport)
	}
}

func TestHandleFileViewerKey_Down_Text_LoadsNextChunk(t *testing.T) {
	// With height=30, CalculateItemsPerScreen=7, itemsPerScreen-5=2.
	// Lines has 2 entries → maxViewport = 2 - 2 = 0. contentViewport=0
	// is NOT < 0, so the advance branch is skipped and the load-more
	// branch runs since contentOffset + len(Lines) < TotalLines.
	file := simulator.FileInfo{Path: "/x.txt"}
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			file: &file,
			content: &simulator.FileContent{
				Type:       simulator.FileTypeText,
				Lines:      []string{"line 1", "line 2"},
				TotalLines: 1000,
			},
			contentOffset:   0,
			contentViewport: 0,
		},
		height: 30,
	}
	got, cmd := m.handleFileViewerKey("down")
	gm := asModel(t, got)

	if gm.fileViewer.contentOffset != 2 {
		t.Errorf("contentOffset = %d, want 2 (next chunk starts after loaded lines)", gm.fileViewer.contentOffset)
	}
	if gm.fileViewer.contentViewport != 0 {
		t.Errorf("contentViewport = %d, want 0 (reset for new chunk)", gm.fileViewer.contentViewport)
	}
	if !gm.fileViewer.loading {
		t.Error("fileViewer.loading should be true")
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
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			file: &file,
			content: &simulator.FileContent{
				Type:         simulator.FileTypeBinary,
				BinaryData:   []byte{0, 1, 2, 3, 4, 5, 6, 7},
				BinaryOffset: 0,
				TotalSize:    1024,
			},
			contentOffset:   0,
			contentViewport: 0,
		},
		height: 30,
	}
	got, cmd := m.handleFileViewerKey("down")
	gm := asModel(t, got)

	if gm.fileViewer.contentOffset != 1 {
		t.Errorf("contentOffset = %d, want 1 (len(hexLines)=1)", gm.fileViewer.contentOffset)
	}
	if gm.fileViewer.contentViewport != 0 {
		t.Errorf("contentViewport = %d, want 0 (reset for new chunk)", gm.fileViewer.contentViewport)
	}
	if !gm.fileViewer.loading {
		t.Error("fileViewer.loading should be true")
	}
	if cmd == nil {
		t.Error("expected fetchFileContentCmd")
	}
}

func TestHandleFileViewerKey_Down_Image_AdvancesViewport(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			content: &simulator.FileContent{
				Type: simulator.FileTypeImage,
				ImageInfo: &simulator.ImageInfo{
					Preview: &simulator.ImagePreview{Rows: make([]string, 50)},
				},
			},
			contentViewport: 0,
		},
		height: 30,
	}
	got, _ := m.handleFileViewerKey("down")
	gm := asModel(t, got)
	if gm.fileViewer.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.fileViewer.contentViewport)
	}
}

func TestHandleFileViewerKey_Down_Archive_AdvancesViewport(t *testing.T) {
	entries := make([]simulator.ArchiveEntry, 50)
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			content: &simulator.FileContent{
				Type:        simulator.FileTypeArchive,
				ArchiveInfo: &simulator.ArchiveInfo{Entries: entries},
			},
			contentViewport: 0,
		},
		height: 30,
	}
	got, _ := m.handleFileViewerKey("down")
	gm := asModel(t, got)
	if gm.fileViewer.contentViewport != 1 {
		t.Errorf("contentViewport = %d, want 1", gm.fileViewer.contentViewport)
	}
}

func TestHandleFileViewerKey_Up_Text_AtTopNoOffset_NoOp(t *testing.T) {
	m := Model{
		viewState: FileViewerView,
		fileViewer: fileViewerState{
			content:         &simulator.FileContent{Type: simulator.FileTypeText, Lines: []string{"a"}},
			contentViewport: 0,
			contentOffset:   0,
		},
		height: 30,
	}
	got, cmd := m.handleFileViewerKey("up")
	gm := asModel(t, got)
	if gm.fileViewer.contentViewport != 0 || gm.fileViewer.contentOffset != 0 {
		t.Error("state should not change at top with no offset")
	}
	if cmd != nil {
		t.Error("no cmd expected at top")
	}
}

// ---------- handleDatabaseTableListKey ----------

func TestHandleDatabaseTableListKey_Left_ReturnsToFileList(t *testing.T) {
	m := Model{
		viewState: DatabaseTableListView,
		dbTables: dbTableListState{
			file:     &simulator.FileInfo{Name: "x.db"},
			info:     &simulator.DatabaseInfo{},
			cursor:   2,
			viewport: 1,
		},
		height: 30,
	}
	got, _ := m.handleDatabaseTableListKey("left")
	gm := asModel(t, got)

	if gm.viewState != FileListView {
		t.Errorf("viewState = %v, want FileListView", gm.viewState)
	}
	if gm.dbTables.file != nil || gm.dbTables.info != nil {
		t.Error("database state should be cleared")
	}
	if gm.dbTables.cursor != 0 || gm.dbTables.viewport != 0 {
		t.Error("table cursor/viewport should be reset")
	}
}

func TestHandleDatabaseTableListKey_Right_OpensTableContent(t *testing.T) {
	dbFile := simulator.FileInfo{Path: "/x.db"}
	m := Model{
		viewState: DatabaseTableListView,
		dbTables: dbTableListState{
			file: &dbFile,
			info: &simulator.DatabaseInfo{
				Tables: []simulator.TableInfo{{Name: "users"}, {Name: "posts"}},
			},
			cursor: 1,
		},
		height: 30,
	}
	got, cmd := m.handleDatabaseTableListKey("right")
	gm := asModel(t, got)

	if gm.viewState != DatabaseTableContentView {
		t.Errorf("viewState = %v, want DatabaseTableContentView", gm.viewState)
	}
	if gm.dbContent.table == nil || gm.dbContent.table.Name != "posts" {
		t.Errorf("dbContent.table = %+v", gm.dbContent.table)
	}
	if !gm.dbContent.loading {
		t.Error("dbContent.loading should be true")
	}
	if cmd == nil {
		t.Error("expected fetchTableDataCmd")
	}
}

func TestHandleDatabaseTableListKey_Navigation(t *testing.T) {
	info := &simulator.DatabaseInfo{Tables: []simulator.TableInfo{{Name: "a"}, {Name: "b"}}}
	m := Model{
		viewState: DatabaseTableListView,
		dbTables:  dbTableListState{info: info, cursor: 0},
		height:    30,
	}

	got, _ := m.handleDatabaseTableListKey("down")
	gm := asModel(t, got)
	if gm.dbTables.cursor != 1 {
		t.Errorf("down: dbTables.cursor = %d, want 1", gm.dbTables.cursor)
	}

	got, _ = gm.handleDatabaseTableListKey("up")
	gm = asModel(t, got)
	if gm.dbTables.cursor != 0 {
		t.Errorf("up: dbTables.cursor = %d, want 0", gm.dbTables.cursor)
	}
}

// ---------- handleDatabaseTableContentKey ----------

func TestHandleDatabaseTableContentKey_Left_ReturnsToTableList(t *testing.T) {
	table := simulator.TableInfo{Name: "users"}
	m := Model{
		viewState: DatabaseTableContentView,
		dbContent: dbTableContentState{
			table:    &table,
			data:     []map[string]any{{"id": 1}},
			offset:   5,
			viewport: 2,
		},
		height: 30,
	}
	got, _ := m.handleDatabaseTableContentKey("left")
	gm := asModel(t, got)

	if gm.viewState != DatabaseTableListView {
		t.Errorf("viewState = %v, want DatabaseTableListView", gm.viewState)
	}
	if gm.dbContent.table != nil {
		t.Error("dbContent.table should be cleared")
	}
	if gm.dbContent.data != nil {
		t.Error("dbContent.data should be cleared")
	}
	if gm.dbContent.offset != 0 || gm.dbContent.viewport != 0 {
		t.Error("dbContent offset/viewport should be reset")
	}
}

func TestHandleDatabaseTableContentKey_Up_Scrolls(t *testing.T) {
	m := Model{
		viewState: DatabaseTableContentView,
		dbContent: dbTableContentState{viewport: 3},
		height:    30,
	}
	got, _ := m.handleDatabaseTableContentKey("up")
	gm := asModel(t, got)
	if gm.dbContent.viewport != 2 {
		t.Errorf("dbContent.viewport = %d, want 2", gm.dbContent.viewport)
	}
}

func TestHandleDatabaseTableContentKey_Up_AtTop_NoOp(t *testing.T) {
	m := Model{
		viewState: DatabaseTableContentView,
		dbContent: dbTableContentState{viewport: 0},
		height:    30,
	}
	got, _ := m.handleDatabaseTableContentKey("up")
	gm := asModel(t, got)
	if gm.dbContent.viewport != 0 {
		t.Errorf("dbContent.viewport = %d, want 0", gm.dbContent.viewport)
	}
}

func TestHandleDatabaseTableContentKey_Down_LoadsMore(t *testing.T) {
	dbFile := simulator.FileInfo{Path: "/x.db"}
	table := simulator.TableInfo{Name: "users", RowCount: 500}
	m := Model{
		viewState: DatabaseTableContentView,
		dbTables:  dbTableListState{file: &dbFile},
		dbContent: dbTableContentState{
			table:    &table,
			data:     make([]map[string]any, 50),
			offset:   0,
			viewport: 1000, // way past end, forces load
		},
		height: 30,
	}
	got, cmd := m.handleDatabaseTableContentKey("down")
	gm := asModel(t, got)
	if gm.dbContent.offset != 50 {
		t.Errorf("dbContent.offset = %d, want 50", gm.dbContent.offset)
	}
	if !gm.dbContent.loading {
		t.Error("dbContent.loading should be true")
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
	m.simList.searchMode = true

	_, cmd := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	// In search mode, "q" is typed into the query, so handleSimulatorSearchInput
	// handles it — not the quit path.
	_ = cmd
}

func TestHandleKeyPress_NavigationClearsStatus(t *testing.T) {
	m := testModelWithKeyMap()
	m.viewState = SimulatorListView
	m.simList.simulators = fakeSims()
	m.simList.cursor = 1
	m.statusMessage = "previous status"

	got, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // j → down
	gm := asModel(t, got)

	if gm.statusMessage != "" {
		t.Errorf("statusMessage = %q, want empty (cleared by navigation)", gm.statusMessage)
	}
	if gm.simList.cursor != 2 {
		t.Errorf("simList.cursor = %d, want 2 (down advanced)", gm.simList.cursor)
	}
}

func TestHandleKeyPress_NonNavigationKeepsStatus(t *testing.T) {
	m := testModelWithKeyMap()
	m.viewState = SimulatorListView
	m.simList.simulators = fakeSims()
	m.simList.cursor = 0
	m.statusMessage = "keep me"

	got, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) // f → filter
	gm := asModel(t, got)

	if gm.statusMessage != "keep me" {
		t.Errorf("statusMessage = %q, want 'keep me' (non-nav should not clear)", gm.statusMessage)
	}
	if !gm.simList.filterActive {
		t.Error("simList.filterActive should be toggled")
	}
}

func TestHandleKeyPress_DispatchesByViewState(t *testing.T) {
	// Pressing "j" (down) on FileListView should move fileList.cursor,
	// not simList.cursor. Confirms the state-based dispatch works.
	m := testModelWithKeyMap()
	m.viewState = FileListView
	m.fileList.files = fakeFiles()
	m.fileList.cursor = 0
	m.simList.cursor = 0

	got, _ := m.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	gm := asModel(t, got)

	if gm.fileList.cursor != 1 {
		t.Errorf("fileList.cursor = %d, want 1", gm.fileList.cursor)
	}
	if gm.simList.cursor != 0 {
		t.Errorf("simList.cursor = %d, want 0 (untouched)", gm.simList.cursor)
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
