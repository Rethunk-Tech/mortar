package problems

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// Vintage Story keeps its log and its crash report apart and writes a mod's failure to the main log, a crash into
// code to the crash report: the check reads both, as the crash hint does.
func TestGameLogReadsTheLogAndTheCrashReport(t *testing.T) {
	home := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("APPDATA", cfg)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("FLATPAK_ID", "")
	set, profiles := testenv.Env(t)
	const vs = "vintage-story"
	p := testenv.Profile(t, profiles, vs, "A")
	s := NewService(home, set, profiles, nil)
	write := func(role, body string) {
		t.Helper()
		path, err := game.PathFor(home, set.Get(), vs, "", role)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(game.PathLog, "MAINLOG\n")
	write(game.PathCrashLog, "CRASHLOG\n")
	got := s.gameLog(vs, p.ID)
	if !strings.Contains(got, "MAINLOG") || !strings.Contains(got, "CRASHLOG") {
		t.Fatalf("game log = %q", got)
	}
}
