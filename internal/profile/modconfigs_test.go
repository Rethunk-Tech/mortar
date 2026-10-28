package profile

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// A switched-off mod's config is set aside and comes back when it is switched on; removing the mod deletes it.
func TestModConfigFollowsTheMod(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.InstallSource(t.Context(), "lethal-company", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	en := res.Profile.Entries[0]
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	live := filepath.Join(dir, "BepInEx", "config", "Mod.cfg")
	aside := filepath.Join(dir, configAsideDir, "Mod.cfg")
	writeFile(t, dir, "BepInEx/config/Mod.cfg", "[General]\nA = 1\n")
	exists := func(path string) bool {
		_, err := fsx.Stat(path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
		return err == nil
	}
	toggle := func(on bool) {
		t.Helper()
		if _, _, err := e.enableMod("lethal-company", p.ID, en.Key, en.Mods[0].ID, on); err != nil {
			t.Fatal(err)
		}
		if err := e.SyncPackages("lethal-company", p.ID); err != nil {
			t.Fatal(err)
		}
	}
	toggle(false)
	if exists(live) || !exists(aside) {
		t.Fatalf("switched off: live %v, aside %v", exists(live), exists(aside))
	}
	toggle(true)
	if !exists(live) || exists(aside) {
		t.Fatalf("switched on: live %v, aside %v", exists(live), exists(aside))
	}
	if _, err := e.RemoveEntries("lethal-company", p.ID, []string{en.Key}); err != nil {
		t.Fatal(err)
	}
	if exists(live) || exists(aside) {
		t.Fatalf("removed: live %v, aside %v", exists(live), exists(aside))
	}
}

// A fork that shares a package name with a mod still switched on does not take that mod's config, and a removal that
// fails leaves every config where it was.
func TestModConfigStaysWithTheEnabledOwnerAndAFailedRemoveKeepsIt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	var keys []Entry
	for _, name := range []string{"Ns-Mod", "Fork-Mod"} {
		res, err := e.InstallSource(t.Context(), "lethal-company", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: name, Version: "1.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, res.Profile.Entries[len(res.Profile.Entries)-1])
	}
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	writeFile(t, dir, "BepInEx/config/Mod.cfg", "[General]\nA = 1\n")
	live := filepath.Join(dir, "BepInEx", "config", "Mod.cfg")
	if _, _, err := e.enableMod("lethal-company", p.ID, keys[0].Key, keys[0].Mods[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages("lethal-company", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := fsx.Stat(live); err != nil {
		t.Fatalf("the enabled fork's config was parked: %v", err)
	}
	if _, err := e.RemoveEntries("lethal-company", p.ID, []string{keys[0].Key, "no-such-key"}); err == nil {
		t.Fatal("a missing key removed")
	}
	if _, err := e.RemoveEntries("lethal-company", p.ID, []string{keys[0].Key}); err != nil {
		t.Fatal(err)
	}
	if _, err := fsx.Stat(live); err != nil {
		t.Fatalf("removing the disabled one took the enabled one's config: %v", err)
	}
}
