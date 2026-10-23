package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

// stageValheimMods writes three staged mod folders no SMAPI manifest describes: a Thunderstore package, a bare
// plugin DLL and a folder with nothing the loader reads.
func stageValheimMods(t *testing.T, mods string) {
	t.Helper()
	files := map[string]string{
		"Pkg-1/manifest.json":      `{"name":"Pkg","version_number":"1.2.0","namespace":"Author","dependencies":[]}`,
		"Pkg-1/plugins/Pkg.dll":    "x",
		"Raw-1/BepInEx/Cool.dll":   "x",
		"Loose-1/readme.txt":       "x",
		"Loose-1/notes/other.json": "{}",
	}
	for rel, body := range files {
		p := filepath.Join(mods, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func assertValheimStaged(t *testing.T, mods []ModPreview) {
	t.Helper()
	want := map[string]struct{ id, name, version string }{
		"Pkg-1":   {"thunderstore:Author-Pkg", "Pkg", "1.2.0"},
		"Raw-1":   {"bepinex:Cool", "Cool", ""},
		"Loose-1": {"folder:Loose-1", "Loose-1", ""},
	}
	if len(mods) != len(want) {
		t.Fatalf("mods = %#v", mods)
	}
	for _, m := range mods {
		w, ok := want[filepath.Base(m.SourcePath)]
		if !ok || string(m.ID) != w.id || m.Name != w.name || m.Version != w.version {
			t.Fatalf("mod %q = %#v, want %+v", m.SourcePath, m, w)
		}
	}
}

func TestMO2ImportReadsNonSMAPIGameModsByTheirLoader(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	inst := filepath.Join(home, "AppData", "Local", "ModOrganizer", "Valheim")
	stageValheimMods(t, filepath.Join(inst, "mods"))
	profile := filepath.Join(inst, "profiles", "Default")
	if err := os.MkdirAll(profile, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inst, "ModOrganizer.ini"), []byte("[General]\ngameName=Valheim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "modlist.txt"), []byte("+Pkg-1\n+Raw-1\n+Loose-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := Preview(home, "", "valheim", "", KindMO2, "Default")
	if err != nil {
		t.Fatal(err)
	}
	assertValheimStaged(t, preview.Mods)
}

func TestVortexImportReadsNonSMAPIGameModsByTheirLoader(t *testing.T) {
	home := t.TempDir()
	state := filepath.Join(configDir(home), "Vortex", "state.v2")
	stageValheimMods(t, filepath.Join(configDir(home), "Vortex", "valheim", "mods"))
	if err := os.MkdirAll(state, 0o750); err != nil {
		t.Fatal(err)
	}
	db, err := leveldb.OpenFile(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	kvs := map[string]string{
		"persistent###profiles###p1###id":     `"p1"`,
		"persistent###profiles###p1###gameId": `"valheim"`,
		"persistent###profiles###p1###name":   `"Run"`,
	}
	for _, id := range []string{"Pkg-1", "Raw-1", "Loose-1"} {
		kvs["persistent###profiles###p1###modState###"+id+"###enabled"] = `true`
		kvs["persistent###mods###valheim###"+id] = `{"id":"` + id + `","installationPath":"` + id + `"}`
	}
	for k, v := range kvs {
		if err := db.Put([]byte(k), []byte(v), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	preview, err := Preview(home, "", "valheim", "", KindVortex, "p1")
	if err != nil {
		t.Fatal(err)
	}
	assertValheimStaged(t, preview.Mods)
}
