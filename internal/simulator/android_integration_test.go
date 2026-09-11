package simulator

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The selected stopped AVD runs with a disposable, read-only disk overlay.
func TestAndroidBrowserIntegration(t *testing.T) {
	avd := os.Getenv("SIMTOOL_ANDROID_AVD")
	if avd == "" {
		t.Skip("set SIMTOOL_ANDROID_AVD to test a disposable headless session")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	b, err := NewBrowser(ctx, "android")
	if err != nil {
		t.Fatal(err)
	}
	b.android.deps.readOnly = true
	before, err := b.android.runningEmulators(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before[avd] != "" {
		_ = b.Close()
		t.Fatalf("test AVD %s must be stopped", avd)
	}
	cache := b.cache
	t.Cleanup(func() {
		if err := b.Close(); err != nil {
			t.Error(err)
		}
		if _, err := os.Stat(cache); !os.IsNotExist(err) {
			t.Errorf("private preview cache remains after close: %v", err)
		}
		probe, err := NewAndroidSession(context.Background())
		if err != nil {
			t.Error(err)
			return
		}
		defer probe.Close()
		var after map[string]string
		for attempt := 0; attempt < 20; attempt++ {
			after, err = probe.runningEmulators(context.Background())
			if err == nil && after[avd] == "" {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if err != nil {
			t.Error(err)
			return
		}
		if after[avd] != "" {
			t.Errorf("managed emulator %s remains running", avd)
		}
		for name, serial := range before {
			if after[name] != serial {
				t.Errorf("pre-existing %s changed from %s to %s", name, serial, after[name])
			}
		}
	})
	apk := buildAndroidFixture(ctx, t, b.aapt)
	if _, err := b.android.Command(ctx, avd, "install", apk); err != nil {
		t.Fatal(err)
	}
	const pkg = "com.example.simtool.fixture"
	setup := `mkdir -p files databases && printf 'hello Android\n' > 'files/a quote '\'' and space.txt'`
	if _, err := b.androidDataCommand(ctx, androidPath{avd: avd, pkg: pkg}, setup); err != nil {
		t.Fatal(err)
	}
	item := Item{Simulator: Simulator{UDID: androidID(avd), Platform: "android", Name: avd, State: "Booted"}}
	apps, err := b.Apps(item)
	if err != nil {
		t.Fatal(err)
	}
	var fixture *App
	for i := range apps {
		if apps[i].BundleID == pkg {
			fixture = &apps[i]
			break
		}
	}
	if fixture == nil {
		t.Fatal("installed fixture not discovered")
	}
	if fixture.Name != "SimTool Fixture" || fixture.Version != "1.0" || fixture.Size <= 0 || fixture.Access != "Private data accessible" {
		t.Fatalf("incorrect app metadata: %+v", fixture)
	}
	files, err := b.Files(fixture.Container + "/data/files")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "a quote ' and space.txt" {
		t.Fatalf("files = %+v", files)
	}
	local, err := b.Prepare(files[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(local)
	if err != nil || string(contents) != "hello Android\n" {
		t.Fatalf("preview = %q, %v", contents, err)
	}
	if mode, err := os.Stat(local); err != nil || mode.Mode().Perm() != 0600 {
		t.Fatalf("private file permissions: %v", err)
	}
	archive, err := b.Prepare(fixture.Path)
	if err != nil {
		t.Fatal(err)
	}
	if DetectFileType(archive) != FileTypeArchive {
		t.Fatal("APK was not recognized as an archive")
	}

	// Leave committed rows in the WAL, matching an app with an open database.
	dbPath := filepath.Join(t.TempDir(), "fixture.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0; CREATE TABLE notes (value TEXT); INSERT INTO notes VALUES ('committed in WAL');"); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "-wal"} {
		remote := "/data/local/tmp/simtool-fixture.db" + suffix
		if _, err := b.android.Command(ctx, avd, "push", dbPath+suffix, remote); err != nil {
			t.Fatal(err)
		}
		if _, err := b.androidShell(ctx, avd, "chmod 644 "+shellQuote(remote)); err != nil {
			t.Fatal(err)
		}
		if _, err := b.androidDataCommand(ctx, androidPath{avd: avd, pkg: pkg}, "cp "+shellQuote(remote)+" "+shellQuote("databases/fixture.db"+suffix)); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := b.Prepare(fixture.Container + "/data/databases/fixture.db")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ReadTableData(snapshot, "notes", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["value"] != "committed in WAL" {
		t.Fatalf("WAL row missing: %#v", rows)
	}
	if err := b.OpenInFinder(fixture.Container + "/data/files"); err != nil {
		t.Fatal(err)
	}
	if len(b.mounts) != 1 {
		t.Fatal("Finder mount was not tracked")
	}
	mounted := filepath.Join(b.mounts[0], "data", "files", files[0].Name)
	contents, err = os.ReadFile(mounted)
	if err != nil || !strings.Contains(string(contents), "hello Android") {
		t.Fatalf("Finder read failed: %v", err)
	}
}

func buildAndroidFixture(ctx context.Context, t *testing.T, aapt string) string {
	t.Helper()
	if aapt == "" {
		t.Fatal("the integration test requires Android SDK build-tools and JDK keytool")
	}
	dir := t.TempDir()
	manifest := filepath.Join(dir, "AndroidManifest.xml")
	contents := `<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="com.example.simtool.fixture" android:versionCode="1" android:versionName="1.0"><uses-sdk android:minSdkVersion="26" android:targetSdkVersion="36"/><application android:label="SimTool Fixture" android:debuggable="true" android:hasCode="false"/></manifest>`
	if err := os.WriteFile(manifest, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	sdk := filepath.Dir(filepath.Dir(filepath.Dir(aapt)))
	platforms, err := filepath.Glob(filepath.Join(sdk, "platforms", "*", "android.jar"))
	if err != nil || len(platforms) == 0 {
		t.Fatal("install an Android SDK platform for the integration test")
	}
	unsigned := filepath.Join(dir, "unsigned.apk")
	key := filepath.Join(dir, "fixture.keystore")
	signed := filepath.Join(dir, "fixture.apk")
	commands := [][]string{
		{aapt, "link", "-o", unsigned, "--manifest", manifest, "-I", platforms[len(platforms)-1]},
		{"keytool", "-genkeypair", "-keystore", key, "-storepass", "android", "-keypass", "android", "-alias", "fixture", "-keyalg", "RSA", "-keysize", "2048", "-validity", "2", "-dname", "CN=SimToolFixture"},
		{filepath.Join(filepath.Dir(aapt), "apksigner"), "sign", "--ks", key, "--ks-pass", "pass:android", "--out", signed, unsigned},
	}
	for _, args := range commands {
		if output, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("build temporary APK fixture: %v: %s", err, output)
		}
	}
	return signed
}
