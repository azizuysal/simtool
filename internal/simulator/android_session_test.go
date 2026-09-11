package simulator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAndroidSessionListDoesNotStartAVD(t *testing.T) {
	fake := &fakeAndroid{avds: []string{"Pixel", "Tablet"}, running: map[string]string{"Pixel": "emulator-5554"}}
	session := newTestAndroidSession(fake, nil)

	devices, err := session.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if fake.startCount() != 0 {
		t.Fatalf("List() started %d emulators, want 0", fake.startCount())
	}
	if len(devices) != 2 {
		t.Fatalf("List() returned %d devices, want 2", len(devices))
	}
	if devices[0].Name != "Pixel" || devices[0].Serial != "emulator-5554" || devices[0].State != androidBooted {
		t.Fatalf("running device = %#v", devices[0])
	}
	if devices[1].Name != "Tablet" || devices[1].State != androidShutdown {
		t.Fatalf("stopped device = %#v", devices[1])
	}
}

func TestAndroidSessionEnsureReusesExistingEmulator(t *testing.T) {
	fake := &fakeAndroid{avds: []string{"Pixel"}, running: map[string]string{"Pixel": "emulator-5554"}, booted: true}
	session := newTestAndroidSession(fake, nil)

	serial, err := session.Ensure(context.Background(), "Pixel")
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if serial != "emulator-5554" {
		t.Fatalf("Ensure() serial = %q", serial)
	}
	if fake.startCount() != 0 {
		t.Fatalf("Ensure() started %d emulators, want 0", fake.startCount())
	}
	if err := session.Release("Pixel"); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if session.ownedCount() != 0 {
		t.Fatalf("Close() retained %d owned processes", session.ownedCount())
	}
}

func TestAndroidSessionReleaseStopsOnlyOwnedAVD(t *testing.T) {
	fake := &fakeAndroid{avds: []string{"Pixel"}, running: map[string]string{}, booted: true}
	session := newTestAndroidSession(fake, func() { fake.setRunning("Pixel", "emulator-5554") })
	if _, err := session.Ensure(context.Background(), "Pixel"); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	fake.delayDisconnect("emulator-5554", 1)
	if err := session.Release("Pixel"); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if !waitFor(func() bool { return session.ownedCount() == 0 }) {
		t.Fatal("Release() left managed process running")
	}
	if err := session.Release("Pixel"); err != nil {
		t.Fatalf("second Release() error = %v", err)
	}
}

func TestAndroidSessionReleaseWaitsForOwnedSerialBeforeNextInventory(t *testing.T) {
	fake := &fakeAndroid{avds: []string{"Pixel", "Tablet"}, running: map[string]string{}, booted: true}
	session := newTestAndroidSession(fake, func() { fake.setRunning("Pixel", "emulator-5554") })
	if _, err := session.Ensure(context.Background(), "Pixel"); err != nil {
		t.Fatalf("Ensure(Pixel) error = %v", err)
	}
	fake.delayDisconnect("emulator-5554", 2)
	if err := session.Release("Pixel"); err != nil {
		t.Fatalf("Release(Pixel) error = %v", err)
	}
	fake.setRunning("Tablet", "emulator-5556")
	if serial, err := session.Ensure(context.Background(), "Tablet"); err != nil || serial != "emulator-5556" {
		t.Fatalf("Ensure(Tablet) = %q, %v", serial, err)
	}
	_ = session.Close()
}

func TestAndroidSessionEnsureStartsAndClosesOnlyOwnedEmulator(t *testing.T) {
	fake := &fakeAndroid{avds: []string{"Pixel"}, running: map[string]string{}, booted: true}
	session := newTestAndroidSession(fake, func() { fake.setRunning("Pixel", "emulator-5554") })

	serial, err := session.Ensure(context.Background(), "Pixel")
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if serial != "emulator-5554" {
		t.Fatalf("Ensure() serial = %q", serial)
	}
	if fake.startCount() != 1 {
		t.Fatalf("starts = %d, want 1", fake.startCount())
	}
	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !waitFor(func() bool { return session.ownedCount() == 0 }) {
		t.Fatalf("managed helper process was not stopped")
	}
}

func TestAndroidSessionEnsureCancelsStartupAndCleansOwnedProcess(t *testing.T) {
	fake := &fakeAndroid{avds: []string{"Pixel"}, running: map[string]string{}}
	session := newTestAndroidSession(fake, nil)
	session.deps.bootTimeout = time.Second
	session.deps.pollInterval = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := session.Ensure(ctx, "Pixel")
		result <- err
	}()
	if !waitFor(func() bool { return fake.startCount() == 1 }) {
		t.Fatal("emulator did not start")
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Ensure() error = %v, want context cancellation", err)
	}
	if !waitFor(func() bool { return session.ownedCount() == 0 }) {
		t.Fatal("canceled startup left managed helper process running")
	}
}

func TestAndroidSessionIncludesEmulatorOutputWhenBootExits(t *testing.T) {
	fake := &fakeAndroid{avds: []string{"Pixel"}, running: map[string]string{}}
	session := newTestAndroidSession(fake, nil)
	session.deps.newCommand = func(_ string, _ ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestAndroidSessionHelperProcess", "--")
		cmd.Env = append(os.Environ(), "SIMTOOL_ANDROID_HELPER=exit")
		return cmd
	}
	_, err := session.Ensure(context.Background(), "Pixel")
	if err == nil {
		t.Fatal("Ensure() succeeded after emulator exit")
	}
	if !strings.Contains(err.Error(), "simtool helper startup failed") {
		t.Fatalf("Ensure() error omitted emulator output: %v", err)
	}
}

func TestAndroidSessionConcurrentEnsureStartsOnce(t *testing.T) {
	fake := &fakeAndroid{avds: []string{"Pixel"}, running: map[string]string{}, booted: true}
	session := newTestAndroidSession(fake, func() { fake.setRunning("Pixel", "emulator-5554") })

	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := session.Ensure(context.Background(), "Pixel")
			results <- err
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("concurrent Ensure() error = %v", err)
		}
	}
	if fake.startCount() != 1 {
		t.Fatalf("starts = %d, want 1", fake.startCount())
	}
	_ = session.Close()
}

func TestAndroidSessionHeadlessLifecycle(t *testing.T) {
	avd := os.Getenv("SIMTOOL_ANDROID_AVD")
	if avd == "" {
		t.Skip("set SIMTOOL_ANDROID_AVD to run the Android emulator lifecycle smoke test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	session, err := NewAndroidSession(ctx)
	if err != nil {
		t.Fatalf("NewAndroidSession() error = %v", err)
	}
	if !session.Available() {
		t.Fatal("Android SDK tools are unavailable")
	}
	session.deps.readOnly = true
	serial, err := session.Ensure(ctx, avd)
	if err != nil {
		t.Fatalf("Ensure(%q) error = %v", avd, err)
	}
	if serial == "" {
		t.Fatal("Ensure() returned an empty serial")
	}
	if err := session.Release(avd); err != nil {
		t.Fatalf("Release(%q) error = %v", avd, err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestParseAVDName(t *testing.T) {
	for _, test := range []struct {
		output string
		want   string
	}{
		{"avd name: Pixel_10_Pro\nOK\n", "Pixel_10_Pro"},
		{"Pixel_10_Pro\n", "Pixel_10_Pro"},
		{"KO: unavailable\n", ""},
	} {
		if got := parseAVDName(test.output); got != test.want {
			t.Errorf("parseAVDName(%q) = %q, want %q", test.output, got, test.want)
		}
	}
}

type fakeAndroid struct {
	mu      sync.Mutex
	avds    []string
	running map[string]string
	stale   map[string]int
	booted  bool
	starts  int
}

func (f *fakeAndroid) startCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.starts
}

func (f *fakeAndroid) setRunning(avd, serial string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.running[avd] = serial
}

func (f *fakeAndroid) delayDisconnect(serial string, polls int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stale == nil {
		f.stale = make(map[string]int)
	}
	f.stale[serial] = polls
}

func (f *fakeAndroid) command(_ context.Context, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if name == "emulator" && len(args) == 1 && args[0] == "-list-avds" {
		return []byte(strings.Join(f.avds, "\n") + "\n"), nil
	}
	if name != "adb" {
		return nil, errors.New("unexpected command")
	}
	if len(args) == 2 && args[0] == "devices" && args[1] == "-l" {
		var lines []string
		for avd, serial := range f.running {
			lines = append(lines, serial+" device product:sdk model:Pixel device:generic")
			if f.stale[serial] > 0 {
				f.stale[serial]--
				if f.stale[serial] == 0 {
					delete(f.running, avd)
					delete(f.stale, serial)
				}
			}
		}
		return []byte("List of devices attached\n" + strings.Join(lines, "\n") + "\n"), nil
	}
	if len(args) >= 4 && args[0] == "-s" && args[2] == "emu" && args[3] == "avd" {
		for avd, serial := range f.running {
			if serial == args[1] {
				return []byte("avd name: " + avd + "\nOK\n"), nil
			}
		}
		return nil, errors.New("unknown serial")
	}
	if len(args) >= 5 && args[0] == "-s" && args[2] == "shell" && args[3] == "getprop" {
		switch args[4] {
		case "sys.boot_completed":
			if f.booted {
				return []byte("1\n"), nil
			}
			return []byte("0\n"), nil
		case "ro.build.version.release":
			return []byte("36\n"), nil
		}
	}
	return nil, errors.New("unexpected adb arguments")
}

func newTestAndroidSession(fake *fakeAndroid, started func()) *AndroidSession {
	session := newAndroidSession(androidSessionDeps{
		adbPath:         "adb",
		emulatorPath:    "emulator",
		command:         fake.command,
		bootTimeout:     time.Second,
		pollInterval:    time.Millisecond,
		shutdownTimeout: time.Second,
	})
	session.deps.newCommand = func(_ string, _ ...string) *exec.Cmd {
		fake.mu.Lock()
		fake.starts++
		fake.mu.Unlock()
		if started != nil {
			started()
		}
		cmd := exec.Command(os.Args[0], "-test.run=TestAndroidSessionHelperProcess", "--")
		cmd.Env = append(os.Environ(), "SIMTOOL_ANDROID_HELPER=1")
		return cmd
	}
	return session
}

func TestAndroidSessionHelperProcess(t *testing.T) {
	switch os.Getenv("SIMTOOL_ANDROID_HELPER") {
	case "exit":
		_, _ = fmt.Fprintln(os.Stderr, "simtool helper startup failed")
		os.Exit(17)
	case "1":
	default:
		return
	}
	for {
		time.Sleep(time.Second)
	}
}

func waitFor(condition func() bool) bool {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return condition()
}

func (s *AndroidSession) ownedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.owned)
}
