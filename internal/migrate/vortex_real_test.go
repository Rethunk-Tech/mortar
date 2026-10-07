package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

// Vortex 2.8 stores a mod installed from a zip under an archive-named folder holding the archive's own top folder,
// with no uniqueId attribute and a profile whose game id is Vortex's "stardewvalley".
func TestVortex28ProfileWithNestedZipMod(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(configDir(home), "Vortex")
	manifest := filepath.Join(root, "stardewvalley", "mods", "TestRetexture-1.0.1", "TestMod", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o750); err != nil {
		t.Fatal(err)
	}
	body := `{"Name":"Test Retexture","Author":"Tester","Version":"1.0.1","UniqueID":"Tester.TestRetexture"}`
	if err := os.WriteFile(manifest, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := leveldb.OpenFile(filepath.Join(root, "state.v2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	const mod = "persistent###mods###stardewvalley###TestRetexture-1.0.1###"
	for k, v := range map[string]string{
		"persistent###profiles###6LuTh37Vu###id":                                       `"6LuTh37Vu"`,
		"persistent###profiles###6LuTh37Vu###gameId":                                   `"stardewvalley"`,
		"persistent###profiles###6LuTh37Vu###name":                                     `"Default"`,
		"persistent###profiles###6LuTh37Vu###modState###TestRetexture-1.0.1###enabled": `true`,
		mod + "id":                           `"TestRetexture-1.0.1"`,
		mod + "installationPath":             `"TestRetexture-1.0.1"`,
		mod + "attributes###name":            `"TestRetexture-1.0.1"`,
		mod + "attributes###manifestVersion": `"1.0.1"`,
		"settings###gameMode###discovered###stardewvalley###path": `"C:\\Games\\Stardew Valley"`,
		"settings###profiles###lastActiveProfile###stardewvalley": `"6LuTh37Vu"`,
	} {
		if err := db.Put([]byte(k), []byte(v), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	sources, err := Detect(home, t.TempDir(), "stardew", root)
	if err != nil || len(sources) != 1 || sources[0].Kind != KindVortex || sources[0].Profiles[0].Mods != 1 {
		t.Fatalf("sources = %#v, %v", sources, err)
	}
	preview, err := Preview(home, t.TempDir(), "stardew", root, KindVortex, "6LuTh37Vu")
	if err != nil || len(preview.Mods) != 1 || preview.Mods[0].ID != "smapi:Tester.TestRetexture" || !preview.Mods[0].Enabled {
		t.Fatalf("preview = %#v, %v", preview, err)
	}
}
