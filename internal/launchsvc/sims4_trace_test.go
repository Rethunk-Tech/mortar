//go:build !windows

package launchsvc

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
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

// TestAModsSettingsFileAcrossLaunches records what the game sees and what survives when a script mod rewrites a file
// its archive shipped. It asserts the behaviour as it is: a placed file whose content changed is left at purge, and the
// next launch of the same profile treats that leftover as the player's own file.
func TestAModsSettingsFileAcrossLaunches(t *testing.T) {
	w := newSims4World(t)
	w.install(t, 1, map[string]string{"mc/mc.ts4script": "script", "mc/mc_settings.cfg": "default=1"})
	other := testenv.Profile(t, w.profiles, sims4, "B")
	cfg := filepath.Join(w.docs, "Mods", "mc", "mc_settings.cfg")
	state := filepath.Join(w.docs, "Mods", "mc", "mc_state.dat")

	w.launch(t, w.profileID, func(string) {
		if got := readFile(t, cfg); got != "default=1" {
			t.Fatalf("launch 1: the game sees %q", got)
		}
		writeFile(t, cfg, "player=2")
		writeFile(t, state, "created during play")
	})
	if got := readFile(t, cfg); got != "player=2" {
		t.Fatalf("after launch 1 the rewritten file is %q: it is left, not removed or reverted", got)
	}
	if fileExists(filepath.Join(w.docs, "Mods", "mc", "mc.ts4script")) {
		t.Fatal("after launch 1 the unchanged script was not taken back")
	}
	if got := readFile(t, state); got != "created during play" {
		t.Fatalf("a file the mod created is %q: no manifest names it, so it stays", got)
	}

	w.launch(t, w.profileID, func(string) {
		if got := readFile(t, cfg); got != "default=1" {
			t.Fatalf("launch 2: the game sees %q, the archive's bytes, not the settings it saved", got)
		}
		if got := readFile(t, state); got != "created during play" {
			t.Fatalf("launch 2: the created file is %q", got)
		}
		writeFile(t, cfg, "player=3")
	})
	if got := readFile(t, cfg); got != "player=2" {
		t.Fatalf("after launch 2 the file is %q: the displaced launch-1 bytes came back over the session's own changes", got)
	}

	w.launch(t, other.ID, func(string) {
		if got := readFile(t, cfg); got != "player=2" {
			t.Fatalf("a profile without the mod sees the leftover settings file as %q", got)
		}
		if fileExists(filepath.Join(w.docs, "Mods", "mc", "mc.ts4script")) {
			t.Fatal("a profile without the mod sees its script")
		}
	})
	if got := readFile(t, cfg); got != "player=2" {
		t.Fatalf("after the other profile's launch the file is %q", got)
	}
}
