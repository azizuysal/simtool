//go:build darwin

package simulator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFinderBridgeMountWebDAV(t *testing.T) {
	if os.Getenv("SIMTOOL_TEST_NATIVE_WEBDAV") != "1" {
		t.Skip("set SIMTOOL_TEST_NATIVE_WEBDAV=1 to mount the loopback WebDAV bridge")
	}

	fixture := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(fixture, []byte("hello from finder"), 0600); err != nil {
		t.Fatal(err)
	}
	bridge, err := NewFinderBridge(context.Background(), func(_ context.Context, directory string) ([]FileInfo, error) {
		if directory != "container" {
			t.Fatalf("listed unexpected directory %q", directory)
		}
		return []FileInfo{{Name: "hello.txt", Path: "container/hello.txt", Size: 17}}, nil
	}, func(_ context.Context, remotePath string) (string, error) {
		if remotePath != "container/hello.txt" {
			t.Fatalf("materialized unexpected path %q", remotePath)
		}
		return fixture, nil
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

	mountPoint := t.TempDir()
	mount := exec.Command("/sbin/mount_webdav", "-S", bridgeURL, mountPoint)
	if output, err := mount.CombinedOutput(); err != nil {
		t.Fatalf("mount WebDAV bridge: %v: %s", err, output)
	}
	t.Cleanup(func() {
		output, err := exec.Command("/sbin/umount", mountPoint).CombinedOutput()
		if err != nil {
			t.Errorf("unmount WebDAV bridge: %v: %s", err, output)
		}
	})

	contents, err := os.ReadFile(filepath.Join(mountPoint, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(contents); got != "hello from finder" {
		t.Fatalf("mounted file = %q", got)
	}
}
