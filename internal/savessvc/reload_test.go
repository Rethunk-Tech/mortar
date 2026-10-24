package savessvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/customgame"
)

// A catalog fetched after start that changes a game's save files is what the scan reads next, without a restart.
func TestReloadTakesTheSaveFilesOfANewerCatalog(t *testing.T) {
	s, _ := backupService(t)
	t.Cleanup(func() { components.SetUserGames(nil) })
	install := t.TempDir()
	exe := filepath.Join(install, "Game.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	e, _, err := customgame.Add("Mail Game", exe, customgame.Folder, "")
	if err != nil {
		t.Fatal(err)
	}
	catalog := func(files ...string) {
		info, _ := components.Game(e.ID)
		info.Paths = map[string]components.PathTemplate{"saves": {Windows: "{home}/Saves", Linux: "{home}/Saves"}}
		info.SaveFiles = files
		components.SetUserGames([]components.GameInfo{info})
	}
	catalog()
	s.scanners = s.buildScanners(t.TempDir())
	before := s.scanner(e.ID)
	if before == nil {
		t.Fatal("no scanner for the game")
	}
	// Saved where the game writes it: the Windows build's folder inside its prefix.
	file := filepath.Join(before.Dir, "GameSaves", "GameSave_1.bin")
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("save"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := before.Scan(t.Context(), nil); len(got) != 0 {
		t.Fatalf("the old layout listed %+v", got)
	}
	catalog("GameSaves/GameSave_*.bin")
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	sc := s.scanner(e.ID)
	if sc == nil {
		t.Fatal("no scanner for the game")
	}
	got, err := sc.Scan(t.Context(), nil)
	if err != nil || len(got) != 1 {
		t.Fatalf("saves after the reload = %+v, %v", got, err)
	}
}
