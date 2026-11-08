package launchsvc

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testcatalog"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestSandboxSpecBindsRuntimeSocketsAndCache(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	runtime := t.TempDir()
	auth := filepath.Join(t.TempDir(), "Xauthority")
	t.Setenv("XDG_RUNTIME_DIR", runtime)
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("XAUTHORITY", auth)
	extra := t.TempDir()
	svc, p, inst := nosaveSandboxEnv(t)
	spec, err := svc.sandboxSpec("nosave-game", p.ID, inst, &launchplan.Plan{}, extra)
	if err != nil {
		t.Fatal(err)
	}
	if spec.CacheDir == "" || !strings.HasSuffix(spec.CacheDir, "sandbox-cache") {
		t.Fatalf("CacheDir = %q", spec.CacheDir)
	}
	if _, err := os.Stat(spec.CacheDir); err != nil {
		t.Fatal(err)
	}
	wantSock := []string{
		auth,
		filepath.Join(runtime, "pipewire-0"),
		filepath.Join(runtime, "pulse"),
		filepath.Join(runtime, "wayland-0"),
	}
	if !slices.Equal(spec.Sockets, wantSock) {
		t.Fatalf("Sockets = %v, want %v", spec.Sockets, wantSock)
	}
	if !slices.Contains(spec.RW, extra) {
		t.Fatalf("RW missing extra folder: %v", spec.RW)
	}
	if spec.Warn == "" || !strings.Contains(spec.Warn, "save folder is not known") {
		t.Fatalf("Warn = %q", spec.Warn)
	}

	if _, err := svc.sandboxSpec("nosave-game", p.ID, inst, &launchplan.Plan{}, "relative"); usererr.KindOf(err) != usererr.Invalid {
		t.Fatalf("invalid extra = %v", err)
	}
}

func TestSandboxReportReadsWritableFoldersAndDenials(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "A")
	home := t.TempDir()
	svc := NewService(home, set, profiles)
	modsDir, err := profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir := runsDir(modsDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	writable := t.TempDir()
	if err := datadir.WriteJSON(filepath.Join(dir, "index.json"), runIndex{Runs: []Run{
		{ID: "r1", Sandbox: []string{writable}},
	}}); err != nil {
		t.Fatal(err)
	}
	denied := "/home/player/secret.cfg"
	if err := os.WriteFile(runLogPath(dir, "r1"), []byte("access to the path "+denied+" is denied\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := svc.SandboxReport("stardew", p.ID, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rep.Writable, []string{writable}) {
		t.Fatalf("Writable = %v", rep.Writable)
	}
	if len(rep.Denied) != 1 || rep.Denied[0].Path != denied {
		t.Fatalf("Denied = %+v", rep.Denied)
	}

	empty, err := svc.SandboxReport("stardew", p.ID, "missing")
	if err != nil || len(empty.Writable) != 0 || len(empty.Denied) != 0 {
		t.Fatalf("missing run: %+v %v", empty, err)
	}
}

func nosaveSandboxEnv(t *testing.T) (*Service, profile.Profile, game.Install) {
	t.Helper()
	testcatalog.AddGames(t, components.GameInfo{
		ID: "nosave-game", Name: "No Save", Enabled: true, Marker: "G.dll", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods", MaxDepth: map[string]int{"pkg": 1}}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "7"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
	})
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "G.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders["nosave-game"] = folder }); err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "nosave-game", "A")
	svc := NewService(t.TempDir(), set, profiles)
	inst := game.Install{ID: "folder", Dir: folder, Platform: "linux"}
	return svc, p, inst
}
