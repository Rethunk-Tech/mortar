package stardew

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/meta"

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
	g := Game{ReleasesURL: srv.URL, AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir()}
	for range 2 {
		v, err := g.LatestLoader(context.Background())
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
	g := Game{ReleasesURL: srv.URL, AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir()}
	if v, err := g.LatestLoader(context.Background()); err != nil || v != "4.8.0" {
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
	if v, err := g.LatestLoader(context.Background()); err != nil || v != "4.8.0" {
		t.Fatalf("stale fallback = %q, %v", v, err)
	}
	if _, err := (Game{ReleasesURL: srv.URL, AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir()}).LatestLoader(context.Background()); err == nil {
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
	_, err := Game{ReleasesURL: srv.URL, AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir()}.LatestLoader(context.Background())
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
	g := Game{LogDir: logs}
	if s, gv := g.logVersions(); s != "4.5.2" || gv != "1.6.15" {
		t.Fatalf("log = %q %q", s, gv)
	}
	if s, gv := (Game{LogDir: t.TempDir()}).logVersions(); s != "" || gv != "" {
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
func fakeSMAPI(t *testing.T, script string) Game {
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
	return Game{ReleasesURL: srv.URL + "/releases", DownloadBase: srv.URL + "/dl", AssetPattern: "SMAPI-{version}-installer.zip", CacheDir: t.TempDir(), LogDir: t.TempDir()}
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
	write(t, filepath.Join(dir, identity().Marker), "")
	write(t, filepath.Join(dir, linuxLauncher), "native stub")
	return dir
}

func TestInstallLoaderRunsInstallerAndReportsSteps(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the fake installer is a shell script")
	}
	g := fakeSMAPI(t, okScript)
	dir := vanillaInstall(t)
	var steps []loader.Step
	var bundledMods []string
	version, err := g.InstallLoader(context.Background(), dir, func(v, mods string) error {
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
	if st := g.LoaderStatus(dir, "9.9.9"); !st.Installed || st.Broken || st.Version != "9.9.9" {
		t.Fatalf("status = %+v", st)
	}
}

func stepStrings(in []loader.Step) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = string(s)
	}
	return out
}

func TestInstallLoaderReportsInstallerFailure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the fake installer is a shell script")
	}
	g := fakeSMAPI(t, "#!/bin/sh\necho 'game path is not valid' >&2\nexit 1\n")
	_, err := g.InstallLoader(context.Background(), vanillaInstall(t), func(string, string) error {
		t.Error("bundled mods added after a failed install")
		return nil
	}, func(loader.Step) {})
	if err == nil || !strings.Contains(err.Error(), "game path is not valid") {
		t.Fatalf("err = %v", err)
	}
}

func TestInstallLoaderRejectsNonGameFolder(t *testing.T) {
	g := fakeSMAPI(t, okScript)
	if _, err := g.InstallLoader(context.Background(), t.TempDir(), nil, func(loader.Step) {}); err == nil {
		t.Fatal("installed into a folder that is not Stardew Valley")
	}
}
