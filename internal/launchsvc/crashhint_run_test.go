package launchsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// A run's hint is about that run: the game logs belong to whichever profile ran last, so the hint reads the copies the
// run kept, and reads the live game log only while this profile is still the one that ran last.
func TestCrashHintReadsTheRunsOwnLogsNotTheLiveOnes(t *testing.T) {
	cfg := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("APPDATA", cfg)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("FLATPAK_ID", "")
	set, profiles := testenv.Env(t)
	const vs = "vintage-story"
	a := testenv.Profile(t, profiles, vs, "A")
	b := testenv.Profile(t, profiles, vs, "B")
	svc := NewService(home, set, profiles)

	modsA, err := profiles.ModsDir(vs, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir := runsDir(modsA)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeIndex := func(profileID, id string) {
		t.Helper()
		mods, err := profiles.ModsDir(vs, profileID)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(runsDir(mods), 0o700); err != nil {
			t.Fatal(err)
		}
		index := mustJSON(t, runIndex{Runs: []Run{{ID: id, Outcome: launch.OutcomeCrashed}}})
		if err := os.WriteFile(filepath.Join(runsDir(mods), "index.json"), index, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeIndex(a.ID, "r1")
	// Vintage Story's mod loader writes a mod's failure to the game's log, which the run kept with its crash report.
	stored := "30.11.2025 18:29:40 [Error] [FishBasket] An exception was thrown when trying to start the mod:\n" +
		"System.NullReferenceException: caf\xff boom\n"
	if err := os.WriteFile(runLogPath(dir, "r1"), []byte(stored), 0o600); err != nil {
		t.Fatal(err)
	}
	setText := func(role, text string) string {
		t.Helper()
		path, err := game.PathFor(home, set.Get(), vs, "", role)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	setText(game.PathLog, "30.11.2025 19:00:00 [Error] [Other] An exception was thrown when trying to start the mod:\nSystem.Exception: live\n")
	setText(game.PathCrashLog, "CRASH-REPORT\n")

	// Both roles reach the analyzers, and only while this profile is the one that ran last.
	if got := svc.gameLogText(vs, a.ID, game.PathLog, game.PathCrashLog); !strings.Contains(got, "live") || !strings.Contains(got, "CRASH-REPORT") {
		t.Fatalf("game log text = %q", got)
	}
	hint, err := svc.CrashHint(vs, a.ID)
	if err != nil || hint == nil {
		t.Fatalf("hint = %+v, %v", hint, err)
	}
	if !strings.Contains(hint.Reason, "NullReferenceException") || strings.Contains(hint.Reason, "live") {
		t.Fatalf("the hint must read the run's copy, not the live log: %q", hint.Reason)
	}
	if !utf8.ValidString(hint.Reason) {
		t.Fatalf("the run's log reached the hint with a broken rune: %q", hint.Reason)
	}

	// A later run of another profile owns the game's logs now, and this profile's hint does not change.
	writeIndex(b.ID, "r2")
	if got := svc.gameLogText(vs, a.ID, game.PathLog, game.PathCrashLog); got != "" {
		t.Fatalf("another profile ran after this one, so its log is not this profile's: %q", got)
	}
	again, err := svc.CrashHint(vs, a.ID)
	if err != nil || again == nil || again.Reason != hint.Reason || again.RunID != "r1" {
		t.Fatalf("hint changed after another profile ran: %+v, %v", again, err)
	}
}
