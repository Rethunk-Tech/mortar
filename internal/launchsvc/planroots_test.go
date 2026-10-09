package launchsvc

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestPlanRootsResolvesEachRoleOnceAndRefusesAnUnknownOne(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: "roots-game", Name: "Roots Game", Enabled: true, Marker: "R.x86_64", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods"}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "4242"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
		Paths:   map[string]components.PathTemplate{"mods": {Windows: "{home}/ModsOut", Linux: "{home}/ModsOut", Darwin: "{home}/ModsOut"}},
	})
	c := components.NewClient(nil)
	c.SetManifest(m)
	game.ConfigureComponents(c)
	t.Cleanup(func() { game.ConfigureComponents(nil) })

	home := t.TempDir()
	steamRoot := filepath.Join(home, ".local", "share", "Steam")
	dir := filepath.Join(steamRoot, "steamapps", "common", "Roots Game")
	for path, body := range map[string]string{
		filepath.Join(dir, "R.x86_64"):                                   "",
		filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"):   "\"libraryfolders\"\n{\n\"0\"\n{\n\"path\" \"" + steamRoot + "\"\n}\n}\n",
		filepath.Join(steamRoot, "steamapps", "appmanifest_4242.acf"): "\"AppState\"\n{\n\"installdir\" \"Roots Game\"\n}\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, _, all, err := game.Resolve(home, set.Get(), "roots-game")
	if err != nil || len(all) != 1 {
		t.Fatalf("install = %v, %v", all, err)
	}
	svc := NewService(home, set, nil)

	plan := launchplan.New(launchplan.ModeProfile)
	plan.AddFile(launchplan.PlanFile{Src: "a", Dst: "a", Root: "mods"})
	plan.AddFile(launchplan.PlanFile{Src: "b", Dst: "b", Root: "mods"})
	plan.AddFile(launchplan.PlanFile{Src: "c", Dst: "c"})
	roots, err := svc.planRoots("roots-game", all[0], plan)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "ModsOut"); len(roots) != 1 || roots["mods"] != want {
		t.Fatalf("roots = %v, want mods -> %s", roots, want)
	}
	if roots, err := svc.planRoots("roots-game", all[0], launchplan.New(launchplan.ModeProfile)); err != nil || roots != nil {
		t.Fatalf("a plan with no root needs none: %v, %v", roots, err)
	}

	plan.AddFile(launchplan.PlanFile{Src: "d", Dst: "d", Root: "unknownRole"})
	if _, err := svc.planRoots("roots-game", all[0], plan); err == nil {
		t.Fatal("a role the game has no path for must be refused")
	}
}
