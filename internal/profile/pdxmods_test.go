package profile

import (
	"path/filepath"
	"testing"
)

const pdxGame = "cities-skylines-ii"

func paradoxSource(version string) Source {
	return Source{Kind: KindParadox, Name: "74604", ModName: "Anarchy", Version: version}
}

// A Paradox Mods archive goes to the game's own Mods folder, below a folder named for the mod, whatever BepInEx makes
// of an archive holding DLLs; its entries use Windows separators, as the site's do.
func TestParadoxArchivesLandInTheGamesModsFolderAndLeaveWithTheMod(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mods := filepath.Join(t.TempDir(), "Mods")
	e.TrayFolder = func(_, role string) (string, error) {
		if role != "pdxMods" {
			t.Errorf("role = %q", role)
		}
		return mods, nil
	}
	p, err := e.Create(pdxGame, "S")
	if err != nil {
		t.Fatal(err)
	}
	v1 := zipOf(t, "a.zip", map[string]string{"Anarchy.dll": "d1", `images\x.svg`: "s", "old.txt": "o"})
	res, err := e.InstallSource(t.Context(), pdxGame, p.ID, v1, paradoxSource("1.0"))
	if err != nil {
		t.Fatal(err)
	}
	got := trayTree(t, mods)
	if len(got) != 3 || got[filepath.Join("paradox-74604", "images", "x.svg")] != "s" || got[filepath.Join("paradox-74604", "Anarchy.dll")] != "d1" {
		t.Fatalf("Mods folder = %v", got)
	}
	if len(res.Profile.Entries) != 1 || res.Profile.Entries[0].Mods[0].Name != "Anarchy" {
		t.Fatalf("entries = %+v", res.Profile.Entries)
	}
	owners, _ := e.PackageFileOwners(pdxGame, p.ID)
	if len(owners) != 0 {
		t.Fatalf("the profile must lay out nothing for it: %v", owners)
	}

	v2 := zipOf(t, "b.zip", map[string]string{"Anarchy.dll": "d2", `images\x.svg`: "s"})
	res, err = e.InstallSource(t.Context(), pdxGame, p.ID, v2, paradoxSource("1.1"))
	if err != nil {
		t.Fatal(err)
	}
	got = trayTree(t, mods)
	if len(got) != 2 || got[filepath.Join("paradox-74604", "Anarchy.dll")] != "d2" || len(res.Profile.Entries) != 1 {
		t.Fatalf("after the update: Mods folder = %v, entries = %v", got, keysOf(res.Profile))
	}

	if _, err := e.RemoveEntries(pdxGame, p.ID, []string{res.Profile.Entries[0].Key}); err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, mods); len(got) != 0 {
		t.Fatalf("after removal: %v", got)
	}
}
