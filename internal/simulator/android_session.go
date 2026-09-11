package simulator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	androidBooted           = "Booted"
	androidShutdown         = "Shutdown"
	maxAndroidCommandOutput = 256 << 20
	androidCommandTimeout   = 30 * time.Second
)

// AndroidDevice is an Android Virtual Device known to the local Android SDK.
// Serial is present only while that AVD is running.
type AndroidDevice struct {
	Name    string
	Serial  string
	State   string
	Runtime string
}

type androidSessionDeps struct {
	ctx             context.Context
	adbPath         string
	emulatorPath    string
	command         func(context.Context, string, ...string) ([]byte, error)
	newCommand      func(string, ...string) *exec.Cmd
	readOnly        bool
	bootTimeout     time.Duration
	pollInterval    time.Duration
	shutdownTimeout time.Duration
	releaseTimeout  time.Duration
}

// AndroidSession discovers Android Virtual Devices and manages headless
// emulators started by SimTool. It never stops an emulator it did not start.
type AndroidSession struct {
	deps   androidSessionDeps
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	closed   bool
	starting map[string]*androidStart
	owned    map[string]*ownedAndroidEmulator
}

type androidStart struct {
	done   chan struct{}
	serial string
	err    error
}

type ownedAndroidEmulator struct {
	cmd      *exec.Cmd
	done     chan error
	serial   string
	log      *rollingBuffer
	stopOnce sync.Once
	stopErr  error
}

// NewAndroidSession creates an Android session. Missing Android tools are
// represented by Available returning false so iOS use remains unaffected.
func NewAndroidSession(ctx context.Context) (*AndroidSession, error) {
	adbPath, emulatorPath := findAndroidTools()
	return newAndroidSession(androidSessionDeps{
		ctx:             ctx,
		adbPath:         adbPath,
		emulatorPath:    emulatorPath,
		command:         runAndroidCommand,
		newCommand:      exec.Command,
		bootTimeout:     2 * time.Minute,
		pollInterval:    500 * time.Millisecond,
		shutdownTimeout: 10 * time.Second,
		releaseTimeout:  5 * time.Second,
	}), nil
}

func newAndroidSession(deps androidSessionDeps) *AndroidSession {
	if deps.ctx == nil {
		deps.ctx = context.Background()
	}
	if deps.command == nil {
		deps.command = runAndroidCommand
	}
	if deps.newCommand == nil {
		deps.newCommand = exec.Command
	}
	if deps.bootTimeout <= 0 {
		deps.bootTimeout = 2 * time.Minute
	}
	if deps.pollInterval <= 0 {
		deps.pollInterval = 500 * time.Millisecond
	}
	if deps.shutdownTimeout <= 0 {
		deps.shutdownTimeout = 10 * time.Second
	}
	if deps.releaseTimeout <= 0 {
		deps.releaseTimeout = 5 * time.Second
	}
	ctx, cancel := context.WithCancel(deps.ctx)
	return &AndroidSession{
		deps:     deps,
		ctx:      ctx,
		cancel:   cancel,
		starting: make(map[string]*androidStart),
		owned:    make(map[string]*ownedAndroidEmulator),
	}
}

// Available reports whether both adb and the Android emulator executable exist.
func (s *AndroidSession) Available() bool {
	return s != nil && s.deps.adbPath != "" && s.deps.emulatorPath != ""
}

// List reports configured AVDs and running Android emulators. It never starts
// an emulator.
func (s *AndroidSession) List(ctx context.Context) ([]AndroidDevice, error) {
	if err := s.requireAvailable(); err != nil {
		return nil, err
	}
	ctx, done, err := s.withSession(ctx)
	if err != nil {
		return nil, err
	}
	defer done()

	avds, err := s.listAVDs(ctx)
	if err != nil {
		return nil, err
	}
	running, err := s.runningEmulators(ctx)
	if err != nil {
		return nil, err
	}

	devices := make(map[string]AndroidDevice, len(avds)+len(running))
	for _, name := range avds {
		devices[name] = AndroidDevice{Name: name, State: androidShutdown, Runtime: "Android (unknown)"}
	}
	for name, serial := range running {
		if name == "" {
			continue
		}
		devices[name] = AndroidDevice{Name: name, Serial: serial, State: androidBooted}
	}

	result := make([]AndroidDevice, 0, len(devices))
	for _, device := range devices {
		if device.State == androidBooted {
			device.Runtime = s.runtime(ctx, device.Serial)
		}
		result = append(result, device)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// Ensure returns a boot-complete emulator serial for avd. Existing emulators
// are reused. A stopped AVD is launched headlessly and becomes owned by this
// session until Close.
func (s *AndroidSession) Ensure(ctx context.Context, avd string) (string, error) {
	if err := s.requireAvailable(); err != nil {
		return "", err
	}
	avd = strings.TrimSpace(avd)
	if avd == "" {
		return "", errors.New("an Android AVD name is required")
	}
	ctx, done, err := s.withSession(ctx)
	if err != nil {
		return "", err
	}
	defer done()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return "", errors.New("the Android session is closed")
	}
	if start := s.starting[avd]; start != nil {
		s.mu.Unlock()
		select {
		case <-start.done:
			return start.serial, start.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	start := &androidStart{done: make(chan struct{})}
	s.starting[avd] = start
	s.mu.Unlock()

	serial, err := s.ensure(ctx, avd)

	s.mu.Lock()
	start.serial = serial
	start.err = err
	delete(s.starting, avd)
	close(start.done)
	s.mu.Unlock()
	return serial, err
}

func (s *AndroidSession) ensure(ctx context.Context, avd string) (string, error) {
	running, err := s.runningEmulators(ctx)
	if err != nil {
		return "", err
	}
	if serial := running[avd]; serial != "" {
		return s.waitForBoot(ctx, avd, serial, nil)
	}

	if err := s.ensureAVDExists(ctx, avd); err != nil {
		return "", err
	}
	owned, err := s.startHeadless(avd)
	if err != nil {
		return "", err
	}
	serial, err := s.waitForBoot(ctx, avd, "", owned)
	if err != nil {
		return "", errors.Join(err, s.stopOwned(owned))
	}
	s.mu.Lock()
	owned.serial = serial
	s.mu.Unlock()
	return serial, nil
}

// Command runs adb with the requested arguments after ensuring the AVD is
// running and boot complete. Arguments are appended after "-s SERIAL".
func (s *AndroidSession) Command(ctx context.Context, avd string, args ...string) ([]byte, error) {
	ctx, done, err := s.withSession(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	serial, err := s.Ensure(ctx, avd)
	if err != nil {
		return nil, err
	}
	adbArgs := append([]string{"-s", serial}, args...)
	output, err := s.deps.command(ctx, s.deps.adbPath, adbArgs...)
	if err != nil {
		return output, fmt.Errorf("adb command for %s: %w", avd, err)
	}
	return output, nil
}

// Release stops a headless emulator only when it was launched by this session.
// Emulators that were already running before Ensure are left alone.
func (s *AndroidSession) Release(avd string) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	start := s.starting[avd]
	owned := s.owned[avd]
	s.mu.Unlock()
	if start != nil {
		<-start.done
		s.mu.Lock()
		owned = s.owned[avd]
		s.mu.Unlock()
	}
	if err := s.stopOwned(owned); err != nil {
		return err
	}
	if owned == nil || owned.serial == "" {
		return nil
	}
	return s.waitForReleasedSerial(avd, owned.serial)
}

func (s *AndroidSession) waitForReleasedSerial(avd, serial string) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.deps.releaseTimeout)
	defer cancel()
	for {
		present, state, err := s.adbSerialState(ctx, serial)
		if err != nil {
			return err
		}
		if !present {
			return nil
		}
		if state == "device" {
			nameOutput, nameErr := s.deps.command(ctx, s.deps.adbPath, "-s", serial, "emu", "avd", "name")
			if nameErr == nil && parseAVDName(string(nameOutput)) != avd {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			if s.ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("wait for released Android emulator %q to disconnect: %w", avd, ctx.Err())
		case <-time.After(s.deps.pollInterval):
		}
	}
}

func (s *AndroidSession) adbSerialState(ctx context.Context, serial string) (bool, string, error) {
	output, err := s.deps.command(ctx, s.deps.adbPath, "devices", "-l")
	if err != nil {
		return false, "", fmt.Errorf("list Android devices while releasing emulator: %w", err)
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == serial {
			return true, fields[1], nil
		}
	}
	return false, "", nil
}

// Close stops and reaps only emulator processes launched by this session.
// It is safe to call more than once.
func (s *AndroidSession) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.cancel()
	owned := make([]*ownedAndroidEmulator, 0, len(s.owned))
	for _, emulator := range s.owned {
		owned = append(owned, emulator)
	}
	s.mu.Unlock()

	var errs []error
	for _, emulator := range owned {
		if err := s.stopOwned(emulator); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *AndroidSession) requireAvailable() error {
	if s == nil || !s.Available() {
		return errors.New("the Android SDK tools were not found; set ANDROID_HOME or ANDROID_SDK_ROOT")
	}
	return nil
}

func (s *AndroidSession) withSession(ctx context.Context) (context.Context, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-s.ctx.Done():
		return nil, nil, errors.New("the Android session is closed")
	default:
	}
	combined, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(s.ctx, cancel)
	return combined, func() {
		stop()
		cancel()
	}, nil
}

func (s *AndroidSession) listAVDs(ctx context.Context) ([]string, error) {
	output, err := s.deps.command(ctx, s.deps.emulatorPath, "-list-avds")
	if err != nil {
		return nil, fmt.Errorf("list Android AVDs: %w: %s", err, strings.TrimSpace(string(output)))
	}
	var avds []string
	for _, line := range strings.Split(string(output), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			avds = append(avds, name)
		}
	}
	return avds, nil
}

func (s *AndroidSession) ensureAVDExists(ctx context.Context, avd string) error {
	avds, err := s.listAVDs(ctx)
	if err != nil {
		return err
	}
	for _, candidate := range avds {
		if candidate == avd {
			return nil
		}
	}
	return fmt.Errorf("find Android AVD %q: not found", avd)
}

func (s *AndroidSession) runningEmulators(ctx context.Context) (map[string]string, error) {
	output, err := s.deps.command(ctx, s.deps.adbPath, "devices", "-l")
	if err != nil {
		return nil, fmt.Errorf("list Android devices: %w: %s", err, strings.TrimSpace(string(output)))
	}
	running := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "emulator-") {
			continue
		}
		if fields[1] != "device" {
			return nil, fmt.Errorf("the Android emulator %s is %s; wait for it to become available", fields[0], fields[1])
		}
		nameOutput, nameErr := s.deps.command(ctx, s.deps.adbPath, "-s", fields[0], "emu", "avd", "name")
		if nameErr != nil {
			return nil, fmt.Errorf("identify Android emulator %s: %w", fields[0], nameErr)
		}
		name := parseAVDName(string(nameOutput))
		if name == "" {
			return nil, fmt.Errorf("identify Android emulator %s: emulator did not return an AVD name", fields[0])
		}
		running[name] = fields[0]
	}
	return running, nil
}

func (s *AndroidSession) runtime(ctx context.Context, serial string) string {
	output, err := s.deps.command(ctx, s.deps.adbPath, "-s", serial, "shell", "getprop", "ro.build.version.release")
	if err != nil {
		return "Android (unknown)"
	}
	if version := strings.TrimSpace(string(output)); version != "" {
		return "Android " + version
	}
	return "Android (unknown)"
}

func (s *AndroidSession) startHeadless(avd string) (*ownedAndroidEmulator, error) {
	args := []string{"-avd", avd, "-no-window", "-no-snapshot-save"}
	if s.deps.readOnly {
		args = append(args, "-read-only")
	}
	cmd := s.deps.newCommand(s.deps.emulatorPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errors.New("the Android session is closed")
	}
	log := newRollingBuffer(8 << 10)
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("start Android AVD %q: %w", avd, err)
	}
	owned := &ownedAndroidEmulator{cmd: cmd, done: make(chan error, 1), log: log}
	s.owned[avd] = owned
	s.mu.Unlock()

	go func() {
		err := cmd.Wait()
		owned.done <- err
		close(owned.done)
		s.mu.Lock()
		if s.owned[avd] == owned {
			delete(s.owned, avd)
		}
		s.mu.Unlock()
	}()
	return owned, nil
}

func (s *AndroidSession) waitForBoot(ctx context.Context, avd, knownSerial string, owned *ownedAndroidEmulator) (string, error) {
	deadline, cancel := context.WithTimeout(ctx, s.deps.bootTimeout)
	defer cancel()
	for {
		if owned != nil {
			select {
			case err := <-owned.done:
				if err == nil {
					err = errors.New("emulator exited")
				}
				return "", withEmulatorLog(fmt.Errorf("the Android AVD %q exited before boot completed: %w", avd, err), owned)
			default:
			}
		}
		serial := knownSerial
		if serial == "" {
			running, err := s.runningEmulators(deadline)
			if err == nil {
				serial = running[avd]
			}
		}
		if serial != "" {
			output, err := s.deps.command(deadline, s.deps.adbPath, "-s", serial, "shell", "getprop", "sys.boot_completed")
			if err == nil && strings.TrimSpace(string(output)) == "1" {
				return serial, nil
			}
		}
		select {
		case <-deadline.Done():
			return "", withEmulatorLog(fmt.Errorf("wait for Android AVD %q to boot: %w", avd, deadline.Err()), owned)
		case <-time.After(s.deps.pollInterval):
		}
	}
}

func withEmulatorLog(err error, owned *ownedAndroidEmulator) error {
	if owned == nil || owned.log == nil {
		return err
	}
	output := strings.TrimSpace(owned.log.String())
	if output == "" {
		return err
	}
	return fmt.Errorf("%w\nemulator output:\n%s", err, output)
}

func (s *AndroidSession) stopOwned(owned *ownedAndroidEmulator) error {
	if owned == nil {
		return nil
	}
	owned.stopOnce.Do(func() {
		owned.stopErr = s.stopOwnedProcess(owned)
	})
	return owned.stopErr
}

func (s *AndroidSession) stopOwnedProcess(owned *ownedAndroidEmulator) error {
	if owned == nil || owned.cmd == nil || owned.cmd.Process == nil {
		return nil
	}
	select {
	case <-owned.done:
		return nil
	default:
	}

	pid := owned.cmd.Process.Pid
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("stop managed Android emulator: %w", err)
	}
	select {
	case <-owned.done:
		return nil
	case <-time.After(s.deps.shutdownTimeout):
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("force-stop managed Android emulator: %w", err)
	}
	select {
	case <-owned.done:
		return nil
	case <-time.After(s.deps.shutdownTimeout):
		return errors.New("managed Android emulator did not exit")
	}
}

func parseAVDName(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if before, after, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(before), "avd name") {
			return strings.TrimSpace(after)
		}
		if !strings.Contains(line, ":") {
			return line
		}
	}
	return ""
}

func runAndroidCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, androidCommandTimeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	stdout := &limitedBuffer{limit: maxAndroidCommandOutput}
	stderr := &limitedBuffer{limit: maxAndroidCommandOutput}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		if stdout.err != nil {
			return stdout.Bytes(), stdout.err
		}
		if stderr.err != nil {
			return stdout.Bytes(), stderr.err
		}
		return stdout.Bytes(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stdout.err != nil {
		return stdout.Bytes(), stdout.err
	}
	return stdout.Bytes(), nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
	err   error
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	remaining := b.limit - b.Len()
	if remaining <= 0 || len(p) > remaining {
		if remaining > 0 {
			_, _ = b.Buffer.Write(p[:remaining])
		}
		b.err = fmt.Errorf("command output exceeds %d MiB", b.limit>>20)
		return 0, b.err
	}
	return b.Buffer.Write(p)
}

var _ io.Writer = (*limitedBuffer)(nil)

type rollingBuffer struct {
	mu    sync.Mutex
	data  []byte
	limit int
}

func newRollingBuffer(limit int) *rollingBuffer {
	return &rollingBuffer{limit: limit}
}

func (b *rollingBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(data) >= b.limit {
		b.data = append(b.data[:0], data[len(data)-b.limit:]...)
		return len(data), nil
	}
	if overflow := len(b.data) + len(data) - b.limit; overflow > 0 {
		copy(b.data, b.data[overflow:])
		b.data = b.data[:len(b.data)-overflow]
	}
	b.data = append(b.data, data...)
	return len(data), nil
}

func (b *rollingBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}

var _ io.Writer = (*rollingBuffer)(nil)

func findAndroidTools() (string, string) {
	var roots []string
	for _, value := range []string{os.Getenv("ANDROID_HOME"), os.Getenv("ANDROID_SDK_ROOT")} {
		if value != "" {
			roots = append(roots, value)
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, "Library", "Android", "sdk"))
	}
	for _, root := range roots {
		adb := filepath.Join(root, "platform-tools", "adb")
		emulator := filepath.Join(root, "emulator", "emulator")
		if isExecutable(adb) && isExecutable(emulator) {
			return adb, emulator
		}
	}
	adb, adbErr := exec.LookPath("adb")
	emulator, emulatorErr := exec.LookPath("emulator")
	if adbErr == nil && emulatorErr == nil {
		return adb, emulator
	}
	return "", ""
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}
