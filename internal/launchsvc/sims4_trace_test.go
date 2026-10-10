//go:build !windows

package launchsvc

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// launch runs the pipeline stages Start runs, up to but not including the process, for one profile; play is the game
// session, and the launch is taken back afterwards.
func (w *sims4World) launch(t *testing.T, profileID string, play func(mods string)) {
	t.Helper()
	g := game.Find(sims4)
	_, inst := w.svc.installs(g)
	plan, err := w.svc.planProfile(t.Context(), g, inst, profileID, launchplan.ModeProfile, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	dep, err := w.svc.deployProfile(t.Context(), sims4, inst, profileID, plan)
	if err != nil {
		t.Fatal(err)
	}
	play(filepath.Join(w.docs, "Mods"))
	dep.unwind(t.Context())
}

// TestAModsSettingsFileAcrossLaunches is the acceptance test of per-profile write-back: a script mod rewrites a file its
// archive shipped and creates another in its own folder. The profile keeps both, the shared folder keeps neither, the
// next launch of the profile sees what was saved, and another profile sees none of it. A file at the Mods root stays the
// player's own. An update keeps the saved copy, an uninstall holds it under changed/ and a backup round trips it.
func TestAModsSettingsFileAcrossLaunches(t *testing.T) {
	w := newSims4World(t)
	w.install(t, 1, map[string]string{"mc/mc.ts4script": "script", "mc/mc_settings.cfg": "default=1"})
	other := testenv.Profile(t, w.profiles, sims4, "B")
	cfg := filepath.Join(w.docs, "Mods", "mc", "mc_settings.cfg")
	state := filepath.Join(w.docs, "Mods", "mc", "mc_state.dat")
	rootFile := filepath.Join(w.docs, "Mods", "root_created.dat")
	pdir, err := w.profiles.ProfileDir(sims4, w.profileID)
	if err != nil {
		t.Fatal(err)
	}
	ownCfg := filepath.Join(pdir, "Mods", "mc", "mc_settings.cfg")

	w.launch(t, w.profileID, func(string) {
		if got := readFile(t, cfg); got != "default=1" {
			t.Fatalf("launch 1: the game sees %q", got)
		}
		writeFile(t, cfg, "player=2")
		writeFile(t, state, "created during play")
		writeFile(t, rootFile, "root file")
	})
	if fileExists(cfg) || fileExists(state) || fileExists(filepath.Join(w.docs, "Mods", "mc", "mc.ts4script")) {
		t.Fatal("a file of the mod is left in the shared folder after launch 1")
	}
	if got := readFile(t, ownCfg); got != "player=2" {
		t.Fatalf("the profile's copy after launch 1 = %q", got)
	}
	if got := readFile(t, filepath.Join(pdir, "Mods", "mc", "mc_state.dat")); got != "created during play" {
		t.Fatalf("the created file was not adopted: %q", got)
	}
	if got := readFile(t, rootFile); got != "root file" {
		t.Fatalf("a file at the Mods root = %q: it stays the player's own", got)
	}

	w.launch(t, w.profileID, func(string) {
		if got := readFile(t, cfg); got != "player=2" {
			t.Fatalf("launch 2: the game sees %q, not the settings it saved", got)
		}
		if got := readFile(t, state); got != "created during play" {
			t.Fatalf("launch 2: the adopted file is %q", got)
		}
		writeFile(t, cfg, "player=3")
	})
	if got := readFile(t, ownCfg); got != "player=3" {
		t.Fatalf("after launch 2 the profile's copy = %q", got)
	}

	w.launch(t, other.ID, func(string) {
		if fileExists(cfg) || fileExists(state) || fileExists(filepath.Join(w.docs, "Mods", "mc", "mc.ts4script")) {
			t.Fatal("a profile without the mod sees its files")
		}
	})

	// An update of the archive keeps the saved copy of a file it still ships.
	w.install(t, 2, map[string]string{"mc/mc.ts4script": "script v2", "mc/mc_settings.cfg": "default=9"})
	if err := w.profiles.SyncPackages(sims4, w.profileID); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, ownCfg); got != "player=3" {
		t.Fatalf("after an update the profile's copy = %q", got)
	}
	if got := readFile(t, filepath.Join(pdir, "Mods", "mc", "mc.ts4script")); got != "script v2" {
		t.Fatalf("an unchanged file did not take the update: %q", got)
	}

	// A backup carries the changed copies and a restore puts them back.
	prof, files, err := w.profiles.Backup(sims4, w.profileID)
	if err != nil {
		t.Fatal(err)
	}
	dirs, err := w.profiles.BackupDirs(sims4, prof, func(profile.Entry) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := w.profiles.RestoreBackup(t.Context(), sims4, prof, files, dirs)
	if err != nil {
		t.Fatal(err)
	}
	rdir, _ := w.profiles.ProfileDir(sims4, restored.ID)
	if got := readFile(t, filepath.Join(rdir, "Mods", "mc", "mc_settings.cfg")); got != "player=3" {
		t.Fatalf("a restored backup holds %q", got)
	}
	for i := range 2 {
		if err := w.profiles.SyncPackages(sims4, restored.ID); err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, filepath.Join(rdir, "Mods", "mc", "mc_settings.cfg")); got != "player=3" {
			t.Fatalf("sync %d after the restore overwrote the copy with %q", i+1, got)
		}
	}

	// An uninstall holds the changed copy under changed/ instead of deleting it.
	cur, err := w.profiles.Installed(sims4, w.profileID)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, m := range cur {
		keys = append(keys, m.Key)
	}
	if _, err := w.profiles.RemoveEntries(sims4, w.profileID, keys); err != nil {
		t.Fatal(err)
	}
	if err := w.profiles.SyncPackages(sims4, w.profileID); err != nil {
		t.Fatal(err)
	}
	if fileExists(ownCfg) {
		t.Fatal("the removed mod's settings file is still laid out")
	}
	if got := readFile(t, filepath.Join(pdir, "changed", "Mods", "mc", "mc_settings.cfg")); got != "player=3" {
		t.Fatalf("the held copy = %q", got)
	}
}
