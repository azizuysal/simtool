package simulator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const androidPathPrefix = "/@android/"

var androidPackagePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z0-9_]+)+$`)

type androidPath struct{ avd, pkg, area, relative string }

func isAndroidPath(name string) bool { return strings.HasPrefix(name, androidPathPrefix) }

func androidContainer(avd, pkg string) string {
	return androidPathPrefix + base64.RawURLEncoding.EncodeToString([]byte(avd)) + "/" + pkg
}

func parseAndroidPath(name string) (androidPath, error) {
	var result androidPath
	if !isAndroidPath(name) || path.Clean(name) != name || strings.ContainsRune(name, '\x00') {
		return result, errors.New("invalid Android path")
	}
	parts := strings.Split(strings.TrimPrefix(name, androidPathPrefix), "/")
	if len(parts) < 2 || !androidPackagePattern.MatchString(parts[1]) {
		return result, errors.New("invalid Android package path")
	}
	avd, err := androidName("android:" + parts[0])
	if err != nil {
		return result, err
	}
	result.avd = avd
	result.pkg = parts[1]
	if len(parts) > 2 {
		result.area = parts[2]
		if result.area != "data" && result.area != "apk" {
			return result, errors.New("unknown Android app folder")
		}
		result.relative = strings.Join(parts[3:], "/")
	}
	return result, nil
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

func (b *DeviceBrowser) androidShell(ctx context.Context, avd, command string) ([]byte, error) {
	if b.android == nil {
		return nil, errors.New("the Android SDK is unavailable")
	}
	return b.android.Command(ctx, avd, "shell", "-T", command)
}

func (b *DeviceBrowser) androidPackages(ctx context.Context, avd string) ([]string, error) {
	out, err := b.androidShell(ctx, avd, "cmd package list packages -3 --user 0")
	if err != nil {
		return nil, fmt.Errorf("list Android apps: %w", err)
	}
	var packages []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pkg := strings.TrimPrefix(strings.TrimSpace(line), "package:")
		if pkg == "" {
			continue
		}
		if !androidPackagePattern.MatchString(pkg) {
			return nil, fmt.Errorf("invalid Android package list entry %q", pkg)
		}
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	return packages, nil
}

func (b *DeviceBrowser) androidApps(ctx context.Context, avd string) ([]App, error) {
	packages, err := b.androidPackages(ctx, avd)
	if err != nil {
		return nil, err
	}
	apps := make([]App, 0, len(packages))
	var warnings []error
	for _, pkg := range packages {
		out, err := b.androidShell(ctx, avd, "dumpsys package "+shellQuote(pkg))
		if err != nil {
			return apps, err
		}
		app := App{Platform: "android", Name: pkg, BundleID: pkg, Container: androidContainer(avd, pkg), Access: "Private data restricted"}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if value, ok := strings.CutPrefix(line, "versionName="); ok {
				app.Version = value
			}
		}
		if app.Version == "" {
			app.Version = "unknown"
		}
		if _, err := b.androidDataCommand(ctx, androidPath{avd: avd, pkg: pkg}, "pwd"); err == nil {
			app.Access = "Private data accessible"
		} else if ctx.Err() != nil {
			return apps, ctx.Err()
		}
		apks, err := b.androidAPKs(ctx, avd, pkg)
		if err != nil {
			return apps, err
		}
		for _, apk := range apks {
			info, err := b.androidStat(ctx, avd, "", apk)
			if err != nil {
				return apps, err
			}
			app.Size += info.Size
			if info.ModifiedAt.After(app.ModTime) {
				app.ModTime = info.ModifiedAt
			}
			if path.Base(apk) == "base.apk" {
				app.Path = app.Container + "/apk/base.apk"
			}
		}
		// aapt2 resolves resource-backed display names from the installed APK.
		if aapt := b.aapt; aapt != "" && app.Path != "" {
			local, err := b.Prepare(app.Path)
			if err != nil {
				warnings = append(warnings, fmt.Errorf("%s label: %w", pkg, err))
				apps = append(apps, app)
				continue
			}
			badging, err := exec.CommandContext(ctx, aapt, "dump", "badging", local).Output()
			if err != nil {
				warnings = append(warnings, fmt.Errorf("%s label: %w", pkg, err))
				apps = append(apps, app)
				continue
			}
			for _, line := range strings.Split(string(badging), "\n") {
				if value, ok := strings.CutPrefix(line, "application-label:'"); ok {
					app.Name = strings.TrimSuffix(value, "'")
					break
				}
				if value, ok := strings.CutPrefix(line, "application: label='"); ok {
					if label, _, found := strings.Cut(value, "' icon='"); found && label != "" {
						app.Name = label
					}
				}
			}
		}
		apps = append(apps, app)
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	return apps, errors.Join(warnings...)
}

func findAAPT2() string {
	if found, err := exec.LookPath("aapt2"); err == nil {
		return found
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	roots := []string{os.Getenv("ANDROID_HOME"), os.Getenv("ANDROID_SDK_ROOT"), filepath.Join(home, "Library", "Android", "sdk")}
	for _, root := range roots {
		if root == "" {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(root, "build-tools", "*", "aapt2"))
		if err != nil {
			continue
		}
		sort.Sort(sort.Reverse(sort.StringSlice(matches)))
		for _, candidate := range matches {
			if info, err := os.Stat(candidate); err == nil && info.Mode()&0111 != 0 {
				return candidate
			}
		}
	}
	return ""
}

func (b *DeviceBrowser) androidAPKs(ctx context.Context, avd, pkg string) ([]string, error) {
	if !androidPackagePattern.MatchString(pkg) {
		return nil, errors.New("invalid Android package")
	}
	out, err := b.androidShell(ctx, avd, "pm path --user 0 "+shellQuote(pkg))
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name := strings.TrimPrefix(strings.TrimSpace(line), "package:")
		if !strings.HasPrefix(name, "/data/app/") || path.Clean(name) != name || path.Ext(name) != ".apk" || strings.ContainsAny(name, "\x00\r\n") {
			return nil, fmt.Errorf("unexpected installed APK path %q", name)
		}
		paths = append(paths, name)
	}
	if len(paths) == 0 {
		return nil, errors.New("app has no installed APKs")
	}
	return paths, nil
}

// Each command runs as the app's own UID when run-as is available. Root is used
// only when adbd is already running as root; SimTool never enables it.
func (b *DeviceBrowser) androidDataCommand(ctx context.Context, p androidPath, script string) ([]byte, error) {
	root := "/data/user/0/" + p.pkg
	safeScript := "cd " + shellQuote(root) + " && " + script
	command := "run-as " + shellQuote(p.pkg) + " sh -c " + shellQuote(safeScript)
	identity, err := b.androidShell(ctx, p.avd, "id -u")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(identity)) == "0" {
		command = "sh -c " + shellQuote(safeScript)
	}
	out, err := b.androidShell(ctx, p.avd, command)
	if err != nil {
		return nil, fmt.Errorf("private data for %s is unavailable; use a debuggable app or an emulator with root already enabled: %w", p.pkg, err)
	}
	return out, nil
}

func confinedAndroidScript(relative, script string) string {
	if relative == "" {
		relative = "."
	}
	// Reject symlinks in every traversed component and require a path inside the
	// selected app. This also avoids opening devices, sockets and named pipes.
	var checks []string
	current := "."
	for _, part := range strings.Split(relative, "/") {
		current = path.Join(current, part)
		checks = append(checks, "[ ! -L "+shellQuote(current)+" ]")
	}
	return strings.Join(checks, " && ") + " && " + script
}

func (b *DeviceBrowser) androidFiles(ctx context.Context, name string) ([]FileInfo, error) {
	p, err := parseAndroidPath(name)
	if err != nil {
		return nil, err
	}
	if p.area == "" {
		return []FileInfo{{Name: "data", Path: name + "/data", IsDirectory: true, Size: -1}, {Name: "apk", Path: name + "/apk", IsDirectory: true, Size: -1}}, nil
	}
	if p.area == "apk" {
		if p.relative != "" {
			return nil, errors.New("APK is not a directory")
		}
		paths, err := b.androidAPKs(ctx, p.avd, p.pkg)
		if err != nil {
			return nil, err
		}
		var files []FileInfo
		for _, apk := range paths {
			info, err := b.androidStat(ctx, p.avd, "", apk)
			if err != nil {
				return nil, err
			}
			info.Name = path.Base(apk)
			info.Path = name + "/" + info.Name
			files = append(files, info)
		}
		return files, nil
	}
	relative := p.relative
	if relative == "" {
		relative = "."
	}
	// NUL-delimited records preserve spaces, tabs and newlines in filenames.
	script := confinedAndroidScript(relative, "cd "+shellQuote(relative)+` && find . -mindepth 1 -maxdepth 1 ! -type l -exec sh -c 'for f do [ -f "$f" ] || [ -d "$f" ] || continue; stat -c "%f %s %Y" "$f" || exit; printf "%s\000" "$f"; done' sh {} +`)
	out, err := b.androidDataCommand(ctx, p, script)
	if err != nil {
		return nil, err
	}
	files, err := parseAndroidFileList(out, name)
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDirectory != files[j].IsDirectory {
			return files[i].IsDirectory
		}
		return files[i].Name < files[j].Name
	})
	return files, nil
}

func parseAndroidFileList(data []byte, parent string) ([]FileInfo, error) {
	var files []FileInfo
	for len(data) > 0 {
		line, rest, ok := bytes.Cut(data, []byte{'\n'})
		if !ok {
			return nil, errors.New("invalid Android file metadata")
		}
		name, tail, ok := bytes.Cut(rest, []byte{0})
		if !ok {
			return nil, errors.New("invalid Android filename record")
		}
		info, err := parseAndroidStat(string(line))
		if err != nil {
			return nil, err
		}
		info.Name = strings.TrimPrefix(string(name), "./")
		if info.Name == "" || info.Name == "." || info.Name == ".." || strings.Contains(info.Name, "/") {
			return nil, errors.New("invalid Android filename")
		}
		info.Path = parent + "/" + info.Name
		files = append(files, info)
		data = tail
	}
	return files, nil
}

func parseAndroidStat(value string) (FileInfo, error) {
	var info FileInfo
	fields := strings.Fields(value)
	if len(fields) != 3 {
		return info, errors.New("invalid Android stat output")
	}
	mode, err := strconv.ParseUint(fields[0], 16, 32)
	if err != nil {
		return info, err
	}
	info.Size, err = strconv.ParseInt(fields[1], 10, 64)
	if err != nil || info.Size < 0 {
		return info, errors.New("invalid Android file size")
	}
	modified, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return info, err
	}
	info.ModifiedAt = time.Unix(modified, 0)
	switch mode & 0170000 {
	case 0040000:
		info.IsDirectory = true
		info.Size = -1
	case 0100000:
	default:
		return info, errors.New("only regular Android files and directories can be browsed")
	}
	return info, nil
}

func (b *DeviceBrowser) androidStat(ctx context.Context, avd, pkg, name string) (FileInfo, error) {
	command := "stat -c '%f %s %Y' " + shellQuote(name)
	var out []byte
	var err error
	if pkg == "" {
		out, err = b.androidShell(ctx, avd, command)
	} else {
		out, err = b.androidDataCommand(ctx, androidPath{avd: avd, pkg: pkg}, confinedAndroidScript(name, command))
	}
	if err != nil {
		return FileInfo{}, err
	}
	return parseAndroidStat(string(out))
}

func (b *DeviceBrowser) androidRead(ctx context.Context, p androidPath, remote string) ([]byte, error) {
	info, err := b.androidStat(ctx, p.avd, p.pkg, remote)
	if err != nil {
		return nil, err
	}
	if info.IsDirectory {
		return nil, errors.New("cannot preview a directory")
	}
	if info.Size > 256<<20 {
		return nil, errors.New("the Android preview exceeds the 256 MiB transfer limit")
	}
	command := "[ -f " + shellQuote(remote) + " ] && head -c 268435457 " + shellQuote(remote)
	var data []byte
	if p.pkg == "" {
		data, err = b.androidShell(ctx, p.avd, command)
	} else {
		data, err = b.androidDataCommand(ctx, p, confinedAndroidScript(remote, command))
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 256<<20 {
		return nil, errors.New("the Android preview exceeds the 256 MiB transfer limit")
	}
	return data, nil
}

func (b *DeviceBrowser) androidPrepare(ctx context.Context, name string) (string, error) {
	p, err := parseAndroidPath(name)
	if err != nil {
		return "", err
	}
	if p.relative == "" {
		return "", errors.New("select an Android file to preview")
	}
	remote := p.relative
	cacheKey := ""
	if p.area == "apk" {
		paths, err := b.androidAPKs(ctx, p.avd, p.pkg)
		if err != nil {
			return "", err
		}
		remote = ""
		for _, apk := range paths {
			if path.Base(apk) == p.relative {
				remote = apk
				break
			}
		}
		if remote == "" {
			return "", os.ErrNotExist
		}
		// Package Manager installs each APK version in a new directory.
		cacheKey = androidID(p.avd) + remote
		if local, ok := b.prepared[cacheKey]; ok {
			return local, nil
		}
		p.pkg = ""
	}
	dir, err := os.MkdirTemp(b.cache, "preview-")
	if err != nil {
		return "", err
	}
	local := filepath.Join(dir, path.Base(remote))
	data, err := b.androidRead(ctx, p, remote)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(local, data, 0600); err != nil {
		return "", err
	}
	if p.area == "data" && bytes.HasPrefix(data, []byte("SQLite format 3\x00")) {
		if err := b.androidDatabaseCopy(ctx, p, remote, local, data); err != nil {
			return "", err
		}
	}
	if cacheKey != "" {
		b.prepared[cacheKey] = local
	}
	return local, nil
}

// Retain journals and reject changing or invalid copies. This is a validated
// inspection copy; a transactional backup requires SQLite's cooperation.
func (b *DeviceBrowser) androidDatabaseCopy(ctx context.Context, p androidPath, remote, local string, first []byte) error {
	for attempt := 0; attempt < 3; attempt++ {
		captured := map[string][]byte{"": first}
		for _, suffix := range []string{"-wal", "-journal"} {
			present, err := b.androidDataCommand(ctx, p, confinedAndroidScript(remote+suffix, "if [ -f "+shellQuote(remote+suffix)+" ]; then echo yes; else echo no; fi"))
			if err != nil {
				return err
			}
			if strings.TrimSpace(string(present)) == "yes" {
				data, err := b.androidRead(ctx, p, remote+suffix)
				if err != nil {
					return err
				}
				captured[suffix] = data
			}
		}
		stable := true
		for _, suffix := range []string{"", "-wal", "-journal"} {
			present, err := b.androidDataCommand(ctx, p, confinedAndroidScript(remote+suffix, "if [ -f "+shellQuote(remote+suffix)+" ]; then echo yes; else echo no; fi"))
			if err != nil {
				return err
			}
			previous, had := captured[suffix]
			if (strings.TrimSpace(string(present)) == "yes") != had {
				stable = false
				break
			}
			if had {
				data, err := b.androidRead(ctx, p, remote+suffix)
				if err != nil {
					return err
				}
				if sha256.Sum256(data) != sha256.Sum256(previous) {
					stable = false
					break
				}
			}
		}
		if stable {
			for suffix, data := range captured {
				if err := os.WriteFile(local+suffix, data, 0600); err != nil {
					return err
				}
			}
			db, err := openReadOnlyDB(local)
			if err != nil {
				return err
			}
			var check string
			checkErr := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&check)
			closeErr := db.Close()
			if checkErr != nil {
				return fmt.Errorf("validate Android database copy: %w", checkErr)
			}
			if closeErr != nil {
				return closeErr
			}
			if check != "ok" {
				return fmt.Errorf("the Android database copy failed validation: %s", check)
			}
			return nil
		}
		var err error
		first, err = b.androidRead(ctx, p, remote)
		if err != nil {
			return err
		}
	}
	return errors.New("the Android database is changing during transfer; stop activity in the app and reopen the file")
}
