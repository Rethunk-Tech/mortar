package profile

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestFixStaleManifestSetsOnlyTheVersionInStoreAndProfile(t *testing.T) {
	e := newEnv(t)
	e.item(t, "nexus-1-2", map[string]string{
		"Pack/A/manifest.json": `{
  // the author forgot this one
  "Name": "A", "Author": "me", "Version": "1.0.0", "UniqueID": "X.A",
  "Dependencies": [{ "UniqueID": "X.B", "MinimumVersion": "1.0.0" }],
}`,
		"Pack/B/manifest.json": manifestJSON("X.B"),
	})
	p, _ := e.Create("stardew", "P")
	if _, err := e.AddEntry("stardew", p.ID, "nexus-1-2", Source{Kind: KindNexus}); err != nil {
		t.Fatal(err)
	}
	if err := e.FixStaleManifest("stardew", p.ID, "nexus-1-2", "x.a", "1.2.0"); err != nil {
		t.Fatal(err)
	}
	storeDir, _ := e.items.Path("stardew", "nexus-1-2")
	for _, root := range []string{storeDir, filepath.Join(e.mods(p.ID), "nexus-1-2")} {
		b, err := fsx.ReadFile(filepath.Join(root, "Pack/A/manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		got := string(b)
		if !strings.Contains(got, `"Version": "1.2.0"`) || !strings.Contains(got, "// the author forgot this one") ||
			!strings.Contains(got, `"MinimumVersion": "1.0.0"`) {
			t.Fatalf("%s manifest:\n%s", root, got)
		}
		other, _ := fsx.ReadFile(filepath.Join(root, "Pack/B/manifest.json"))
		if !strings.Contains(string(other), `"Version":"1.0.0"`) {
			t.Fatalf("another mod's manifest changed: %s", other)
		}
	}
	got, _ := e.read("stardew", p.ID)
	if got.Entries[0].Mods[0].Version != "1.2.0" {
		t.Fatalf("entry version = %+v", got.Entries[0].Mods[0])
	}
	if drift, err := e.ScanModsDrift("stardew", p.ID); err != nil || len(drift) != 0 {
		t.Fatalf("the fix must not read as a change outside Mortar: %+v, %v", drift, err)
	}
}

func TestFixStaleManifestLeavesAManifestFromAnotherFile(t *testing.T) {
	e := newEnv(t)
	e.item(t, "nexus-1-2", map[string]string{"Pack/A/manifest.json": manifestJSON("X.A")})
	p, _ := e.Create("stardew", "P")
	if _, err := e.AddEntry("stardew", p.ID, "nexus-1-2", Source{Kind: KindNexus}); err != nil {
		t.Fatal(err)
	}
	if err := e.FixStaleManifest("stardew", p.ID, "nexus-1-2", "X.Missing", "9.9.9"); err != nil {
		t.Fatalf("a mod the download does not hold is skipped, not an error: %v", err)
	}
	b, _ := fsx.ReadFile(filepath.Join(e.mods(p.ID), "nexus-1-2", "Pack/A/manifest.json"))
	if strings.Contains(string(b), "9.9.9") {
		t.Fatalf("changed: %s", b)
	}
}

func TestFixStaleManifestLeavesAModFromAnExtraFile(t *testing.T) {
	e := newEnv(t)
	e.item(t, "nexus-1-2", map[string]string{"Main/manifest.json": manifestJSON("X.Main")})
	p, _ := e.Create("stardew", "P")
	if _, err := e.AddEntry("stardew", p.ID, "nexus-1-2", Source{Kind: KindNexus}); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(e.mods(p.ID), "nexus-1-2", "nexus-1-3", "Optional")
	writeFile(t, extra, "manifest.json", manifestJSON("X.Optional"))
	if _, err := e.update("stardew", p.ID, func(pr *Profile, _ string) error {
		pr.Entries[0].ExtraStoreKeys = []string{"nexus-1-3"}
		pr.Entries[0].Mods = append(pr.Entries[0].Mods, EntryMod{UniqueID: "X.Optional", Version: "1.0.0", Folder: "nexus-1-3/Optional"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := e.FixStaleManifest("stardew", p.ID, "nexus-1-2", "X.Optional", "2.5.0"); err != nil {
		t.Fatal(err)
	}
	b, _ := fsx.ReadFile(filepath.Join(extra, "manifest.json"))
	if strings.Contains(string(b), "2.5.0") {
		t.Fatalf("the main download's version was written into another file's mod: %s", b)
	}
}
