package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestBepInExArchiveWithoutAManifestInstallsAsAPlugin(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "lbtokg.dll-17-1-0-0-1700625972.zip"), map[string]string{"lbtokg.dll": "MZ"})
	res, err := e.InstallArchive("lethal-company", p.ID, zip)
	if err != nil || len(res.Profile.Entries) != 1 {
		t.Fatalf("install = %+v, %v", res, err)
	}
	en := res.Profile.Entries[0]
	if !en.Package || len(en.Mods) != 1 || en.Mods[0].ID != "bepinex:lbtokg" {
		t.Fatalf("entry = %+v", en)
	}
	if err := e.SyncPackages("lethal-company", p.ID); err != nil {
		t.Fatal(err)
	}
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	if _, err := os.Stat(filepath.Join(dir, "BepInEx", "plugins", en.Key, "lbtokg.dll")); err != nil {
		t.Fatal(err)
	}
}
