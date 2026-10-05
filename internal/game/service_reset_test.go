package game

import (
	"os"
	"strings"
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

func TestListComesFromTheCatalog(t *testing.T) {
	list, err := NewService(t.TempDir(), testStore(t)).List()
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %+v, %v", list, err)
	}
	sdv, lc := list[0], list[1]
	if sdv.ID != "stardew" || !sdv.Available || sdv.LoaderID != "smapi" || strings.Join(sdv.Sources, ",") != "nexus,github,moddrop" || sdv.AppID != "413150" {
		t.Fatalf("stardew row = %+v", sdv)
	}
	if lc.ID != "lethal-company" || lc.Available || lc.LoaderID != "bepinex5" || lc.AppID != "1966720" {
		t.Fatalf("lethal-company row = %+v", lc)
	}
}
