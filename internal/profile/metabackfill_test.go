package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// A package installed before its manifest.json and icon.png were placed gains them on the next sync, from the store's
// extraction, and a further sync changes nothing.
func TestSyncPlacesMetaFilesAnEarlierInstallLeftOut(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "CatLib-1.0.0.zip"), map[string]string{
		"manifest.json": `{"name":"CatLib","version_number":"1.0.0","author":"Cat"}`,
		"icon.png":      "PNG",
		"CatLib.dll":    "MZ",
	})
	res, err := e.InstallArchive(t.Context(), "lethal-company", p.ID, zip)
	if err != nil || len(res.Profile.Entries) != 1 {
		t.Fatalf("install = %+v, %v", res, err)
	}
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	sync := func() []byte {
		t.Helper()
		if err := e.SyncPackages("lethal-company", p.ID); err != nil {
			t.Fatal(err)
		}
		b, err := fsx.ReadFile(filepath.Join(dir, placedFile))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	sync()
	pkg := filepath.Join(dir, "BepInEx", "plugins", "CatLib")
	// The old layout: no meta files placed, none recorded.
	for _, f := range []string{"manifest.json", "icon.png"} {
		if err := os.Remove(filepath.Join(pkg, f)); err != nil {
			t.Fatal(err)
		}
	}
	rec := readPlaced(dir)
	delete(rec, "BepInEx/plugins/CatLib/manifest.json")
	delete(rec, "BepInEx/plugins/CatLib/icon.png")
	// An earlier build also laid the store's marker out and recorded it.
	marker := filepath.Join(pkg, ".complete")
	if err := os.WriteFile(marker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	rec["BepInEx/plugins/CatLib/.complete"] = placedRec{Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", Size: 0}
	if err := writePlaced(dir, rec); err != nil {
		t.Fatal(err)
	}

	first := sync()
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the store's marker was placed")
	}
	for _, f := range []string{"manifest.json", "icon.png"} {
		if _, err := os.Stat(filepath.Join(pkg, f)); err != nil {
			t.Fatal(err)
		}
		if _, ok := readPlaced(dir)["BepInEx/plugins/CatLib/"+f]; !ok {
			t.Fatalf("%s not recorded", f)
		}
	}
	if second := sync(); string(second) != string(first) {
		t.Fatalf("second sync changed the record:\n%s\n%s", first, second)
	}
}
