package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// A game the catalog alone describes (Unity, BepInEx 5, Thunderstore, Steam) is selectable, found on Steam and given
// its loader with no Go of its own. The entry lives only in this test's manifest, never the shipped catalog.
func TestACatalogOnlyGameNeedsNoCode(t *testing.T) {
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: "content-warning", Name: "Content Warning", Enabled: true, Marker: "Content Warning.exe",
		R2modmanFolder: "ContentWarning", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "profile", Root: "{profile}"}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "2881650"}},
		Loaders: []components.GameLoader{{ID: "bepinex5", Name: "BepInEx 5"}},
		Sources: []components.GameSource{{ID: "thunderstore", Key: "content-warning"}},
	})
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	ConfigureComponents(c)
	t.Cleanup(func() { ConfigureComponents(nil) })

	if !slices.Contains(Implemented(), "content-warning") {
		t.Fatalf("Implemented = %v", Implemented())
	}
	g := Find("content-warning")
	if g == nil || g.Name() != "Content Warning" || g.SteamAppID() != "2881650" ||
		!slices.Equal(g.ModSources(), []string{"thunderstore"}) ||
		!slices.Equal(g.GameProcesses(), []string{"Content Warning.exe", "Content Warning"}) {
		t.Fatalf("game = %#v", g)
	}
	if l, ok := PrimaryLoader("content-warning"); !ok || l.ID() != "bepinex5" {
		t.Fatalf("loader = %v, %v", l, ok)
	}
	if id, ok := ByR2modmanFolder("ContentWarning"); !ok || id != "content-warning" {
		t.Fatalf("r2modman folder = %q, %v", id, ok)
	}

	h := t.TempDir()
	root := filepath.Join(h, ".local", "share", "Steam")
	dir := filepath.Join(root, "steamapps", "common", "Content Warning")
	for path, body := range map[string]string{
		filepath.Join(dir, "Content Warning.exe"):                   "",
		filepath.Join(root, "steamapps", "libraryfolders.vdf"):      "\"libraryfolders\"\n{\n\"0\"\n{\n\"path\" \"" + root + "\"\n}\n}\n",
		filepath.Join(root, "steamapps", "appmanifest_2881650.acf"): "\"AppState\"\n{\n\"installdir\" \"Content Warning\"\n}\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.ValidInstall(dir); err != nil {
		t.Fatal(err)
	}
	list, err := NewService(h, testStore(t)).List()
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(list, func(r GameInfo) bool { return r.ID == "content-warning" })
	if i < 0 || !list[i].Installed || list[i].InstallDir != dir || list[i].LoaderID != "bepinex5" {
		t.Fatalf("list = %+v", list)
	}
}

func TestACatalogOnlyEAGameIsFoundInAnAddedFolderAndStartsWithoutSteam(t *testing.T) {
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: "ea-only", Name: "EA Only", Enabled: true, Marker: "EAOnly.exe", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "profile", Root: "{profile}"}},
		Stores:  components.GameStores{EA: &components.EAStore{Folder: "EA Only"}},
		Loaders: []components.GameLoader{{ID: "bepinex5", Name: "BepInEx 5"}},
		Sources: []components.GameSource{{ID: "thunderstore", Key: "ea-only"}},
	})
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	ConfigureComponents(c)
	t.Cleanup(func() { ConfigureComponents(nil) })

	lib := t.TempDir()
	dir := filepath.Join(lib, "EA Only")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "EAOnly.exe"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	s := settings.Settings{LauncherRoots: map[string][]string{LauncherEA: {lib}}}
	got, store, all, err := Resolve(t.TempDir(), s, "ea-only")
	if err != nil || got != dir || store != StoreEA || len(all) != 1 {
		t.Fatalf("Resolve = %q %q %v %v", got, store, all, err)
	}
	cmd, err := Starter{}.Command("windows", all[0], launchplan.New(launchplan.ModeProfile))
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name != filepath.Join(dir, "EAOnly.exe") || cmd.Relay {
		t.Fatalf("an EA game starts its own executable, not a Steam relay: %+v", cmd)
	}
}

func TestValidInstallFindsMarkerBelowRoot(t *testing.T) {
	dir := t.TempDir()
	g := catalogOnly("sims4")
	if g.ValidInstall(dir) == nil {
		t.Fatal("an empty folder validated")
	}
	bin := filepath.Join(dir, "Game", "Bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "TS4_x64.exe"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := g.ValidInstall(dir); err != nil {
		t.Fatal(err)
	}
}
