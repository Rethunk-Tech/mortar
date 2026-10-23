package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// A Mods folder holding a folder with no SMAPI manifest, in an install of each game.
func modsFolderService(t *testing.T, gameID string) (*Service, string) {
	t.Helper()
	testfs.DataHome(t)
	e := newEnv(t)
	install := t.TempDir()
	info, _ := components.Game(gameID)
	writeFile(t, install, info.Marker, "game")
	writeFile(t, install, "Mods/NotSmapi/mod.dll", "x")
	st, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Update(func(s *settings.Settings) { s.GameFolders = map[string]string{gameID: install} }); err != nil {
		t.Fatal(err)
	}
	return NewService(e.Store, t.TempDir(), st), install
}

// Import from the game's Mods folder, its new-folders diff and the manager imports read SMAPI manifests, so they
// apply only to a game whose loader reads manifest mods from a folder of the install (the manager imports still
// work for any game, with no Mods folder to fall back on).
func TestModsFolderImportAppliesOnlyToGamesWhoseLoaderReadsManifestMods(t *testing.T) {
	for _, tc := range []struct {
		game string
		want bool
	}{{"stardew", true}, {"lethal-company", false}, {"schedule-i", false}} {
		t.Run(tc.game, func(t *testing.T) {
			svc, _ := modsFolderService(t, tc.game)
			if got := svc.ImportsModsFolder(tc.game); got != tc.want {
				t.Fatalf("ImportsModsFolder = %v, want %v", got, tc.want)
			}
			preview, err := svc.PreviewGameMods(tc.game)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(preview.Mods) > 0; got != tc.want {
				t.Errorf("preview rows = %+v, want rows: %v", preview.Mods, tc.want)
			}
			fresh, err := svc.NewGameModsFolders(tc.game)
			if err != nil || fresh.Mods == nil {
				t.Fatalf("new folders = %+v, %v", fresh, err)
			}
			diff, err := svc.GameModsDiff(tc.game, "none")
			if tc.want {
				if err == nil {
					t.Error("stardew diff of a missing profile must read the profile")
				}
			} else if err != nil || len(diff.Unreadable) != 0 || diff.Missing == nil {
				t.Errorf("diff = %+v, %v", diff, err)
			}
			if !tc.want {
				if _, err := svc.ImportGameMods(t.Context(), tc.game, false); err == nil {
					t.Error("importing a Mods folder this game does not read must be refused")
				}
			}
			if _, err := svc.ExternalSources(tc.game); err != nil {
				t.Errorf("sources: %v", err)
			}
		})
	}
}
