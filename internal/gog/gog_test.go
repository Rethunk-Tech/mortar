package gog

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// testGame is a game the way a Game implementation describes it to the GOG locators.
var testGame = Game{ProductID: "1453375253", Folder: "Stardew Valley", Marker: "Stardew Valley.dll"}

func writeGame(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, testGame.Marker), []byte("dll"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLocateLinuxOfflineAndHeroic(t *testing.T) {
	home := t.TempDir()
	offline := filepath.Join(home, "GOG Games", "Stardew Valley", "game")
	writeGame(t, offline)
	heroicRoot := filepath.Join(home, "Games", "Heroic", "Stardew Valley")
	writeGame(t, heroicRoot)
	cfg := filepath.Join(home, ".config", "heroic", "gog_store", "installed.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"games":[{"appName":"1453375253","install_path":"` + filepath.ToSlash(heroicRoot) + `"},{"appName":"1","install_path":"/nope"}]}`
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got := Locate(home, testGame, Roots{})
	var offlineHit, heroicHit bool
	for _, in := range got {
		if in.Dir == offline && in.Store == StoreGOG {
			offlineHit = true
		}
		if in.Dir == heroicRoot && in.Store == StoreHeroic {
			heroicHit = true
		}
	}
	if !heroicHit {
		t.Fatalf("heroic missing: %#v", got)
	}
	if runtime.GOOS != "windows" && !offlineHit {
		t.Fatalf("offline missing: %#v", got)
	}
}

func TestHeroicKeyedJSONAndNestedGame(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "gog", "stardew")
	writeGame(t, filepath.Join(root, "game"))
	cfg := filepath.Join(home, ".config", "heroic", "gog_store", "installed.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"1453375253":{"appName":"1453375253","install_path":"` + filepath.ToSlash(root) + `"}}`
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got := Locate(home, testGame, Roots{})
	var found Install
	for _, in := range got {
		if in.Store == StoreHeroic {
			found = in
		}
	}
	want := filepath.Join(root, "game")
	if found.Dir != want {
		t.Fatalf("heroic nested = %#v, want %s", found, want)
	}
}

func TestLocateIgnoresMissing(t *testing.T) {
	if got := Locate(t.TempDir(), testGame, Roots{}); len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestMinigalaxyInstallDirFromItsConfig(t *testing.T) {
	home := t.TempDir()
	lib := filepath.Join(home, "Games", "GOG")
	writeGame(t, filepath.Join(lib, "Stardew Valley"))
	cfg := filepath.Join(home, ".var", "app", "io.github.sharkwouter.Minigalaxy", "config", "minigalaxy")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "config.json"), []byte(`{"install_dir": "`+filepath.ToSlash(lib)+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := Locate(home, testGame, Roots{})
	if len(got) != 1 || got[0].Store != StoreMinigalaxy || got[0].Dir != filepath.Join(lib, "Stardew Valley") {
		t.Fatalf("got %#v", got)
	}
}

func TestMinigalaxyFolderIsNotAlsoGOGs(t *testing.T) {
	home := t.TempDir()
	writeGame(t, filepath.Join(home, "GOG Games", "Stardew Valley"))
	cfg := filepath.Join(home, ".config", "minigalaxy")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "config.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Locate(home, testGame, Roots{}); len(got) != 1 || got[0].Store != StoreMinigalaxy {
		t.Fatalf("one install, credited to Minigalaxy: %#v", got)
	}
	if dirs := OfflineDirs(home, Roots{}); len(dirs) != 0 {
		t.Fatalf("Minigalaxy's default folder must leave GOG's list: %v", dirs)
	}
}
