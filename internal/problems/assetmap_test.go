package problems

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func TestAssetIndexFromFixturePacks(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	low := loadPack(t, "Pack.Low", "Characters/Abigail", "Low")
	high := loadPack(t, "Pack.High", "Characters/Abigail", "High")
	data := dataConflictPack(t, "Pack.Data", "alpha")
	page := AssetMapOf([]Installed{low, high, data}, "", 0)
	if page.Total < 2 {
		t.Fatalf("targets %+v", page.Targets)
	}
	abigail := findTarget(page.Targets, "characters/abigail", "")
	if abigail.Winner != "Pack.High" {
		t.Fatalf("winner %q mods %+v", abigail.Winner, abigail.Mods)
	}
	var sawHigh bool
	for _, m := range abigail.Mods {
		if m.ModID == "Pack.High" {
			sawHigh = m.Winner && m.Action == kindLoad && m.LoadOrder == 1
		}
		if m.ModID == "Pack.Low" && m.Winner {
			t.Fatalf("low marked winner %+v", m)
		}
	}
	if !sawHigh {
		t.Fatalf("high winner missing %+v", abigail.Mods)
	}
	objects := findTarget(page.Targets, "data/objects", "123")
	if objects.Winner != "Pack.Data" || len(objects.Mods) != 1 || objects.Mods[0].Action != kindEditData {
		t.Fatalf("objects %+v", objects)
	}
}

func TestWhoChangesFuzzyAndAlias(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	portrait := loadPack(t, "Pack.Portrait", "Portraits/Abigail", "")
	house := loadPack(t, "Pack.House", "Maps/FarmHouse", "")
	mods := []Installed{portrait, house}

	got := WhoChangesOf(mods, "abigail")
	if len(got.Targets) != 1 || got.Targets[0].Target != "portraits/abigail" {
		t.Fatalf("alias %+v", got.Targets)
	}
	got = WhoChangesOf(mods, "farmhouse")
	if len(got.Targets) != 1 || got.Targets[0].Target != "maps/farmhouse" {
		t.Fatalf("farmhouse %+v", got.Targets)
	}
	got = WhoChangesOf(mods, "123")
	if len(got.Targets) != 0 {
		t.Fatalf("no data key 123, got %+v", got.Targets)
	}
	data := dataConflictPack(t, "Pack.Obj", "x")
	got = WhoChangesOf([]Installed{data}, "123")
	if len(got.Targets) != 1 || got.Targets[0].Key != "123" {
		t.Fatalf("data key %+v", got.Targets)
	}
}

func TestAssetIndexExclusiveLoadsHaveNoWinner(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	a := loadPack(t, "Pack.A", "Maps/Town", "Exclusive")
	b := loadPack(t, "Pack.B", "Maps/Town", "Exclusive")
	page := WhoChangesOf([]Installed{a, b}, "town")
	if len(page.Targets) != 1 || page.Targets[0].Winner != "" {
		t.Fatalf("exclusive loads %+v", page.Targets)
	}
	for _, m := range page.Targets[0].Mods {
		if m.Winner {
			t.Fatalf("no exclusive winner %+v", m)
		}
	}
}

func TestAssetIndexReplacedXnb(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Characters"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"Name":"Xnb.Mod","UniqueID":"Xnb.Mod","Version":"1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(root, "Characters", "Abigail.xnb"), []byte("xnb"), 0o600); err != nil {
		t.Fatal(err)
	}
	page := AssetMapOf([]Installed{{
		Key: "Xnb.Mod", Enabled: true, Folder: root, Name: "Xnb.Mod", UniqueID: "Xnb.Mod",
	}}, "", 0)
	if page.Total != 1 || page.Targets[0].Target != "characters/abigail" || page.Targets[0].Mods[0].Action != kindLoad {
		t.Fatalf("xnb %+v", page.Targets)
	}
}

func loadPack(t *testing.T, id, target, priority string) Installed {
	t.Helper()
	root := t.TempDir()
	writeManifest(t, root, id)
	pri := ""
	if priority != "" {
		pri = `,"Priority":"` + priority + `"`
	}
	content := `{"Changes":[{"Action":"Load","Target":"` + target + `","FromFile":"a.png"` + pri + `}]}`
	if err := fsx.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(root, "a.png"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return fromDisk(Installed{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
}

func findTarget(targets []AssetTarget, name, key string) AssetTarget {
	for _, t := range targets {
		if t.Target == name && t.Key == key {
			return t
		}
	}
	return AssetTarget{}
}
