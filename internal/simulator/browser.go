package simulator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Browser interface {
	Fetcher
	Apps(Item) ([]App, error)
	AllApps() ([]App, error)
	Files(string) ([]FileInfo, error)
	Prepare(string) (string, error)
	OpenInFinder(string) error
}

// DeviceBrowser owns the Android processes, Finder bridge and private previews
// for one invocation of SimTool.
type DeviceBrowser struct {
	ctx       context.Context
	cancel    context.CancelFunc
	ios       Fetcher
	android   *AndroidSession
	platform  string
	cache     string
	aapt      string
	bridge    *FinderBridge
	mu        sync.Mutex
	closed    bool
	work      sync.WaitGroup
	prepared  map[string]string
	counts    map[string]int
	mounts    []string
	previewMu sync.Mutex
}

func NewBrowser(ctx context.Context, platform string) (*DeviceBrowser, error) {
	if platform != "all" && platform != "ios" && platform != "android" {
		return nil, fmt.Errorf("invalid platform %q: use all, ios or android", platform)
	}
	ctx, cancel := context.WithCancel(ctx)
	b := &DeviceBrowser{ctx: ctx, cancel: cancel, platform: platform, prepared: make(map[string]string), counts: make(map[string]int)}
	if platform != "android" {
		if err := exec.CommandContext(ctx, "xcrun", "--find", "simctl").Run(); err == nil {
			b.ios = NewFetcher()
		} else if platform == "ios" {
			cancel()
			return nil, fmt.Errorf("iOS support requires Xcode and simctl: %w", err)
		}
	}
	var err error
	if platform != "ios" {
		b.android, err = NewAndroidSession(ctx)
		if err != nil {
			cancel()
			return nil, err
		}
		if platform == "android" && !b.android.Available() {
			cancel()
			return nil, errors.New("for Android support, install SDK platform-tools and emulator; set ANDROID_HOME")
		}
	}
	if b.ios == nil && (b.android == nil || !b.android.Available()) {
		cancel()
		return nil, errors.New("no simulator tools found: install Xcode or Android SDK platform-tools and emulator")
	}
	b.cache, err = os.MkdirTemp("", "simtool-session-")
	if err != nil {
		cancel()
		return nil, err
	}
	b.aapt = findAAPT2()
	return b, nil
}

func (b *DeviceBrowser) begin() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("the SimTool session is closed")
	}
	if err := b.ctx.Err(); err != nil {
		return err
	}
	b.work.Add(1)
	return nil
}

func (b *DeviceBrowser) Fetch() ([]Item, error) {
	if err := b.begin(); err != nil {
		return nil, err
	}
	defer b.work.Done()
	var items []Item
	var errs []error
	if b.ios != nil {
		found, err := b.ios.Fetch()
		if err != nil {
			errs = append(errs, fmt.Errorf("iOS: %w", err))
		}
		for i := range found {
			found[i].Platform = "ios"
		}
		items = append(items, found...)
	}
	if b.android != nil && b.android.Available() {
		devices, err := b.android.List(b.ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("discover Android emulators: %w", err))
		}
		for _, device := range devices {
			id := androidID(device.Name)
			b.mu.Lock()
			count, ok := b.counts[id]
			b.mu.Unlock()
			if !ok {
				count = -1
			}
			items = append(items, Item{Simulator: Simulator{UDID: id, Name: device.Name, State: device.State, Platform: "android", IsAvailable: true}, Runtime: device.Runtime, AppCount: count})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, errors.Join(errs...)
}

func (b *DeviceBrowser) FetchSimulators() ([]Simulator, error) {
	items, err := b.Fetch()
	sims := make([]Simulator, len(items))
	for i := range items {
		sims[i] = items[i].Simulator
	}
	return sims, err
}

func androidID(avd string) string {
	return "android:" + base64.RawURLEncoding.EncodeToString([]byte(avd))
}

func androidName(id string) (string, error) {
	if !strings.HasPrefix(id, "android:") {
		return "", errors.New("invalid Android device identifier")
	}
	value, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(id, "android:"))
	if err != nil || len(value) == 0 || strings.ContainsAny(string(value), "/\x00\r\n") {
		return "", errors.New("invalid Android device identifier")
	}
	return string(value), nil
}

func (b *DeviceBrowser) Boot(id string) error {
	if err := b.begin(); err != nil {
		return err
	}
	defer b.work.Done()
	if strings.HasPrefix(id, "android:") {
		name, err := androidName(id)
		if err != nil {
			return err
		}
		if b.android == nil {
			return errors.New("the Android SDK is unavailable")
		}
		_, err = b.android.Ensure(b.ctx, name)
		return err
	}
	if b.ios == nil {
		return errors.New("the Xcode simulator tools are unavailable")
	}
	return b.ios.Boot(id)
}

func (b *DeviceBrowser) Apps(item Item) ([]App, error) {
	if err := b.begin(); err != nil {
		return nil, err
	}
	defer b.work.Done()
	if item.Platform == "android" || strings.HasPrefix(item.UDID, "android:") {
		name, err := androidName(item.UDID)
		if err != nil {
			return nil, err
		}
		apps, err := b.androidApps(b.ctx, name)
		if err == nil {
			b.mu.Lock()
			b.counts[item.UDID] = len(apps)
			b.mu.Unlock()
		}
		return apps, err
	}
	apps, err := GetAppsForSimulator(item.UDID, item.IsRunning())
	for i := range apps {
		apps[i].Platform = "ios"
	}
	return apps, err
}

func (b *DeviceBrowser) AllApps() ([]App, error) {
	items, fetchErr := b.Fetch()
	errs := []error{fetchErr}
	var result []App
	for _, item := range items {
		if err := b.ctx.Err(); err != nil {
			return result, err
		}
		apps, err := b.Apps(item)
		if item.Platform == "android" && !item.IsRunning() {
			name, nameErr := androidName(item.UDID)
			if nameErr != nil {
				errs = append(errs, nameErr)
			} else {
				errs = append(errs, b.android.Release(name))
			}
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", item.Name, err))
		}
		for i := range apps {
			apps[i].SimulatorName = item.Name
			apps[i].SimulatorUDID = item.UDID
		}
		result = append(result, apps...)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].SimulatorName < result[j].SimulatorName
	})
	return result, errors.Join(errs...)
}

func (b *DeviceBrowser) Files(name string) ([]FileInfo, error) {
	if err := b.begin(); err != nil {
		return nil, err
	}
	defer b.work.Done()
	if isAndroidPath(name) {
		return b.androidFiles(b.ctx, name)
	}
	return GetFilesForContainer(name)
}

func (b *DeviceBrowser) Prepare(name string) (string, error) {
	if err := b.begin(); err != nil {
		return "", err
	}
	defer b.work.Done()
	if !isAndroidPath(name) {
		return strings.TrimPrefix(name, "file://"), nil
	}
	b.previewMu.Lock()
	defer b.previewMu.Unlock()
	local, err := b.androidPrepare(b.ctx, name)
	if err != nil {
		return "", err
	}
	return local, nil
}

func (b *DeviceBrowser) OpenInFinder(name string) error {
	if err := b.begin(); err != nil {
		return err
	}
	defer b.work.Done()
	if !isAndroidPath(name) {
		return exec.CommandContext(b.ctx, "open", "-R", strings.TrimPrefix(name, "file://")).Run()
	}
	// Finder requires a mounted WebDAV volume; passing an HTTP URL to open
	// would launch the browser instead.
	parts, err := parseAndroidPath(name)
	if err != nil {
		return err
	}
	root := androidContainer(parts.avd, parts.pkg)
	b.mu.Lock()
	if b.bridge == nil {
		b.bridge, err = NewFinderBridge(b.ctx, func(ctx context.Context, path string) ([]FileInfo, error) { return b.androidFiles(ctx, path) }, func(ctx context.Context, path string) (string, error) {
			b.previewMu.Lock()
			defer b.previewMu.Unlock()
			return b.androidPrepare(ctx, path)
		})
	}
	bridge := b.bridge
	b.mu.Unlock()
	if err != nil {
		return err
	}
	address, err := bridge.URL(b.ctx, root)
	if err != nil {
		return err
	}
	mount, err := os.MkdirTemp(b.cache, "finder-")
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.mounts = append(b.mounts, mount)
	b.mu.Unlock()
	if _, err := exec.CommandContext(b.ctx, "/sbin/mount_webdav", "-S", "-o", "rdonly", "-v", parts.pkg, address, mount).CombinedOutput(); err != nil {
		return fmt.Errorf("mount Android folder in Finder: %w", err)
	}
	rel, err := filepath.Rel(root, name)
	if err != nil {
		return err
	}
	return exec.CommandContext(b.ctx, "open", "-R", filepath.Join(mount, rel)).Run()
}

func (b *DeviceBrowser) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.cancel()
	b.mu.Unlock()
	var errs []error
	if b.android != nil {
		errs = append(errs, b.android.Close())
	}
	b.work.Wait()
	var mountErrors []error
	for _, mount := range b.mounts {
		if err := unmountFinder(mount); err != nil {
			mountErrors = append(mountErrors, err)
		}
	}
	if b.bridge != nil {
		errs = append(errs, b.bridge.Close())
	}
	// Never remove a mounted directory if unmount failed.
	if err := errors.Join(mountErrors...); err != nil {
		return errors.Join(append(errs, err)...)
	}
	return errors.Join(append(errs, os.RemoveAll(b.cache))...)
}

func unmountFinder(mount string) error {
	// mount_webdav -S also unmounts automatically when its server disappears.
	// Compare device IDs so a volume already ejected in Finder is harmless.
	if mounted, err := isFinderMounted(mount); err != nil || !mounted {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(ctx, "/sbin/umount", "-f", mount).CombinedOutput(); err != nil {
		if mounted, checkErr := isFinderMounted(mount); checkErr == nil && !mounted {
			return nil
		}
		return fmt.Errorf("unmount Android Finder folder: %w: %s", err, output)
	}
	return nil
}
