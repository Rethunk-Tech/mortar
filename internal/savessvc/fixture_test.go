package savessvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// backupService is a service over fresh data and config homes, with the game's saves folder (empty) it scans.
func backupService(t *testing.T) (*Service, string) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	savesDir := filepath.Join(cfg, "StardewValley", "Saves")
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	return &Service{settings: store, scanners: map[string]*saves.Scanner{"stardew": {Dir: savesDir}}}, savesDir
}
