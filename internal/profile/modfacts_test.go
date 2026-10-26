package profile

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// addPackage puts a Thunderstore package's files in the store under its lower-case name for Lethal Company and records it in p.
func addPackage(t *testing.T, e env, p Profile, name string, files map[string]string) {
	t.Helper()
	key := strings.ToLower(name)
	src := t.TempDir()
	for rel, body := range files {
		writeFile(t, src, rel, body)
	}
	if err := addAnyDir(t, e.items, "lethal-company", key, src); err != nil {
		t.Fatal(err)
	}
	if _, err := e.update("lethal-company", p.ID, func(p *Profile, _ string) error {
		p.Entries = append(p.Entries, Entry{
			Key: key, Package: true, Source: Source{Kind: KindThunderstore, Name: name, Version: "1.0.0"},
			Mods: []Component{{ID: mod.NewID(mod.FormatThunderstore, name), Name: name, Version: "1.0.0"}},
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestModFactsReadsWhatABepInExPackageShipsAndWhatTheIndexSays(t *testing.T) {
	// The fixture assembly declares plugin com.fixture.plugin 1.2.3 with hard dependencies com.fixture.hard and
	// com.fixture.min, and a soft one that is left out.
	dll, err := os.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	e := newEnv(t)
	p, err := e.Create("lethal-company", "Lobby")
	if err != nil {
		t.Fatal(err)
	}
	addPackage(t, e, p, "Ns-Plug", map[string]string{"manifest.json": "{}", "BepInEx/plugins/mod.dll": string(dll)})
	addPackage(t, e, p, "Ns-Patch", map[string]string{"BepInEx/patchers/p.dll": "patcher"})
	addPackage(t, e, p, "Ns-Both", map[string]string{"plugins/mod.dll": string(dll), "patchers/p.dll": "patcher"})
	addPackage(t, e, p, "Ns-Lib", map[string]string{"lib.dll": "not an assembly"})
	addPackage(t, e, p, "Ns-Sounds", map[string]string{"manifest.json": "{}", "sounds/a.ogg": "ogg"})

	if _, err := e.update("lethal-company", p.ID, func(p *Profile, _ string) error {
		p.Entries = append(p.Entries, Entry{
			Key: "gh", Source: Source{Kind: KindGitHub, Repo: "Owner/Repo", Version: "v1.0.0"},
			Mods: []Component{{ID: "github:Owner/Repo", Name: "Repo", Version: "1.0.0"}},
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var asked, askedGitHub map[string]string
	e.SourceFacts = func(game, kind string, installed map[string]string) map[string]source.Facts {
		if kind == KindGitHub {
			askedGitHub = installed
			return map[string]source.Facts{"owner/repo": {Latest: "1.1.0", Downloads: 40}}
		}
		asked = installed
		return map[string]source.Facts{"ns-plug": {
			Released: "2025-12-01T00:00:00Z", Latest: "2.0.0", LatestReleased: "2026-01-15T10:00:00Z",
			Updated: "2026-01-16", Rating: 7, Downloads: 900, Categories: []string{"Tools"},
		}}
	}
	got, err := e.ModFacts("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if asked["ns-plug"] != "1.0.0" || len(asked) != 5 {
		t.Fatalf("the index was asked %v", asked)
	}
	if askedGitHub["owner/repo"] != "v1.0.0" || len(askedGitHub) != 1 {
		t.Fatalf("github was asked %v", askedGitHub)
	}
	if g := got["github:Owner/Repo"]; g.Latest != "1.1.0" || g.Downloads != 40 {
		t.Fatalf("github mod = %+v", g)
	}
	plug := got["thunderstore:Ns-Plug"]
	if !slices.Equal(plug.IDs, []string{"com.fixture.plugin"}) || !slices.Equal(plug.PluginVersions, []string{"1.2.3"}) ||
		!slices.Equal(plug.HardDeps, []string{"com.fixture.hard", "com.fixture.min"}) || plug.Kind != KindPlugin {
		t.Fatalf("plugin package = %+v", plug)
	}
	if plug.Released != "2025-12-01T00:00:00Z" || plug.Latest != "2.0.0" || plug.LatestReleased != "2026-01-15T10:00:00Z" ||
		plug.Updated != "2026-01-16" || plug.Rating != 7 || plug.Downloads != 900 || !slices.Equal(plug.Categories, []string{"Tools"}) {
		t.Fatalf("source facts = %+v", plug)
	}
	for key, kind := range map[string]string{"Ns-Patch": KindPatcher, "Ns-Both": KindBoth, "Ns-Lib": KindLibrary, "Ns-Sounds": KindAssets} {
		f := got["thunderstore:"+key]
		if f.Kind != kind || f.Latest != "" {
			t.Errorf("%s = %+v, want kind %s and no index facts", key, f, kind)
		}
	}
}

func TestModFactsNameTheIDsOfManifestAndFolderMods(t *testing.T) {
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	if _, err := e.update("stardew", p.ID, func(p *Profile, _ string) error {
		p.Entries = append(p.Entries,
			Entry{Key: "cp", Source: Source{Kind: KindNexus}, Mods: []Component{{ID: "smapi:Pathoschild.ContentPatcher", Name: "Content Patcher"}}},
			Entry{Key: "vs", Source: Source{Kind: KindNexus}, Mods: []Component{{ID: "folder:vs", Name: "VS Mod", ModID: "vsmod"}}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got, err := e.ModFacts("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if f := got["smapi:Pathoschild.ContentPatcher"]; !slices.Equal(f.IDs, []string{"Pathoschild.ContentPatcher"}) || f.Kind != "" {
		t.Fatalf("SMAPI mod = %+v", f)
	}
	if f := got["folder:vs"]; !slices.Equal(f.IDs, []string{"vsmod"}) {
		t.Fatalf("folder mod = %+v", f)
	}
}

func TestPackageScansAreCachedAndBounded(t *testing.T) {
	dll, err := os.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	e := newEnv(t)
	p, err := e.Create("lethal-company", "Lobby")
	if err != nil {
		t.Fatal(err)
	}
	addPackage(t, e, p, "Ns-Plug", map[string]string{"mod.dll": string(dll)})
	entry := Entry{Key: "ns-plug", Package: true, Source: Source{Kind: KindThunderstore, Name: "Ns-Plug"}}
	first, ok := e.scanPackage("lethal-company", entry)
	if !ok || !slices.Equal(first.ids, []string{"com.fixture.plugin"}) {
		t.Fatalf("scan = %+v, %v", first, ok)
	}
	// The DLL is gone from the store, so only the cache can answer the same again.
	dir, err := e.items.Path("lethal-company", "ns-plug")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if again, ok := e.scanPackage("lethal-company", entry); !ok || !slices.Equal(again.ids, first.ids) || again.kind != first.kind {
		t.Fatalf("second scan = %+v, %v", again, ok)
	}
	for i := range 2 * scanCacheMax {
		e.scans.put("k"+strconv.Itoa(i), packageScan{kind: KindAssets})
	}
	if n := len(e.scans.m); n > scanCacheMax {
		t.Fatalf("cache holds %d scans, bound %d", n, scanCacheMax)
	}
}
