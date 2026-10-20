package launchsvc

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// A BepInEx game's mods target (its mod.io mods) is placed into the game's mods folder beside the loader's files.
func TestABepInExLaunchAlsoPlacesTheGamesModsFolder(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	_, ps := testenv.Stores(t)
	const gm = "gladio-mori"
	p := testenv.Profile(t, ps, gm, "Duel")
	dir, err := ps.ProfileDir(gm, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, dir, "GameMods/k/GunioMori.dll", "d")
	install := t.TempDir()
	testfs.WriteFile(t, install, "Gladio Mori.exe", "exe")
	g, err := game.Require(gm)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := (&Service{profiles: ps}).launchPlan(t.Context(), g, game.Install{ID: "g1", Dir: install}, p.ID, launchplan.ModeProfile, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := launchplan.PlanFile{Src: filepath.Join(dir, "GameMods", "k", "GunioMori.dll"), Dst: filepath.Join("k", "GunioMori.dll"), Root: "mods"}
	if !slices.Contains(plan.Files, want) {
		t.Fatalf("plan files = %+v", plan.Files)
	}
}
