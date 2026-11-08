package cp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-lenientjson"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-smapi"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// check analyzes mods as one profile.
func check(mods []Mod) Findings {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m Mod) bool { return !m.Enabled })
	return analyze(Request{Enabled: enabled, All: mods})
}

// analyze is a check with the profile's load order, as Mortar's loader supplies it.
func analyze(req Request) Findings {
	req.Order = testOrder
	return Analyze(req)
}

// testOrder is the profile's folded ids in SMAPI's order.
func testOrder(mods []Mod) []string {
	in := make([]smapi.Mod, 0, len(mods))
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		row := smapi.Mod{
			UniqueID: m.UniqueID.Local(), Name: m.Name, Folder: m.Folder, ContentPackFor: m.ContentPackFor.Local(),
		}
		for _, d := range m.Dependencies {
			row.Dependencies = append(row.Dependencies, smapi.Dependency{UniqueID: d.UniqueID.Local(), IsRequired: d.Required})
		}
		in = append(in, row)
	}
	out := make([]string, 0, len(in))
	for _, row := range smapi.Resolve(in) {
		out = append(out, SMAPI(row.UniqueID).Fold())
	}
	return out
}

// packManifest is the manifest of a Content Patcher pack.
func packManifest(id string) string {
	return `{"Name":"` + id + `","UniqueID":"` + id + `","Version":"1","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
}

// diskPack writes files into a fresh folder and returns the mod installed from it, keyed and named by id.
func diskPack(t *testing.T, id string, files map[string]string) Mod {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		writeFile(t, root, rel, body)
	}
	return packFromDisk(Mod{Key: id, Name: id, UniqueID: SMAPI(id), Folder: root, Enabled: true})
}

// packFromDisk is the mod a profile scan reads from a folder: its manifest's ContentPackFor is what marks it as
// content for a framework.
func packFromDisk(m Mod) Mod {
	raw, err := safefs.ReadFile(filepath.Join(m.Folder, "manifest.json"))
	if err != nil {
		return m
	}
	var doc struct {
		ContentPackFor struct {
			UniqueID string `json:"UniqueID"`
		} `json:"ContentPackFor"`
	}
	if json.Unmarshal(lenientjson.Clean(raw), &doc) == nil && doc.ContentPackFor.UniqueID != "" {
		m.ContentPackFor = SMAPI(doc.ContentPackFor.UniqueID)
	}
	return m
}

func writeFile(t *testing.T, root, rel, body string) string {
	t.Helper()
	path := filepath.Join(root, rel)
	writeAt(t, path, []byte(body))
	return path
}

func writeAt(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

// touchFile puts a file's modification time back the way it was, after ageing it.
func touchFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
}

// dataHome points the caches at a temp folder, as the app's data folder would be.
func dataHome(t *testing.T) {
	t.Helper()
	SetCacheRoot(t.TempDir())
	t.Cleanup(func() { SetCacheRoot("") })
}

// resetCaches drops what one test's temp folders filled, so the next test reads its own packs.
func resetCaches() { Reset() }

// testdataFolder is a pack fixture in testdata.
func testdataFolder(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// settingPack is a pack with a config schema, its changes and its config.json.
func settingPack(t *testing.T, schema, changes, config string) Mod {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "manifest.json", packManifest("Pack.Compat"))
	content := `{"ConfigSchema":` + schema + `,"Changes":` + changes + `}`
	writeFile(t, root, "content.json", content)
	if config != "" {
		writeFile(t, root, "config.json", config)
	}
	return packFromDisk(Mod{Key: "compat", Name: "Compatibility Pack", UniqueID: "Pack.Compat", Folder: root, Enabled: true})
}

// contentPack is a pack with the given content.json.
func contentPack(t *testing.T, id, content string) Mod {
	t.Helper()
	return diskPack(t, id, map[string]string{"manifest.json": packManifest(id), "content.json": content})
}
