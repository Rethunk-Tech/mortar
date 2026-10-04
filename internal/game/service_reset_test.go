package game

import (
	"os"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestResetInstallRefusesNonGameFolder(t *testing.T) {
	dir := t.TempDir()
	store := testStore(t)
	if _, err := store.Update(func(s *settings.Settings) {
		*s = settings.Defaults()
		s.GameFolders = map[string]string{"stardew": dir}
	}); err != nil {
		t.Fatal(err)
	}
	err := NewService(t.TempDir(), store).ResetInstall("stardew")
	if err == nil {
		t.Fatal("ResetInstall accepted a non-game folder")
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatalf("non-game folder was removed: %v", statErr)
	}
}

func TestResetInstallRemovesGameFolder(t *testing.T) {
	dir := stardewFolder(t)
	store := testStore(t)
	if _, err := store.Update(func(s *settings.Settings) {
		*s = settings.Defaults()
		s.GameFolders = map[string]string{"stardew": dir}
	}); err != nil {
		t.Fatal(err)
	}
	if err := NewService(t.TempDir(), store).ResetInstall("stardew"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("game folder still exists, stat error: %v", err)
	}
}
