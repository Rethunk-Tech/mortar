package game

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

func find(t *testing.T, list []StoreApp, id string) StoreApp {
	t.Helper()
	for _, l := range list {
		if l.ID == id {
			return l
		}
	}
	t.Fatalf("no launcher %q in %+v", id, list)
	return StoreApp{}
}

func TestLaunchersReportFoundFoldersGamesAndWhereTheyLooked(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the launcher set differs per OS; Linux has every kind")
	}
	h := home(t)
	if err := os.MkdirAll(filepath.Join(h, ".var", "app", "com.heroicgameslauncher.hgl", "config", "heroic", "gog_store"), 0o700); err != nil {
		t.Fatal(err)
	}
	list, err := Launchers(h, settings.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	st := find(t, list, LauncherSteam)
	if !st.Found || st.Roots[0] != filepath.Join(h, ".local", "share", "Steam") || len(st.Games) != 1 || st.Games[0].ID != "stardew" {
		t.Fatalf("steam = %+v", st)
	}
	if hr := find(t, list, LauncherHeroic); !hr.Found || len(hr.Looked) != 3 {
		t.Fatalf("Flatpak Heroic's config folder must count: %+v", hr)
	}
	lu := find(t, list, LauncherLutris)
	if lu.Found || len(lu.Looked) != 3 || len(lu.Games) != 0 {
		t.Fatalf("lutris = %+v", lu)
	}

	custom := t.TempDir()
	s := settings.Defaults()
	s.LauncherRoots[LauncherLutris] = []string{custom}
	list, err = Launchers(h, s)
	if err != nil {
		t.Fatal(err)
	}
	if lu := find(t, list, LauncherLutris); !lu.Found || len(lu.Custom) != 1 || lu.Roots[0] != custom || lu.Looked[0] != custom {
		t.Fatalf("a chosen folder is searched first: %+v", lu)
	}
}

func TestValidateLauncherRoot(t *testing.T) {
	steamRoot := filepath.Join(home(t), ".local", "share", "Steam")
	if err := ValidateLauncherRoot(LauncherSteam, steamRoot); err != nil {
		t.Errorf("a Steam folder: %v", err)
	}
	if err := ValidateLauncherRoot(LauncherSteam, t.TempDir()); err == nil {
		t.Error("a folder without steamapps must be refused")
	}
	if err := ValidateLauncherRoot("nope", t.TempDir()); err == nil {
		t.Error("an unknown launcher must be refused")
	}
}
