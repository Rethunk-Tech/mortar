package melonloader

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func TestRoute(t *testing.T) {
	// r2modman's rules for BONELAB, cut down: a default route, a folder with sub-routes and extensions.
	classic := []components.LoaderRoute{
		{Dir: "Mods", Extensions: []string{".dll"}, Default: true},
		{Dir: "Plugins", Extensions: []string{".plugin.dll"}},
		{Dir: "UserLibs", Extensions: []string{".lib.dll"}},
		{Dir: "MelonLoader"},
		{Dir: "MelonLoader/Managed", Extensions: []string{".managed.dll"}},
		{Dir: "UserData"},
		{Dir: "UserData/CustomItems", Extensions: []string{".melon"}},
		{Dir: "UserData/ModManager", Track: components.TrackKeep},
	}
	cases := []struct {
		routes []components.LoaderRoute
		in     string
		want   string
	}{
		{nil, "Mod.dll", "Mods/Ns-Mod/Mod.dll"},
		{nil, "Sub/Mod.dll", "Mods/Ns-Mod/Sub/Mod.dll"},
		{nil, "UserData/cfg/a.cfg", "UserData/Ns-Mod/cfg/a.cfg"},
		{nil, "userdata/a.cfg", "UserData/Ns-Mod/a.cfg"},
		{nil, "manifest.json", ""},
		{nil, "Readme.MD", ""},
		{classic, "Mod.dll", "Mods/Mod.dll"},
		{classic, "x/Mod.plugin.dll", "Plugins/Mod.plugin.dll"},
		{classic, "Lib.lib.dll", "UserLibs/Lib.lib.dll"},
		{classic, "a/Fix.managed.dll", "MelonLoader/Managed/Fix.managed.dll"},
		{classic, "Plugins/Deep/P.dll", "Plugins/Deep/P.dll"},
		{classic, "Mods/Sub/Mod.dll", "Mods/Sub/Mod.dll"},
		{classic, "Pack/Sword.melon", "UserData/CustomItems/Sword.melon"},
		{classic, "CustomItems/Axe.melon", "UserData/CustomItems/Axe.melon"},
		{classic, "ModManager/m/a.json", "UserData/ModManager/Ns-Mod/ModManager/m/a.json"},
		{classic, "Data.bin", "Mods/Data.bin"},
		{classic, "LICENCE", ""},
		{classic, "../evil.dll", ""},
		{classic, `Mods\..\..\evil.dll`, ""},
		{classic, "/etc/evil.dll", ""},
		{classic, "C:/evil.dll", ""},
		{classic, "Mods/Mod.dll:stream", ""},
	}
	for _, c := range cases {
		if got := Route(c.in, "Ns-Mod", c.routes); got != c.want {
			t.Errorf("Route(%q, %d routes) = %q, want %q", c.in, len(c.routes), got, c.want)
		}
	}
	for _, bad := range []string{"", "..", "../x", `a\b`, "C:x"} {
		if got := Route("Mod.dll", bad, nil); got != "" {
			t.Errorf("Route with package %q = %q, want none", bad, got)
		}
	}
}

func put(t *testing.T, p, body string) {
	t.Helper()
	testfs.WriteAt(t, p, []byte(body))
}

func buildRelease(t *testing.T, root string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"version.dll":                  "proxy",
		"MelonLoader/net6/Loader.dll":  "loader",
		"MelonLoader/Dependencies/a.x": "dep",
	} {
		w, _ := zw.Create(root + name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "MelonLoader.x64.zip")
	if err := fsx.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInstallAndLaunch(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	tgt := loader.Target{ProfileDir: profile}
	if st, _ := (Loader{}).Status(tgt); st.Installed {
		t.Fatal("an empty profile reads as installed")
	}
	// A reinstall replaces the loader folder, whether the zip holds it at its top or one folder down, and keeps the
	// profile's mods.
	for _, root := range []string{"", "MelonLoader.x64/"} {
		put(t, filepath.Join(profile, "Mods", "Mod.dll"), "mod")
		put(t, filepath.Join(profile, loaderDir, "stale.dll"), "old")
		got, err := (Loader{}).Install(t.Context(), tgt, loader.Package{Version: "0.7.3", Archive: buildRelease(t, root)}, nil)
		if err != nil || got != "0.7.3" {
			t.Fatalf("install %q: %q, %v", root, got, err)
		}
		if st, _ := (Loader{}).Status(tgt); !st.Installed || st.Version != "0.7.3" {
			t.Fatalf("status %+v", st)
		}
		if fsx.IsFile(filepath.Join(profile, loaderDir, "stale.dll")) || !fsx.IsFile(filepath.Join(profile, "Mods", "Mod.dll")) {
			t.Fatal("the loader folder was not replaced, or the profile's mods were")
		}
	}

	plan := launchplan.New(launchplan.ModeProfile)
	view := loader.ProfileView{Dir: profile, Runtime: "proton", Platform: "windows"}
	if err := (Loader{}).Contribute(t.Context(), plan, view); err != nil {
		t.Fatal(err)
	}
	z := `Z:` + strings.ReplaceAll(filepath.ToSlash(profile), "/", `\`)
	if got, want := strings.Join(plan.Args, " "), "--melonloader.basedir "+z+" --melonloader.agfregenerate --melonloader.hideconsole"; got != want {
		t.Fatalf("args %q, want %q", got, want)
	}
	if len(plan.Files) != 1 || plan.Files[0].Dst != "version.dll" || plan.Files[0].Src != filepath.Join(profile, "version.dll") {
		t.Fatalf("files %+v", plan.Files)
	}
	if len(plan.RuntimeReqs) != 1 || plan.RuntimeReqs[0].Key != "version" {
		t.Fatalf("runtime %+v", plan.RuntimeReqs)
	}
	if !(Loader{}).Owns(loader.Process{Args: plan.Args}, view) || (Loader{}).Owns(loader.Process{Args: []string{"--melonloader.basedir", "/elsewhere"}}, view) {
		t.Fatal("ownership")
	}
	put(t, filepath.Join(profile, loaderDir, generatedIl2Cpp, "Assembly-CSharp.dll"), "x")
	if got := LaunchArgs(profile, false); len(got) != 2 || got[1] != profile {
		t.Fatalf("a profile with generated assemblies regenerates them: %v", got)
	}
	view.Platform = "linux"
	if err := (Loader{}).Contribute(t.Context(), launchplan.New(launchplan.ModeProfile), view); err == nil {
		t.Fatal("a native Linux build starts")
	}
	if !(Loader{}).Ready("[12:00:00.000] [MelonLoader] 3 Mods loaded.") || (Loader{}).Ready("0 Plugins loaded.") {
		t.Fatal("ready line")
	}
}

// A release other than the catalog's is trusted only through a checksum: GitHub's digest or the release's .sha512 file.
// The fake releases carry versions no catalog pins.
func TestFetchOtherReleaseVerifies(t *testing.T) {
	body := []byte("release zip")
	s256 := sha256.Sum256(body)
	s512 := sha512.Sum512(body)
	var srv *httptest.Server
	release := func(tag string, assets ...github.Asset) github.Release {
		return github.Release{Tag: tag, Assets: assets}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/LavaGang/MelonLoader/releases", func(w http.ResponseWriter, _ *http.Request) {
		zipAsset := func(digest string) github.Asset {
			return github.Asset{Name: asset, URL: srv.URL + "/dl/zip", Digest: digest}
		}
		sumAsset := github.Asset{Name: "MelonLoader.x64.sha512", URL: srv.URL + "/dl/sum"}
		_ = json.NewEncoder(w).Encode([]github.Release{
			release("v0.0.9", zipAsset("sha256:"+hex.EncodeToString(s256[:]))),
			release("v0.0.8", zipAsset(""), sumAsset),
			release("v0.0.7", zipAsset("sha256:"+strings.Repeat("0", 64))),
			release("v0.0.6", zipAsset("")),
			{Tag: "v0.0.5", Prerelease: true, Assets: []github.Asset{zipAsset("")}},
		})
	})
	mux.HandleFunc("/dl/zip", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) })
	mux.HandleFunc("/dl/sum", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "%s  MelonLoader.x64.zip\n", hex.EncodeToString(s512[:]))
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	l := Loader{Components: components.NewClient(nil), APIBase: srv.URL, HTTP: srv.Client()}

	versions, err := l.Versions(t.Context(), components.GameInfo{})
	if err != nil || strings.Join(versions, " ") != "0.0.9 0.0.8 0.0.7 0.0.6" {
		t.Fatalf("versions %v, %v", versions, err)
	}
	for version, ok := range map[string]bool{"0.0.9": true, "0.0.8": true, "0.0.7": false, "0.0.6": false, "0.0.5": false, "9.9": false} {
		dst := filepath.Join(t.TempDir(), "MelonLoader.x64.zip")
		err := l.Fetch(t.Context(), components.GameInfo{}, version, dst)
		if (err == nil) != ok {
			t.Errorf("Fetch(%s): %v, want ok=%v", version, err, ok)
		}
		if err != nil && fsx.IsFile(dst) {
			t.Errorf("Fetch(%s) left a file it refused", version)
		}
	}
}

func TestLaunchSettingsChooseConsoleAndLogArguments(t *testing.T) {
	dir := t.TempDir()
	l := Loader{}
	if got := strings.Join(settingArgs(dir), " "); got != "--melonloader.hideconsole" {
		t.Fatalf("default args %q", got)
	}
	for id, v := range map[string]string{settingConsole: "true", settingLogLevel: "all"} {
		if err := l.SetLaunchSetting(dir, id, v); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Join(settingArgs(dir), " "); got != "--melonloader.debug --melonloader.harmonyloglevel 4 --melonloader.captureplayerlogs" {
		t.Fatalf("args %q", got)
	}
	got, err := l.LaunchSettings(dir)
	if err != nil || got[0].Value != "true" || got[1].Value != "all" {
		t.Fatalf("settings %+v, err %v", got, err)
	}
	if l.SetLaunchSetting(dir, settingLogLevel, "loud") == nil || l.SetLaunchSetting(dir, "nope", "x") == nil {
		t.Fatal("an unknown level or setting was accepted")
	}
}
