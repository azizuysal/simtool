package simulator

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFinderBridgeServesReadOnlyCapability(t *testing.T) {
	t.Parallel()
	localFile := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(localFile, []byte("hello from android"), 0600); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var listed []string
	var materialized []string
	bridge, err := NewFinderBridge(context.Background(), func(_ context.Context, directory string) ([]FileInfo, error) {
		mu.Lock()
		listed = append(listed, directory)
		mu.Unlock()
		switch directory {
		case "container":
			return []FileInfo{
				{Name: "Documents", Path: "container/Documents", IsDirectory: true, ModifiedAt: time.Unix(100, 0)},
				{Name: "hello.txt", Path: "container/hello.txt", Size: 18, ModifiedAt: time.Unix(100, 0)},
				{Name: "mön%25.txt", Path: "container/mön%25.txt", Size: 18, ModifiedAt: time.Unix(100, 0)},
			}, nil
		case "container/Documents":
			return []FileInfo{{Name: "note.txt", Path: "container/Documents/note.txt", Size: 4}}, nil
		default:
			t.Fatalf("unexpected listed directory %q", directory)
			return nil, nil
		}
	}, func(_ context.Context, remotePath string) (string, error) {
		mu.Lock()
		materialized = append(materialized, remotePath)
		mu.Unlock()
		if remotePath != "container/hello.txt" && remotePath != "container/mön%25.txt" {
			t.Fatalf("unexpected materialized path %q", remotePath)
		}
		return localFile, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := bridge.Close(); err != nil {
			t.Error(err)
		}
	})

	bridgeURL, err := bridge.URL(context.Background(), "container")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(bridgeURL, "http://127.0.0.1:") {
		t.Fatalf("unexpected bridge URL %q", bridgeURL)
	}

	response := finderBridgeRequest(t, http.MethodGet, bridgeURL+"hello.txt", "")
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		t.Fatalf("GET status = %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(body); got != "hello from android" {
		t.Fatalf("GET body = %q", got)
	}
	mu.Lock()
	if got := append([]string(nil), materialized...); len(got) != 1 || got[0] != "container/hello.txt" {
		mu.Unlock()
		t.Fatalf("GET materialized = %#v", got)
	}
	mu.Unlock()
	escapedName := url.PathEscape("mön%25.txt")
	response = finderBridgeRequest(t, http.MethodGet, bridgeURL+escapedName, "")
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		t.Fatalf("escaped-name GET status = %d", response.StatusCode)
	}
	response.Body.Close()

	response = finderBridgeRequest(t, "PROPFIND", bridgeURL, "1")
	if response.StatusCode != http.StatusMultiStatus {
		response.Body.Close()
		t.Fatalf("PROPFIND status = %d", response.StatusCode)
	}
	propfindBody, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	parsedURL, err := url.Parse(bridgeURL)
	if err != nil {
		t.Fatal(err)
	}
	capabilityPath := strings.TrimSuffix(parsedURL.EscapedPath(), "/")
	if !strings.Contains(string(propfindBody), "hello.txt") || !strings.Contains(string(propfindBody), "Documents") || !strings.Contains(string(propfindBody), capabilityPath) {
		t.Fatalf("PROPFIND did not include listed entries: %s", propfindBody)
	}
	mu.Lock()
	if got := append([]string(nil), materialized...); len(got) != 2 {
		mu.Unlock()
		t.Fatalf("PROPFIND materialized additional files: %#v", got)
	}
	mu.Unlock()

	response = finderBridgeRequest(t, http.MethodPut, bridgeURL+"hello.txt", "")
	if response.StatusCode != http.StatusMethodNotAllowed {
		response.Body.Close()
		t.Fatalf("PUT status = %d", response.StatusCode)
	}
	response.Body.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(materialized) == 0 || materialized[0] != "container/hello.txt" {
		t.Fatalf("materialized = %#v", materialized)
	}
	for _, remotePath := range materialized {
		if remotePath != "container/hello.txt" && remotePath != "container/mön%25.txt" {
			t.Fatalf("materialized = %#v", materialized)
		}
	}
	if len(listed) == 0 || listed[0] != "container" {
		t.Fatalf("listed = %#v", listed)
	}
}

func TestFinderBridgeRejectsTraversalWrongHostAndSymlinks(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	target := filepath.Join(directory, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(directory, "link.txt")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatal(err)
	}

	bridge, err := NewFinderBridge(context.Background(), func(_ context.Context, remotePath string) ([]FileInfo, error) {
		if remotePath != "container" {
			t.Fatalf("listed unexpected path %q", remotePath)
		}
		return []FileInfo{{Name: "link.txt", Path: "container/link.txt", Size: 6}}, nil
	}, func(_ context.Context, remotePath string) (string, error) {
		if remotePath != "container/link.txt" {
			t.Fatalf("materialized unexpected path %q", remotePath)
		}
		return symlink, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bridge.Close() })

	bridgeURL, err := bridge.URL(context.Background(), "container")
	if err != nil {
		t.Fatal(err)
	}
	response := finderBridgeRequest(t, http.MethodGet, bridgeURL+"%2e%2e/secret.txt", "")
	if response.StatusCode != http.StatusNotFound {
		response.Body.Close()
		t.Fatalf("traversal status = %d", response.StatusCode)
	}
	response.Body.Close()

	request, err := http.NewRequest(http.MethodGet, bridgeURL+"link.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "localhost"
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusForbidden {
		response.Body.Close()
		t.Fatalf("wrong host status = %d", response.StatusCode)
	}
	response.Body.Close()

	response = finderBridgeRequest(t, http.MethodGet, bridgeURL+"link.txt", "")
	if response.StatusCode == http.StatusOK {
		response.Body.Close()
		t.Fatalf("symlink GET status = %d", response.StatusCode)
	}
	response.Body.Close()
}

func TestFinderBridgeCloseStopsServing(t *testing.T) {
	t.Parallel()
	bridge, err := NewFinderBridge(context.Background(), func(context.Context, string) ([]FileInfo, error) {
		return nil, nil
	}, func(context.Context, string) (string, error) {
		return "", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	bridgeURL, err := bridge.URL(context.Background(), "container")
	if err != nil {
		t.Fatal(err)
	}
	if err := bridge.Close(); err != nil {
		t.Fatal(err)
	}
	if response, err := http.Get(bridgeURL); err == nil {
		response.Body.Close()
		t.Fatal("GET succeeded after Close")
	}
	if _, err := bridge.URL(context.Background(), "container"); err == nil {
		t.Fatal("URL succeeded after Close")
	}
}

func finderBridgeRequest(t *testing.T, method, rawURL, depth string) *http.Response {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, parsed.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if depth != "" {
		request.Header.Set("Depth", depth)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
