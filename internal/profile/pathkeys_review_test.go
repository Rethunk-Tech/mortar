package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/installer"
)

// An entry's key is <store item>#<path>, and the entry that holds an archive's Tray files is <store item>#tray. A file
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
