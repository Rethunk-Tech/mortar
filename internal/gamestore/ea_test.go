package gamestore

import (
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

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
	testfs.WriteAt(t, p, nil)
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

func TestEAFindsAnInstallWhoseMarkerSitsInAFolderBelowTheRoot(t *testing.T) {
	lib := t.TempDir()
	touchFile(t, filepath.Join(lib, "The X", "Game", "Bin", "X.exe"))
	g := eaGame()
	g.MarkerDir = "Game/Bin"
	got := Discover(t.TempDir(), map[string][]string{LauncherEA: {lib}}, g)
	want := []Install{{Store: StoreEA, Dir: filepath.Join(lib, "The X")}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestEAFindsAnInstallOnAnotherDriveByItsRegistryEntry(t *testing.T) {
	away := t.TempDir()
	touchFile(t, filepath.Join(away, "X.exe"))
	var asked string
	lookupEARegistry = func(key string) string { asked = key; return away }
	t.Cleanup(func() { lookupEARegistry = eaRegistryDir })
	g := eaGame()
	g.Stores.EA.Registry = `SOFTWARE\X`
	got := Discover(t.TempDir(), nil, g)
	if want := []Install{{Store: StoreEA, Dir: away}}; !slices.Equal(got, want) || asked != `SOFTWARE\X` {
		t.Fatalf("got %+v (asked %q), want %+v", got, asked, want)
	}
	lib := t.TempDir()
	touchFile(t, filepath.Join(lib, "The X", "X.exe"))
	lookupEARegistry = func(string) string { return filepath.Join(lib, "The X") }
	if got := Discover(t.TempDir(), map[string][]string{LauncherEA: {lib}}, g); len(got) != 1 {
		t.Fatalf("an install in a library folder and in the registry is listed once: %+v", got)
	}
}
