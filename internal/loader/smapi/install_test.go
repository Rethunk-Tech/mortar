package smapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/meta"

	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func recorded(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/releases.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestLatestFromRecordedResponseIsCached(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(recorded(t))
	}))
	defer srv.Close()
	g := Loader{ReleasesURL: srv.URL, AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir()}
	for range 2 {
		v, err := g.Latest(context.Background(), components.GameInfo{})
		if err != nil || v != "4.5.2" {
			t.Fatalf("latest = %q, %v", v, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("api hit %d times, want 1", hits.Load())
	}
}

func TestLatestSkipsPrereleaseAndUsesStaleCacheOnFailure(t *testing.T) {
	body := `[{"tag_name":"5.0.0-beta","prerelease":true,"assets":[{"name":"SMAPI-5.0.0-beta-installer.zip"}]},
	{"tag_name":"4.9.0","assets":[{"name":"SMAPI-4.9.0-installer-for-developers.zip"}]},
	{"tag_name":"4.8.0","assets":[{"name":"SMAPI-4.8.0-installer.zip"}]}]`
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	g := Loader{ReleasesURL: srv.URL, AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir()}
	if v, err := g.Latest(context.Background(), components.GameInfo{}); err != nil || v != "4.8.0" {
		t.Fatalf("latest = %q, %v", v, err)
	}
	fail = true
	path, _ := g.cachePath()
	old := time.Now().Add(-2 * cacheTTL)
	c, _ := readCache(path)
	c.Fetched = old
	if err := writeCache(path, c); err != nil {
		t.Fatal(err)
	}
	if v, err := g.Latest(context.Background(), components.GameInfo{}); err != nil || v != "4.8.0" {
		t.Fatalf("stale fallback = %q, %v", v, err)
	}
	if _, err := (Loader{ReleasesURL: srv.URL, AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir()}).Latest(context.Background(), components.GameInfo{}); err == nil {
		t.Fatal("no cache and a failing API must error")
	}
}

func TestRateLimitMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ratelimit-Remaining", "0")
		w.Header().Set("X-Ratelimit-Reset", "1790000000")
		http.Error(w, `{"message":"API rate limit exceeded"}`, http.StatusForbidden)
	}))
	defer srv.Close()
	_, err := Loader{ReleasesURL: srv.URL, AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir()}.Latest(context.Background(), components.GameInfo{})
	if err == nil || !strings.Contains(err.Error(), "rate limit") || !strings.Contains(err.Error(), "try again after") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoaderState(t *testing.T) {
	dir := t.TempDir()
	if i, b := loaderState(dir, "linux"); i || b {
		t.Fatal("vanilla install reported as SMAPI")
	}
	write(t, filepath.Join(dir, linuxOriginal), "native")
	write(t, filepath.Join(dir, linuxLauncher), "#!/bin/sh\nexec StardewModdingAPI\n")
	write(t, filepath.Join(dir, "StardewModdingAPI.dll"), "")
	if i, b := loaderState(dir, "linux"); !i || b {
		t.Fatalf("installed = %v, broken = %v", i, b)
	}
	write(t, filepath.Join(dir, linuxLauncher), "vanilla launcher after a game update")
	if i, b := loaderState(dir, "linux"); i || !b {
		t.Fatalf("after update: installed = %v, broken = %v", i, b)
	}
	if err := os.Remove(filepath.Join(dir, linuxOriginal)); err != nil {
		t.Fatal(err)
	}
	if i, b := loaderState(dir, "linux"); i || !b {
		t.Fatalf("smapi files with a vanilla launcher: installed = %v, broken = %v", i, b)
	}

	win := t.TempDir()
	if i, b := loaderState(win, "windows"); i || b {
		t.Fatal("windows vanilla")
	}
	write(t, filepath.Join(win, "StardewModdingAPI.exe"), "")
	if i, b := loaderState(win, "windows"); !i || b {
		t.Fatal("windows installed")
	}
}

func TestVersionsFromLogAndRecordedWins(t *testing.T) {
	logs := t.TempDir()
	write(t, filepath.Join(logs, "SMAPI-latest.txt"),
		"SMAPI 4.5.2 with Stardew Valley 1.6.15 build 24356 on Unix 6.1\n[00:00:00 TRACE SMAPI] later line\n")
	g := Loader{LogDir: logs}
	if s, gv := g.logVersions(); s != "4.5.2" || gv != "1.6.15" {
		t.Fatalf("log = %q %q", s, gv)
	}
	if s, gv := (Loader{LogDir: t.TempDir()}).logVersions(); s != "" || gv != "" {
		t.Fatalf("missing log = %q %q", s, gv)
	}
}

func TestInstallerPathsAndNewer(t *testing.T) {
	if d, e, _ := installer("4.5.2", "windows"); filepath.ToSlash(d) != "SMAPI 4.5.2 installer/internal/windows" || e != "SMAPI.Installer.exe" {
		t.Fatalf("windows = %q %q", d, e)
	}
	if _, _, err := installer("4.5.2", "plan9"); err == nil {
		t.Fatal("unsupported OS accepted")
	}
	if !meta.Newer("4.10.0", "4.9.9") || meta.Newer("4.5.2", "4.5.2") || meta.Newer("4.5", "4.5.1") {
		t.Fatal("version order")
	}
}

// fakeSMAPI serves a release list and an installer zip whose installer script does what the real one does to the game folder.
func fakeSMAPI(t *testing.T, script string) Loader {
	t.Helper()
	installDat := testfs.ZipBytes(t, map[string]string{
		"Mods/ConsoleCommands/manifest.json": `{"Name":"Console Commands","UniqueID":"SMAPI.ConsoleCommands","Version":"9.9.9"}`,
		"Mods/SaveBackup/manifest.json":      `{"Name":"Save Backup","UniqueID":"SMAPI.SaveBackup","Version":"9.9.9"}`,
	})
	installer := testfs.ZipBytes(t, map[string]string{
		"SMAPI 9.9.9 installer/internal/linux/SMAPI.Installer": script,
		"SMAPI 9.9.9 installer/internal/linux/install.dat":     string(installDat),
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/releases", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"9.9.9","assets":[{"name":"SMAPI-9.9.9-installer.zip"}]}]`))
	})
	mux.HandleFunc("/dl/9.9.9/SMAPI-9.9.9-installer.zip", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(installer)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return Loader{ReleasesURL: srv.URL + "/releases", DownloadBase: srv.URL + "/dl", AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir(), LogDir: t.TempDir()}
}

const okScript = `#!/bin/sh
[ "$1" = --install ] && [ "$2" = --no-prompt ] && [ "$3" = --game-path ] || { echo "bad args: $*" >&2; exit 2; }
[ -f install.dat ] || { echo "not run from its own folder" >&2; exit 3; }
mv "$4/StardewValley" "$4/StardewValley-original"
echo 'exec StardewModdingAPI' > "$4/StardewValley"
touch "$4/StardewModdingAPI.dll"
`

func vanillaInstall(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, gameInfo().Marker), "")
	write(t, filepath.Join(dir, linuxLauncher), "native stub")
	return dir
}

// install fetches the installer and runs it the way the loader service does.
func install(g Loader, dir string, bundled loader.Bundled, progress func(loader.Step)) (string, error) {
	zip := filepath.Join(os.TempDir(), "smapi-test-installer.zip")
	defer func() { _ = os.Remove(zip) }()
	if err := g.Fetch(context.Background(), components.GameInfo{}, "9.9.9", zip); err != nil {
		return "", err
	}
	progress(loader.StepDownloaded)
	return g.Install(context.Background(), loader.Target{InstallDir: dir, Bundled: bundled}, loader.Package{ID: ID, Version: "9.9.9", Archive: zip}, progress)
}

func TestInstallRunsInstallerAndReportsSteps(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the fake installer is a shell script")
	}
	g := fakeSMAPI(t, okScript)
	dir := vanillaInstall(t)
	var steps []loader.Step
	var bundledMods []string
	version, err := install(g, dir, func(v, mods string) error {
		if v != "9.9.9" {
			t.Errorf("bundled version = %q", v)
		}
		ents, _ := os.ReadDir(mods)
		for _, e := range ents {
			bundledMods = append(bundledMods, e.Name())
		}
		return nil
	}, func(s loader.Step) { steps = append(steps, s) })
	if err != nil || version != "9.9.9" {
		t.Fatalf("install = %q, %v", version, err)
	}
	want := []loader.Step{loader.StepDownloaded, loader.StepFiles, loader.StepLauncher, loader.StepBundled}
	if strings.Join(stepStrings(steps), ",") != strings.Join(stepStrings(want), ",") {
		t.Fatalf("steps = %v", steps)
	}
	if len(bundledMods) != 2 {
		t.Fatalf("bundled mods = %v", bundledMods)
	}
	if st, err := g.Status(loader.Target{InstallDir: dir}); err != nil || !st.Installed || st.Broken {
		t.Fatalf("status = %+v, %v", st, err)
	}
}

func stepStrings(in []loader.Step) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = string(s)
	}
	return out
}

func TestInstallReportsInstallerFailure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the fake installer is a shell script")
	}
	g := fakeSMAPI(t, "#!/bin/sh\necho 'game path is not valid' >&2\nexit 1\n")
	_, err := install(g, vanillaInstall(t), func(string, string) error {
		t.Error("bundled mods added after a failed install")
		return nil
	}, func(loader.Step) {})
	if err == nil || !strings.Contains(err.Error(), "game path is not valid") {
		t.Fatalf("err = %v", err)
	}
}

func TestInstallWindowsBuildRunsInstallerThroughExec(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("a Windows build on a non-Windows host is Linux's case")
	}
	installDat := testfs.ZipBytes(t, map[string]string{
		"Mods/ConsoleCommands/manifest.json": `{"Name":"Console Commands","UniqueID":"SMAPI.ConsoleCommands","Version":"9.9.9"}`,
	})
	archive := filepath.Join(t.TempDir(), "installer.zip")
	write(t, archive, string(testfs.ZipBytes(t, map[string]string{
		"SMAPI 9.9.9 installer/internal/windows/SMAPI.Installer.exe": "MZ",
		"SMAPI 9.9.9 installer/internal/windows/install.dat":         string(installDat),
	})))
	dir := t.TempDir()
	write(t, filepath.Join(dir, "Stardew Valley.exe"), "")
	pkg := loader.Package{ID: ID, Version: "9.9.9", Archive: archive}
	progress := func(loader.Step) {}

	if _, err := (Loader{}).Install(context.Background(), loader.Target{InstallDir: dir}, pkg, progress); err == nil {
		t.Fatal("a Windows build with no runtime must not be installed into")
	}
	var argv []string
	exec := func(_ context.Context, _ launchplan.RuntimeReq, a []string) error {
		argv = a
		write(t, filepath.Join(dir, "StardewModdingAPI.exe"), "")
		return nil
	}
	target := loader.Target{InstallDir: dir, Exec: exec, Bundled: func(string, string) error { return nil }}
	if v, err := (Loader{}).Install(context.Background(), target, pkg, progress); err != nil || v != "9.9.9" {
		t.Fatalf("install = %q, %v", v, err)
	}
	if filepath.Base(argv[0]) != "SMAPI.Installer.exe" || strings.Join(argv[1:], " ") != "--install --no-prompt --game-path "+dir {
		t.Fatalf("argv = %q", argv)
	}
}

func TestInstallerErrorHasNoDanglingColon(t *testing.T) {
	err := errors.New("exit status 1")
	if got := installerError(err, []byte("boom\n")).Error(); got != "SMAPI installer failed (exit status 1): boom" {
		t.Fatalf("with output = %q", got)
	}
	if got := installerError(err, nil).Error(); strings.Contains(got, "):") || !strings.Contains(got, "retry") {
		t.Fatalf("without output = %q", got)
	}
}

// Status reads the launcher many times per start, so a read must cost the launcher's size, not the 16 MB bound.
func TestLauncherReadIsSizedToTheFile(t *testing.T) {
	dir := t.TempDir()
	testfs.WriteFile(t, dir, linuxLauncher, "#!/bin/sh\nexec ./"+smapiMarker+" \"$@\"\n")
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 10 {
		if !launcherHasSMAPI(dir) {
			t.Fatal("SMAPI launcher not recognised")
		}
	}
	runtime.ReadMemStats(&after)
	if n := after.TotalAlloc - before.TotalAlloc; n > 1<<20 {
		t.Fatalf("10 launcher reads allocated %d bytes", n)
	}
}
