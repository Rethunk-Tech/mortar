package problems

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/testenv/testfs"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

func TestAssetIndexFromFixturePacks(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	low := loadPack(t, "Pack.Low", "Characters/Abigail", "Low")
	high := loadPack(t, "Pack.High", "Characters/Abigail", "High")
	data := dataConflictPack(t, "Pack.Data", "alpha")
	page := AssetMapOf(buildAssetIndex([]Installed{low, high, data}), "", false, 0)
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
			sawHigh = m.Winner && m.Action == kindLoad && m.LoadOrder == 2
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
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	portrait := loadPack(t, "Pack.Portrait", "Portraits/Abigail", "")
	house := loadPack(t, "Pack.House", "Maps/FarmHouse", "")
	mods := []Installed{portrait, house}

	got := WhoChangesOf(buildAssetIndex(mods), "abigail")
	if len(got.Targets) != 1 || got.Targets[0].Target != "portraits/abigail" {
		t.Fatalf("alias %+v", got.Targets)
	}
	got = WhoChangesOf(buildAssetIndex(mods), "farmhouse")
	if len(got.Targets) != 1 || got.Targets[0].Target != "maps/farmhouse" {
		t.Fatalf("farmhouse %+v", got.Targets)
	}
	got = WhoChangesOf(buildAssetIndex(mods), "123")
	if len(got.Targets) != 0 {
		t.Fatalf("no data key 123, got %+v", got.Targets)
	}
	data := dataConflictPack(t, "Pack.Obj", "x")
	got = WhoChangesOf(buildAssetIndex([]Installed{data}), "123")
	if len(got.Targets) != 1 || got.Targets[0].Key != "123" {
		t.Fatalf("data key %+v", got.Targets)
	}
}

func TestAssetIndexExclusiveLoadsHaveNoWinner(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	a := loadPack(t, "Pack.A", "Maps/Town", "Exclusive")
	b := loadPack(t, "Pack.B", "Maps/Town", "Exclusive")
	page := WhoChangesOf(buildAssetIndex([]Installed{a, b}), "town")
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
	page := AssetMapOf(buildAssetIndex([]Installed{{
		Key: "Xnb.Mod", Enabled: true, Folder: root, Name: "Xnb.Mod", UniqueID: "Xnb.Mod",
	}}), "", false, 0)
	if page.Total != 1 || page.Targets[0].Target != "characters/abigail" || page.Targets[0].Mods[0].Action != kindLoad {
		t.Fatalf("xnb %+v", page.Targets)
	}
}

func loadPack(t *testing.T, id, target, priority string) Installed {
	t.Helper()
	pri := ""
	if priority != "" {
		pri = `,"Priority":"` + priority + `"`
	}
	content := `{"Changes":[{"Action":"Load","Target":"` + target + `","FromFile":"a.png"` + pri + `}]}`
	return diskPack(t, id, map[string]string{"manifest.json": cpManifest(id), "content.json": content, "a.png": "x"})
}

func findTarget(targets []AssetTarget, name, key string) AssetTarget {
	for _, t := range targets {
		if t.Target == name && t.Key == key {
			return t
		}
	}
	return AssetTarget{}
}

func editPack(t *testing.T, id, priority string) Installed {
	t.Helper()
	root := t.TempDir()
	writeProblemFile(t, root, "manifest.json", cpManifest(id))
	content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","Priority":"` + priority + `","Entries":{"123":"` + id + `"}}]}`
	if err := fsx.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return fromDisk(Installed{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
}

func touchOf(t *testing.T, target AssetTarget, id string) AssetTouch {
	t.Helper()
	for _, m := range target.Mods {
		if m.ModID == id {
			return m
		}
	}
	t.Fatalf("%s missing from %+v", id, target.Mods)
	return AssetTouch{}
}

func TestFilesWinnerFollowsSMAPILoadOrder(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	early := editPack(t, "Pack.A", "Default")
	late := editPack(t, "Pack.Z", "Default")
	// The profile lists Z first, but SMAPI loads A first by name, so Z's tied edit applies last.
	objects := findTarget(buildAssetIndex([]Installed{late, early}), "data/objects", "123")
	if objects.Winner != "Pack.Z" || !touchOf(t, objects, "Pack.A").CanWin || touchOf(t, objects, "Pack.Z").CanWin {
		t.Fatalf("tie %+v", objects)
	}

	early.Dependencies = []manifest.Dependency{{UniqueID: "Pack.Z"}}
	objects = findTarget(buildAssetIndex([]Installed{late, early}), "data/objects", "123")
	if objects.Winner != "Pack.A" || touchOf(t, objects, "Pack.Z").CanWin {
		t.Fatalf("A loads after Z through its dependency, so Z cannot be moved past it: %+v", objects)
	}

	late = editPack(t, "Pack.Z", "Early")
	objects = findTarget(buildAssetIndex([]Installed{late, editPack(t, "Pack.A", "Default")}), "data/objects", "123")
	if objects.Winner != "Pack.A" || touchOf(t, objects, "Pack.Z").CanWin {
		t.Fatalf("a lower priority cannot win by load order: %+v", objects)
	}
}

func TestFilesSharedFilterAndCounts(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	index := buildAssetIndex([]Installed{
		loadPack(t, "Pack.Low", "Characters/Abigail", "Low"),
		loadPack(t, "Pack.High", "Characters/Abigail", "High"),
		loadPack(t, "Pack.Town", "Maps/Town", ""),
	})
	page := AssetMapOf(index, "", true, 0)
	if page.All != 2 || page.Shared != 1 || page.Total != 1 || page.Targets[0].Target != "characters/abigail" {
		t.Fatalf("shared %+v", page)
	}
	if low := touchOf(t, page.Targets[0], "Pack.Low"); low.CanWin {
		t.Fatalf("a Low load cannot outrank High by order: %+v", low)
	}
	if page = AssetMapOf(index, "town", false, 0); page.All != 1 || page.Shared != 0 || page.Total != 1 {
		t.Fatalf("search %+v", page)
	}
}
