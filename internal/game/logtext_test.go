package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// A game keeps its log and its crash report apart, and its mods' analyzers need both; a log that grew for days only
// needs its last run.
func TestGameLogTextReadsTheCrashLogAndKeepsTheTail(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	config := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("APPDATA", config)
	t.Setenv("FLATPAK_ID", "")
	set := settings.Defaults()
	logPath, err := PathFor(home, set, "vintage-story", "", PathLog)
	if err != nil {
		t.Fatal(err)
	}
	crashPath, err := PathFor(home, set, "vintage-story", "", PathCrashLog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		t.Fatal(err)
	}
	big := "FIRST-LINE\n" + strings.Repeat("x", launch.MaxLogBytes) + "\nLAST-LINE\n"
	if err := os.WriteFile(logPath, []byte(big), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(crashPath, []byte("CRASH-REPORT\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := GameLogText(home, set, "vintage-story", PathLog, PathCrashLog)
	if len(got) > 2*launch.MaxLogBytes || strings.Contains(got, "FIRST-LINE") || !strings.Contains(got, "LAST-LINE") ||
		!strings.Contains(got, "CRASH-REPORT") {
		t.Fatalf("log text of %d bytes: starts %.20q", len(got), got)
	}
	if small := GameLogText(home, set, "vintage-story", PathLog, "noSuchRole"); strings.Contains(small, "CRASH-REPORT") {
		t.Fatal("a role the game keeps no path for adds nothing")
	}
}
