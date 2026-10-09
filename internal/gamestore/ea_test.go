package gamestore

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

func eaGame() components.GameInfo {
	return components.GameInfo{
		ID: "x", Name: "X", Marker: "X.exe",
		Stores: components.GameStores{EA: &components.EAStore{Folder: "The X"}},
	}
}

func touchFile(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestEADirsAreWindowsOnlyAndUserFoldersComeFirst(t *testing.T) {
	env := map[string]string{"ProgramFiles": `C:\PF`, "ProgramFiles(x86)": `C:\PF86`}
	get := func(k string) string { return env[k] }
	got := eaDirs("windows", get, "D:/mine")
	want := []string{
		"D:/mine",
		filepath.Join(`C:\PF`, "EA Games"), filepath.Join(`C:\PF`, "Electronic Arts"),
		filepath.Join(`C:\PF86`, "EA Games"), filepath.Join(`C:\PF86`, "Electronic Arts"),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("windows dirs = %v, want %v", got, want)
	}
	if got := eaDirs("linux", get, "/mine"); !slices.Equal(got, []string{"/mine"}) {
		t.Fatalf("linux has no EA App folders of its own: %v", got)
	}
	if (eaStore{}).Launchers("linux") != nil || len((eaStore{}).Launchers("windows")) != 1 {
		t.Fatal("the EA App launcher is listed on Windows only")
	}
}

func TestEAFindsAnInstallInAUserFolderAtTheRootOrOneGameFolderDown(t *testing.T) {
	lib := t.TempDir()
	touchFile(t, filepath.Join(lib, "The X", "game", "Bin", "other"))
	touchFile(t, filepath.Join(lib, "The X", "game", "X.exe"))
	got := Discover(t.TempDir(), map[string][]string{LauncherEA: {lib}}, eaGame())
	want := []Install{{Store: StoreEA, Dir: filepath.Join(lib, "The X", "game")}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestEAIsAskedOnlyForGamesThatNameIt(t *testing.T) {
	lib := t.TempDir()
	touchFile(t, filepath.Join(lib, "The X", "X.exe"))
	roots := map[string][]string{LauncherEA: {lib}}
	g := eaGame()
	g.Stores.EA = nil
	if got := Discover(t.TempDir(), roots, g); len(got) != 0 {
		t.Fatalf("a game without an ea key must never reach the EA driver: %+v", got)
	}
	if Has(g, eaKey) || !Has(eaGame(), eaKey) {
		t.Fatal("Has must follow the catalog's ea field")
	}
}

func TestEARanksAfterSteamAndGOGSoTheyWinTheDefault(t *testing.T) {
	if Rank(StoreEA) <= Rank(StoreSteam) || Rank(StoreEA) <= Rank(StoreGOG) {
		t.Fatal("an EA install must not be chosen over a Steam or GOG one")
	}
	if LauncherOf(StoreEA) != LauncherEA {
		t.Fatal("an EA install is reported by the EA launcher")
	}
}

func TestEAInstallsInsideABottleAreFoundByTheBottlesDriver(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Bottles is Linux only")
	}
	home := t.TempDir()
	bottle := filepath.Join(home, ".local", "share", "bottles", "bottles", "Games")
	touchFile(t, filepath.Join(bottle, "bottle.yml"))
	game := filepath.Join(bottle, "drive_c", "Program Files", "EA Games", "The X")
	touchFile(t, filepath.Join(game, "X.exe"))
	g := eaGame()
	g.Stores = components.GameStores{Bottles: &components.BottlesStore{Folder: "The X"}}
	got := Discover(home, nil, g)
	want := []Install{{Store: StoreBottles, Dir: game, Prefix: bottle}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
