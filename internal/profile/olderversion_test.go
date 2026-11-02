package profile

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestDownloadDepotCommand(t *testing.T) {
	got, err := downloadDepotCommand("413150", "413151", "7081337227611034738")
	if err != nil || got != "download_depot 413150 413151 7081337227611034738" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range [][3]string{{"413150", "413151", "1; quit"}, {"x", "1", "2"}, {"1", "", "2"}} {
		if _, err := downloadDepotCommand(bad[0], bad[1], bad[2]); err == nil {
			t.Fatalf("%v accepted", bad)
		}
	}
}

func TestDepotPaths(t *testing.T) {
	lib := filepath.Join("home", "u", "SteamLibrary")
	if got := game.LibraryOf(filepath.Join(lib, "steamapps", "common", "Stardew Valley")); got != lib {
		t.Fatalf("LibraryOf = %q", got)
	}
	if got := game.LibraryOf(filepath.Join("home", "u", "Games", "Stardew")); got != "" {
		t.Fatalf("a folder outside a library has library %q", got)
	}
	if got, want := steam.DepotDownloadDir(lib, "413150", "413151"), filepath.Join(lib, "steamapps", "content", "app_413150", "depot_413151"); got != want {
		t.Fatalf("DepotDownloadDir = %q, want %q", got, want)
	}
	if got, want := versionFolder("/data", "stardew", "99-20260101"), filepath.Join("/data", "versions", "stardew", "99-20260101"); got != want {
		t.Fatalf("versionFolder = %q", got)
	}
}

func TestOlderVersionFlow(t *testing.T) {
	e := newEnv(t)
	testfs.DataHome(t)
	info, _ := components.Game("stardew")
	root := t.TempDir()
	install := filepath.Join(root, "steamapps", "common", "Stardew Valley")
	writeFile(t, install, info.Marker, "game")
	writeFile(t, filepath.Join(root, "steamapps"), "appmanifest_413150.acf",
		"\"AppState\"\n{\n\"InstalledDepots\"\n{\n\"413151\"\n{\n\"manifest\" \"99\"\n\"size\" \"5\"\n}\n}\n}\n")
	st, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Update(func(s *settings.Settings) { s.GameFolders = map[string]string{"stardew": install} }); err != nil {
		t.Fatal(err)
	}
	svc := NewService(e.Store, t.TempDir(), st)
	p := mustCreate(t, e, "Old")

	if !svc.CanGetOlderVersion("stardew") || svc.CanGetOlderVersion("no-such-game") {
		t.Fatal("CanGetOlderVersion")
	}
	depots, err := svc.OlderVersionDepots("stardew")
	if err != nil || len(depots.Depots) != 1 || depots.Depots[0].Manifest != "99" {
		t.Fatalf("depots = %+v, %v", depots, err)
	}
	if _, err := svc.OlderVersionState("stardew", "1/../x"); err == nil {
		t.Fatal("a depot id with separators was accepted")
	}
	if got, err := svc.OlderVersionState("stardew", "413151"); err != nil || got.Ready {
		t.Fatalf("state before the download = %+v, %v", got, err)
	}
	if _, err := svc.AdoptOlderVersion(t.Context(), "stardew", p.ID, "413151", "99"); err == nil {
		t.Fatal("adopted an unfinished download")
	}
	depotDir := steam.DepotDownloadDir(root, "413150", "413151")
	writeFile(t, depotDir, info.Marker, "old")
	if got, _ := svc.OlderVersionState("stardew", "413151"); !got.Ready {
		t.Fatal("a finished download is not ready")
	}
	if _, err := svc.AdoptOlderVersion(t.Context(), "stardew", p.ID, "413151", "9;9"); err == nil {
		t.Fatal("a manifest id with punctuation was accepted")
	}
	adopted, err := svc.AdoptOlderVersion(t.Context(), "stardew", p.ID, "413151", "99")
	if err != nil {
		t.Fatal(err)
	}
	dir, ok := game.FolderPinDir(adopted.Install)
	if !ok || !safefs.IsFile(filepath.Join(dir, info.Marker)) || !safefs.IsFile(filepath.Join(depotDir, info.Marker)) {
		t.Fatalf("pin %q: copy or source missing", adopted.Install)
	}

	if _, err := svc.SetGameFolder("stardew", p.ID, t.TempDir()); err == nil {
		t.Fatal("a folder with no game was accepted")
	}
	if got, err := svc.SetGameFolder("stardew", p.ID, " "); err != nil || got.Install != "" {
		t.Fatalf("clearing = %+v, %v", got, err)
	}
	if got, err := svc.SetGameFolder("stardew", p.ID, install); err != nil || got.Install != game.FolderPin(install) {
		t.Fatalf("pinning = %+v, %v", got, err)
	}

	if !svc.store.ModsUpTo("stardew", p.ID).IsZero() {
		t.Fatal("a new profile has a cutoff")
	}
	if _, err := svc.SetModsUpTo("stardew", p.ID, "soon"); err == nil {
		t.Fatal("an unreadable date was accepted")
	}
	if _, err := svc.SetModsUpTo("stardew", p.ID, "2024-02-10"); err != nil {
		t.Fatal(err)
	}
	if want := source.ParseCutoff("2024-02-10"); !svc.store.ModsUpTo("stardew", p.ID).Equal(want) {
		t.Fatal("cutoff not stored")
	}
	if _, err := svc.SetModsUpTo("stardew", p.ID, ""); err != nil || !svc.store.ModsUpTo("stardew", p.ID).IsZero() {
		t.Fatalf("lifting the cutoff: %v", err)
	}
}
