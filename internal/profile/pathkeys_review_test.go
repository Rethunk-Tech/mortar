package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// An entry's key is <store item>#<path>, and the entry that holds an archive's Tray files is a key no path can produce. A file
// at the archive root named tray must not become that entry: it is a mod file like any other and is laid out.
func TestAFileNamedTrayIsALaidOutModFileNotTheTrayEntry(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"tray": "t", "a.package": "1"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10)); err != nil {
		t.Fatal(err)
	}
	owners, err := e.PackageFileOwners(folderGame, p.ID)
	if err != nil || owners["Mods/tray"] == "" || owners["Mods/a.package"] == "" {
		t.Fatalf("owners = %v, %v", owners, err)
	}
}

// A path that climbs out of the Tray folder is refused before anything is written, whatever laid it out.
func TestPlaceTrayRefusesAPathOutsideTheTrayFolder(t *testing.T) {
	t.Parallel()
	e, tray := trayEnv(t)
	src := t.TempDir()
	if err := writeSource(filepath.Join(src, "x.trayitem")); err != nil {
		t.Fatal(err)
	}
	p := &Profile{ID: "p"}
	_, err := e.placeTray(folderGame, p, installer.Archive{Dir: src}, []installer.File{{Src: "x.trayitem", Target: "tray", Rel: "../escaped.trayitem"}}, nil)
	if err == nil {
		t.Fatal("a path leaving the Tray folder was accepted")
	}
	if got := trayTree(t, filepath.Dir(tray)); len(got) != 0 {
		t.Fatalf("files written: %v", got)
	}
}

func writeSource(path string) error {
	return os.WriteFile(path, []byte("x"), 0o600)
}

func TestARootFileNamedTrayAndTrayFilesAreTwoDistinctEntries(t *testing.T) {
	t.Parallel()
	e, tray := trayEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"tray": "t", "House.trayitem": "h"})
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10))
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	trays := 0
	for _, en := range res.Profile.Entries {
		keys[en.Key] = true
		if en.Tray {
			trays++
		}
	}
	if len(res.Profile.Entries) != 2 || len(keys) != 2 || trays != 1 {
		t.Fatalf("entries = %v, tray entries %d", keysOf(res.Profile), trays)
	}
	owners, _ := e.PackageFileOwners(folderGame, p.ID)
	if owners["Mods/tray"] == "" || len(trayTree(t, tray)) != 1 {
		t.Fatalf("owners %v, tray %v", owners, trayTree(t, tray))
	}
}

// On a case-insensitive file system House.trayitem and house.trayitem are one file. The fold is injected so this runs
// on any system: with it, the second profile shares the first's file instead of placing another, and removing the first
// profile leaves the file for the second.
func TestTrayOwnersAreKeyedByFoldedCase(t *testing.T) {
	trayFold = strings.ToLower
	t.Cleanup(func() { trayFold = fsx.FoldCase })
	e, tray := trayEnv(t)
	p1, _ := e.Create(folderGame, "A")
	p2, _ := e.Create(folderGame, "B")
	if _, err := e.InstallSource(t.Context(), folderGame, p1.ID, zipOf(t, "a.zip", map[string]string{"House.trayitem": "t"}), cfSource(10)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.InstallSource(t.Context(), folderGame, p2.ID, zipOf(t, "b.zip", map[string]string{"house.trayitem": "t"}), Source{Kind: KindCurseForge, Name: "Other", ModID: 8, FileID: 1}); err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); len(got) != 1 {
		t.Fatalf("one file under two cases: %v", got)
	}
	if err := e.Delete(folderGame, p1.ID); err != nil {
		t.Fatal(err)
	}
	if got := trayTree(t, tray); len(got) != 1 {
		t.Fatalf("removing one profile deleted the other's file: %v", got)
	}
}
