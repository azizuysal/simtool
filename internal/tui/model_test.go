package tui

import (
	"testing"
	"time"

	"simtool/internal/simulator"
)

// mockFetcher implements simulator.Fetcher for testing
type mockFetcher struct {
	simulators []simulator.Item
	fetchErr   error
	bootErr    error
	bootCalled bool
	bootUDID   string
}

func (m *mockFetcher) Fetch() ([]simulator.Item, error) {
	return m.simulators, m.fetchErr
}

func (m *mockFetcher) Boot(udid string) error {
	m.bootCalled = true
	m.bootUDID = udid
	return m.bootErr
}

func TestNew(t *testing.T) {
	fetcher := &mockFetcher{}
	model := New(fetcher)
	
	if model.fetcher != fetcher {
		t.Error("Expected fetcher to be set")
	}
	
	if model.viewState != SimulatorListView {
		t.Error("Expected initial view state to be SimulatorListView")
	}
	
	if model.err != nil {
		t.Error("Expected no initial error")
	}
}

func TestInit(t *testing.T) {
	fetcher := &mockFetcher{}
	model := New(fetcher)
	
	cmd := model.Init()
	
	// Init should return a batch command
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}
}

func TestFetchSimulatorsCmd(t *testing.T) {
	tests := []struct {
		name       string
		simulators []simulator.Item
		err        error
		wantErr    bool
	}{
		{
			name: "successful fetch",
			simulators: []simulator.Item{
				{
					Simulator: simulator.Simulator{
						UDID: "test-123",
						Name: "iPhone 15",
					},
					Runtime:  "iOS 17.0",
					AppCount: 5,
				},
			},
			err:     nil,
			wantErr: false,
		},
		{
			name:       "fetch error",
			simulators: nil,
			err:        simulator.ErrSimulatorNotFound,
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockFetcher{
				simulators: tt.simulators,
				fetchErr:   tt.err,
			}
			
			cmd := fetchSimulatorsCmd(fetcher)
			msg := cmd()
			
			fetchMsg, ok := msg.(fetchSimulatorsMsg)
			if !ok {
				t.Fatal("Expected fetchSimulatorsMsg")
			}
			
			if (fetchMsg.err != nil) != tt.wantErr {
				t.Errorf("fetchSimulatorsCmd() error = %v, wantErr %v", fetchMsg.err, tt.wantErr)
			}
			
			if !tt.wantErr && len(fetchMsg.simulators) != len(tt.simulators) {
				t.Errorf("Expected %d simulators, got %d", len(tt.simulators), len(fetchMsg.simulators))
			}
		})
	}
}

func TestBootSimulatorCmd(t *testing.T) {
	fetcher := &mockFetcher{}
	model := Model{fetcher: fetcher}
	
	udid := "test-udid-123"
	cmd := model.bootSimulatorCmd(udid)
	msg := cmd()
	
	bootMsg, ok := msg.(bootSimulatorMsg)
	if !ok {
		t.Fatal("Expected bootSimulatorMsg")
	}
	
	if bootMsg.udid != udid {
		t.Errorf("Expected UDID %s, got %s", udid, bootMsg.udid)
	}
	
	if !fetcher.bootCalled {
		t.Error("Expected Boot to be called")
	}
	
	if fetcher.bootUDID != udid {
		t.Errorf("Expected boot UDID %s, got %s", udid, fetcher.bootUDID)
	}
}

func TestFetchAppsCmd(t *testing.T) {
	sim := simulator.Item{
		Simulator: simulator.Simulator{
			UDID:  "test-123",
			Name:  "iPhone 15",
			State: "Booted",
		},
	}
	
	model := Model{}
	cmd := model.fetchAppsCmd(sim)
	
	// Since this calls an external function, we can only test that it returns a command
	if cmd == nil {
		t.Error("Expected fetchAppsCmd to return a command")
	}
}

func TestFetchFilesCmd(t *testing.T) {
	containerPath := "/path/to/container"
	
	model := Model{}
	cmd := model.fetchFilesCmd(containerPath)
	
	// Since this calls an external function, we can only test that it returns a command
	if cmd == nil {
		t.Error("Expected fetchFilesCmd to return a command")
	}
}

func TestOpenInFinderCmd(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "regular path",
			path: "/path/to/file",
		},
		{
			name: "file URL path",
			path: "file:///path/to/file",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{}
			cmd := model.openInFinderCmd(tt.path)
			
			if cmd == nil {
				t.Error("Expected openInFinderCmd to return a command")
			}
		})
	}
}

func TestFetchFileContentCmd(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		offset    int
		height    int
		fileType  simulator.FileType
		maxLines  int
	}{
		{
			name:     "text file",
			path:     "/path/to/file.txt",
			offset:   0,
			height:   30,
			maxLines: 500,
		},
		{
			name:     "image file",
			path:     "/path/to/image.png",
			offset:   0,
			height:   30,
			maxLines: 11, // height - 19
		},
		{
			name:     "binary file",
			path:     "/path/to/file.bin",
			offset:   100,
			height:   30,
			maxLines: 500,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{height: tt.height}
			cmd := model.fetchFileContentCmd(tt.path, tt.offset)
			
			if cmd == nil {
				t.Error("Expected fetchFileContentCmd to return a command")
			}
		})
	}
}

func TestModelState(t *testing.T) {
	model := Model{
		viewState: SimulatorListView,
		simCursor: 5,
		simulators: []simulator.Item{
			{Simulator: simulator.Simulator{Name: "iPhone 15"}},
			{Simulator: simulator.Simulator{Name: "iPhone 14"}},
		},
		height: 30,
		width:  80,
	}
	
	// Test view state
	if model.viewState != SimulatorListView {
		t.Errorf("Expected view state %v, got %v", SimulatorListView, model.viewState)
	}
	
	// Test cursor position
	if model.simCursor != 5 {
		t.Errorf("Expected cursor at 5, got %d", model.simCursor)
	}
	
	// Test dimensions
	if model.height != 30 || model.width != 80 {
		t.Errorf("Expected dimensions 30x80, got %dx%d", model.height, model.width)
	}
}

func TestMessageTypes(t *testing.T) {
	// Test fetchSimulatorsMsg
	fetchMsg := fetchSimulatorsMsg{
		simulators: []simulator.Item{{Simulator: simulator.Simulator{Name: "Test"}}},
		err:        nil,
	}
	if len(fetchMsg.simulators) != 1 {
		t.Error("fetchSimulatorsMsg not properly constructed")
	}
	
	// Test bootSimulatorMsg
	bootMsg := bootSimulatorMsg{
		udid: "test-123",
		err:  nil,
	}
	if bootMsg.udid != "test-123" {
		t.Error("bootSimulatorMsg not properly constructed")
	}
	
	// Test fetchAppsMsg
	appsMsg := fetchAppsMsg{
		apps: []simulator.App{{Name: "TestApp"}},
		err:  nil,
	}
	if len(appsMsg.apps) != 1 {
		t.Error("fetchAppsMsg not properly constructed")
	}
	
	// Test tickMsg
	tickMsg := tickMsg(time.Now())
	if time.Time(tickMsg).IsZero() {
		t.Error("tickMsg not properly constructed")
	}
	
	// Test fetchFilesMsg
	filesMsg := fetchFilesMsg{
		files: []simulator.FileInfo{{Name: "test.txt"}},
		err:   nil,
	}
	if len(filesMsg.files) != 1 {
		t.Error("fetchFilesMsg not properly constructed")
	}
	
	// Test openInFinderMsg
	finderMsg := openInFinderMsg{err: nil}
	if finderMsg.err != nil {
		t.Error("openInFinderMsg not properly constructed")
	}
	
	// Test fetchFileContentMsg
	contentMsg := fetchFileContentMsg{
		content: &simulator.FileContent{Type: simulator.FileTypeText},
		err:     nil,
	}
	if contentMsg.content.Type != simulator.FileTypeText {
		t.Error("fetchFileContentMsg not properly constructed")
	}
}

func TestViewStateConstants(t *testing.T) {
	// Test that view states have distinct values
	states := map[ViewState]string{
		SimulatorListView: "SimulatorListView",
		AppListView:       "AppListView",
		FileListView:      "FileListView",
		FileViewerView:    "FileViewerView",
	}
	
	seen := make(map[ViewState]bool)
	for state, name := range states {
		if seen[state] {
			t.Errorf("Duplicate view state value for %s", name)
		}
		seen[state] = true
	}
	
	// Verify initial state
	if SimulatorListView != 0 {
		t.Error("Expected SimulatorListView to be the zero value")
	}
}