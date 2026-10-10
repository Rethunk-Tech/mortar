//go:build !windows

package problems

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// sims4Rows is the Problems service over the real sims4 catalog entry (enabled in the test's copy only), a fake Steam
// library and Proton prefix, and the player's Options.ini holding body; it returns the setting rows for a new profile.
func sims4Rows(t *testing.T, mode, body string) []LoadFailure {
	t.Helper()
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = slices.Clone(m.Games)
	i := slices.IndexFunc(m.Games, func(g components.GameInfo) bool { return g.ID == "sims4" })
	m.Games[i].Enabled = true
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })
	datadirtest.Use(t, t.TempDir())
	home := t.TempDir()
	root := filepath.Join(home, ".local", "share", "Steam")
	put := func(p, s string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	put(filepath.Join(root, "steamapps", "libraryfolders.vdf"), "\"libraryfolders\"\n{\n\"0\"\n{\n\"path\" \""+root+"\"\n}\n}\n")
	put(filepath.Join(root, "steamapps", "appmanifest_1222670.acf"), "\"AppState\"\n{\n\"installdir\" \"The Sims 4\"\n}\n")
	put(filepath.Join(root, "steamapps", "common", "The Sims 4", "Game", "Bin", "TS4_x64.exe"), "exe")
	put(filepath.Join(root, "steamapps", "compatdata", "1222670", "pfx", "drive_c", "users", "steamuser", "Documents", "Electronic Arts", "The Sims 4", "Options.ini"), body)
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if mode != "" {
		if _, err := set.Update(func(v *settings.Settings) {
			gp := v.GamePrefs("sims4")
			gp.GameSettingsMode = mode
			v.Games = map[string]*settings.GameSettings{"sims4": &gp}
		}); err != nil {
			t.Fatal(err)
		}
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "sims4", "A")
	return NewService(home, set, profiles, nil).gameSettingFailures("sims4", p.ID)
}

func TestSims4WarnModeReportsTheSwitchesThatAreOff(t *testing.T) {
	body := "[options]\r\nmodsdisabled = 1\r\nscriptmodsenabled = 0\r\n"
	if rows := sims4Rows(t, settings.GameSettingsWarn, body); len(rows) != 2 {
		t.Fatalf("warn mode rows = %+v", rows)
	}
	if rows := sims4Rows(t, "", body); len(rows) != 0 {
		t.Fatalf("edit mode fixes it at launch, so no rows: %+v", rows)
	}
}

func TestSims4FileMortarCannotEditIsReportedWithTheReasonEvenInEditMode(t *testing.T) {
	rows := sims4Rows(t, "", "\xff\xfem\x00o\x00")
	if len(rows) != 2 || !strings.Contains(rows[0].Message, "UTF-16") {
		t.Fatalf("rows = %+v", rows)
	}
}
