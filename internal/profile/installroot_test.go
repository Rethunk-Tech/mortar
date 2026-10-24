package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// A folder game whose mods are laid out as packed from the install itself has no mods folder to import from: offering
// the install's own folders as mods, with a move, would take the game apart.
func TestAFolderGameWhoseModsFolderIsTheInstallOffersNothingToImport(t *testing.T) {
	testfs.DataHome(t)
	e := newEnv(t)
	install := t.TempDir()
	for _, rel := range []string{"bin/x64_DX12/witcher3.exe", "content/content0/scripts/game/r4Game.ws"} {
		writeFile(t, install, rel, "game")
	}
	st, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Update(func(s *settings.Settings) { s.GameFolders = map[string]string{"witcher3": install} }); err != nil {
		t.Fatal(err)
	}
	svc := NewService(e.Store, t.TempDir(), st)
	preview, err := svc.PreviewGameMods(t.Context(), "witcher3")
	if err != nil || len(preview.Mods) != 0 {
		t.Fatalf("preview = %+v, %v; want nothing", preview, err)
	}
	if _, err := svc.ImportGameMods(t.Context(), "witcher3", true); err == nil {
		t.Fatal("importing the install folder as mods must be refused")
	}
	if _, err := os.Stat(filepath.Join(install, "bin")); err != nil {
		t.Fatalf("the install must be untouched: %v", err)
	}
}
