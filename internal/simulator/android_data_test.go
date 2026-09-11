package simulator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAndroidVirtualPathsAndShellQuote(t *testing.T) {
	container := androidContainer("Pixel 10", "com.example.app")
	parsed, err := parseAndroidPath(container + "/data/Documents")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.avd != "Pixel 10" || parsed.pkg != "com.example.app" || parsed.area != "data" || parsed.relative != "Documents" {
		t.Fatalf("parsed path = %#v", parsed)
	}
	for _, invalid := range []string{
		"/@android/%%%/com.example.app",
		container + "/data/../escape",
		container + "/other",
		container + "/data/file\x00name",
		container + "/data/",
	} {
		if _, err := parseAndroidPath(invalid); err == nil {
			t.Errorf("parseAndroidPath(%q) succeeded", invalid)
		}
	}
	if got, want := shellQuote("a b'$(bad)"), `'a b'"'"'$(bad)'`; got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
	if _, err := parseAndroidPath(container + "/data/file\nname"); err != nil {
		t.Fatalf("newline filename: %v", err)
	}
}

func TestParseAndroidFileListPreservesUnusualNames(t *testing.T) {
	data := []byte("81a4 4 100\n./space name.txt\x00" +
		"81a4 5 101\n./mön'quoted'.txt\x00" +
		"41ed 0 102\n./line\nbreak\x00")
	files, err := parseAndroidFileList(data, "/@android/container/com.example.app/data")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("files = %#v", files)
	}
	if files[0].Name != "space name.txt" || files[1].Name != "mön'quoted'.txt" || files[2].Name != "line\nbreak" {
		t.Fatalf("names = %#v", files)
	}
	if !files[2].IsDirectory || files[2].Size != -1 {
		t.Fatalf("directory = %#v", files[2])
	}
	if files[1].Path != "/@android/container/com.example.app/data/mön'quoted'.txt" {
		t.Fatalf("path = %q", files[1].Path)
	}
}

func TestParseAndroidStatRejectsLinksAndSpecialFiles(t *testing.T) {
	regular, err := parseAndroidStat("81a4 12 100")
	if err != nil || regular.IsDirectory || regular.Size != 12 {
		t.Fatalf("regular = %#v, %v", regular, err)
	}
	directory, err := parseAndroidStat("41ed 99 100")
	if err != nil || !directory.IsDirectory || directory.Size != -1 {
		t.Fatalf("directory = %#v, %v", directory, err)
	}
	for _, value := range []string{"a1ff 1 100", "21b6 0 100", "81a4 -1 100", "invalid"} {
		if _, err := parseAndroidStat(value); err == nil {
			t.Errorf("parseAndroidStat(%q) succeeded", value)
		}
	}
}

func TestAndroidAPKMappingAndPrivateRestriction(t *testing.T) {
	var commands []string
	browser := newAndroidDataTestBrowser(t, func(command string) ([]byte, error) {
		commands = append(commands, command)
		switch {
		case command == "pm path --user 0 'com.example.app'":
			return []byte("package:/data/app/~~id/com.example.app-base/base.apk\npackage:/data/app/~~id/com.example.app-base/split_config.en.apk\n"), nil
		case strings.HasPrefix(command, "stat -c"):
			return []byte("81a4 42 100\n"), nil
		case command == "id -u":
			return []byte("2000\n"), nil
		case strings.HasPrefix(command, "run-as 'com.example.app' sh -c "):
			return nil, errors.New("package is not debuggable")
		default:
			return nil, fmt.Errorf("unexpected Android shell command %q", command)
		}
	})
	container := androidContainer("Pixel", "com.example.app")
	files, err := browser.androidFiles(context.Background(), container+"/apk")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Name != "base.apk" || files[1].Name != "split_config.en.apk" {
		t.Fatalf("APK files = %#v", files)
	}
	if files[0].Path != container+"/apk/base.apk" {
		t.Fatalf("base APK path = %q", files[0].Path)
	}
	_, err = browser.androidDataCommand(context.Background(), androidPath{avd: "Pixel", pkg: "com.example.app"}, "pwd")
	if err == nil || !strings.Contains(err.Error(), "private data for com.example.app is unavailable") {
		t.Fatalf("private restriction error = %v", err)
	}
	if !containsCommand(commands, "run-as 'com.example.app' sh -c ") {
		t.Fatalf("commands did not use run-as: %#v", commands)
	}
}

func TestAndroidPrepareRetainsCommittedSQLiteWALRows(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "db.sqlite")
	source, err := sql.Open("sqlite3", sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err := source.Exec("PRAGMA journal_mode=WAL; CREATE TABLE entries (value TEXT); INSERT INTO entries VALUES ('committed in wal')"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(sourcePath + "-wal"); err != nil || info.Size() == 0 {
		t.Fatalf("expected non-empty source WAL: %v", err)
	}

	browser := newAndroidDataTestBrowser(t, androidLocalSQLiteResponder(t, sourcePath, false))
	container := androidContainer("Pixel", "com.example.app")
	local, err := browser.androidPrepare(context.Background(), container+"/data/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	copyDB, err := sql.Open("sqlite3", "file:"+local+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()
	var value string
	if err := copyDB.QueryRow("SELECT value FROM entries").Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "committed in wal" {
		t.Fatalf("snapshot row = %q", value)
	}
}

func TestAndroidPrepareFailsWhenSQLiteChangesDuringTransfer(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "db.sqlite")
	if err := os.WriteFile(sourcePath, append([]byte("SQLite format 3\x00"), make([]byte, 4096)...), 0600); err != nil {
		t.Fatal(err)
	}
	browser := newAndroidDataTestBrowser(t, androidLocalSQLiteResponder(t, sourcePath, true))
	container := androidContainer("Pixel", "com.example.app")
	_, err := browser.androidPrepare(context.Background(), container+"/data/db.sqlite")
	if err == nil || !strings.Contains(err.Error(), "Android database is changing during transfer") {
		t.Fatalf("changing snapshot error = %v", err)
	}
}

func newAndroidDataTestBrowser(t *testing.T, responder func(string) ([]byte, error)) *DeviceBrowser {
	t.Helper()
	fake := &fakeAndroid{avds: []string{"Pixel"}, running: map[string]string{"Pixel": "emulator-5554"}, booted: true}
	session := newTestAndroidSession(fake, nil)
	baseCommand := session.deps.command
	session.deps.command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "adb" && len(args) == 5 && args[0] == "-s" && args[2] == "shell" && args[3] == "-T" {
			return responder(args[4])
		}
		return baseCommand(ctx, name, args...)
	}
	t.Cleanup(func() { _ = session.Close() })
	return &DeviceBrowser{ctx: context.Background(), android: session, cache: t.TempDir(), prepared: make(map[string]string), counts: make(map[string]int)}
}

func androidLocalSQLiteResponder(t *testing.T, sourcePath string, changing bool) func(string) ([]byte, error) {
	t.Helper()
	reads := 0
	return func(command string) ([]byte, error) {
		switch {
		case command == "id -u":
			return []byte("2000\n"), nil
		case strings.Contains(command, "if [ -f"):
			suffix := androidSQLiteSuffix(command)
			if _, err := os.Stat(sourcePath + suffix); err == nil {
				return []byte("yes\n"), nil
			}
			return []byte("no\n"), nil
		case strings.Contains(command, "stat -c"):
			suffix := androidSQLiteSuffix(command)
			info, err := os.Stat(sourcePath + suffix)
			if err != nil {
				return nil, err
			}
			return []byte(fmt.Sprintf("81a4 %d %d\n", info.Size(), info.ModTime().Unix())), nil
		case strings.Contains(command, "head -c"):
			suffix := androidSQLiteSuffix(command)
			data, err := os.ReadFile(sourcePath + suffix)
			if err != nil {
				return nil, err
			}
			if changing && suffix == "" {
				reads++
				if reads > 1 {
					data[len(data)-1] = byte(reads)
				}
			}
			return data, nil
		default:
			return nil, fmt.Errorf("unexpected Android shell command %q", command)
		}
	}
}

func androidSQLiteSuffix(command string) string {
	if strings.Contains(command, "db.sqlite-wal") {
		return "-wal"
	}
	if strings.Contains(command, "db.sqlite-journal") {
		return "-journal"
	}
	return ""
}

func containsCommand(commands []string, prefix string) bool {
	for _, command := range commands {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}
