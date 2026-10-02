package problems

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadConflictWithBlankLoserIsCosmetic(t *testing.T) {
	t.Run("blank loser", func(t *testing.T) {
		winner := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"winner.json","Priority":"High"}]}`, map[string]string{
			"winner.json": `{"Tile": 1}`,
		})
		loser := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"LOSER.JSON","Priority":"Low"}]}`, map[string]string{
			"loser.JSON": `// harmless
[]`,
		})
		conflicts := assetConflicts([]Installed{winner, loser})
		if len(conflicts) != 1 || !conflicts[0].Cosmetic {
			t.Fatalf("expected a cosmetic load conflict, got %#v", conflicts)
		}
	})

	t.Run("blank winner still wipes data", func(t *testing.T) {
		winner := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"winner.json","Priority":"High"}]}`, map[string]string{
			"winner.json": `{}`,
		})
		loser := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"loser.json","Priority":"Low"}]}`, map[string]string{
			"loser.json": `{"Tile": 1}`,
		})
		conflicts := assetConflicts([]Installed{winner, loser})
		if len(conflicts) != 1 || conflicts[0].Cosmetic {
			t.Fatalf("expected a real load conflict, got %#v", conflicts)
		}
	})
}

func syntheticLoadPack(t *testing.T, content string, files map[string]string) Installed {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return Installed{Enabled: true, Folder: root, UniqueID: filepath.Base(root), Name: filepath.Base(root), Key: filepath.Base(root)}
}

func TestEditMapPatchModesUseSourceLayers(t *testing.T) {
	mapJSON := `{"width":3,"height":2,"layers":[
		{"type":"tilelayer","name":"Back","width":3,"height":2,"data":[1,0,0,0,2,0]},
		{"type":"tilelayer","name":"Front","width":3,"height":2,"data":[0,0,3,0,0,0]}
	]}`

	t.Run("overlay only writes nonempty source tiles", func(t *testing.T) {
		shapes := syntheticMapShapes(t, mapJSON, "Overlay")
		if len(shapes) != 3 {
			t.Fatalf("expected three nonempty tiles, got %#v", shapes)
		}
		if !slices.Contains(shapes, (cpShape{kind: 't', x: 10, y: 20, layer: "back"})) ||
			!slices.Contains(shapes, (cpShape{kind: 't', x: 11, y: 21, layer: "back"})) ||
			!slices.Contains(shapes, (cpShape{kind: 't', x: 12, y: 20, layer: "front"})) {
			t.Fatalf("unexpected overlay shapes: %#v", shapes)
		}
		if shapes[0].overlaps(shapes[2]) {
			t.Fatal("tiles on different layers must not overlap")
		}
	})

	t.Run("replace by layer writes each source layer", func(t *testing.T) {
		shapes := syntheticMapShapes(t, mapJSON, "ReplaceByLayer")
		if len(shapes) != 2 || shapes[0].layer == "" || shapes[1].layer == "" {
			t.Fatalf("expected one full shape per layer, got %#v", shapes)
		}
		if shapes[0].overlaps(shapes[1]) {
			t.Fatal("replace-by-layer shapes must not overlap across layers")
		}
	})

	t.Run("replace writes every layer", func(t *testing.T) {
		shapes := syntheticMapShapes(t, mapJSON, "Replace")
		if len(shapes) != 1 || shapes[0].layer != "" {
			t.Fatalf("expected one all-layer shape, got %#v", shapes)
		}
		if !shapes[0].overlaps(cpShape{kind: 't', x: 10, y: 20, layer: "back"}) {
			t.Fatal("replace shape must overlap every layer")
		}
	})

	t.Run("overlay images do not clash", func(t *testing.T) {
		clash, _ := editsClash(
			[]cpPatch{{image: true, patchMode: "Overlay", shapes: whole}},
			[]cpPatch{{image: true, patchMode: "Overlay", shapes: whole}},
		)
		if clash {
			t.Fatal("two overlay image patches should not clash")
		}
	})
}

func syntheticMapShapes(t *testing.T, mapJSON, patchMode string) []cpShape {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "map.tmj"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	content := `{"Changes":[{"Action":"EditMap","Target":"Maps/Test","FromFile":"map.tmj","FromArea":{"X":0,"Y":0,"Width":3,"Height":2},"ToArea":{"X":10,"Y":20,"Width":3,"Height":2},"PatchMode":"` + patchMode + `"}]}`
	if err := os.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	pack := cachedPack{mentions: map[string]bool{}, schema: map[string]cpSchema{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	if len(pack.patches) != 1 {
		t.Fatalf("expected one patch, got %#v", pack.patches)
	}
	return pack.patches[0].shapes
}

func TestFarmTypeMakesLoadConditionsExclusive(t *testing.T) {
	first := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"first.json","When":{"FarmType":"A_TK.FarmProjectForaging"}}]}`, map[string]string{
		"first.json": `{"Tile": 1}`,
	})
	second := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Test","FromFile":"second.json","When":{"FarmType":"WaFF"}}]}`, map[string]string{
		"second.json": `{"Tile": 2}`,
	})
	if conflicts := assetConflicts([]Installed{first, second}); len(conflicts) != 0 {
		t.Fatalf("different farm types should not conflict: %#v", conflicts)
	}
}

func TestDynamicTokenWhenMergesDefinitionConditions(t *testing.T) {
	root := t.TempDir()
	content := `{"DynamicTokens":[
		{"Name":"FarmChoice","Value":"A_TK.FarmProjectForaging","When":{"HasMod":"Author.Required","Spouse":"Abigail","FarmType":"A_TK.FarmProjectForaging"}},
		{"Name":"FarmChoice","Value":"WaFF","When":{"HasMod":"Other.Mod"}}
	],"Changes":[
		{"Action":"Load","Target":"Maps/Test","FromFile":"map.json","When":{"{{FarmChoice}}":"A_TK.FarmProjectForaging"}}
	]}`
	if err := os.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	pack := cachedPack{mentions: map[string]bool{}, schema: map[string]cpSchema{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	var load cpPatch
	for _, patch := range pack.patches {
		if patch.kind == "load" {
			load = patch
			break
		}
	}
	if load.when.spouse != "abigail" || !slices.Contains(load.when.places["farmtype"], "a_tk.farmprojectforaging") {
		t.Fatalf("dynamic token conditions were not merged: %#v", load)
	}
	if !slices.Contains(load.when.anyOf[0], "author.required") {
		t.Fatalf("dynamic token HasMod condition was not merged: %#v", load.when)
	}
	if !exclusive(load, cpPatch{places: map[string][]string{"farmtype": {"waff"}}}) {
		t.Fatal("different dynamic FarmType values should be exclusive")
	}
}

func TestPriorityParsingKeepsOddChangesIsolated(t *testing.T) {
	root := t.TempDir()
	content := `{"Changes":[
		{"Action":"Load","Target":"Maps/Load","FromFile":"load.json","Priority":42},
		{"Action":"EditImage","Target":"Maps/Image","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"Priority":"Late - 10"},
		{"Action":"Load","Target":"Maps/Odd","FromFile":"odd.json","Priority":{"unexpected":true}}
	]}`
	if err := os.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	pack := cachedPack{mentions: map[string]bool{}, schema: map[string]cpSchema{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	if len(pack.patches) != 3 {
		t.Fatalf("one malformed priority must not discard the other changes: %#v", pack.patches)
	}
	if pack.patches[0].priority != "42" || pack.patches[1].priority != "Late - 10" || pack.patches[2].priority != "" {
		t.Fatalf("unexpected parsed priorities: %#v", pack.patches)
	}
	if contentPatcherPriority("load", "Low") != -1000 ||
		contentPatcherPriority("edit", "Default") != 0 ||
		contentPatcherPriority("load", "High + 25") != 1025 ||
		contentPatcherPriority("edit", "Early - 5") != -1005 {
		t.Fatal("Content Patcher priority scale was not applied")
	}
}

func TestConflictWinnerUsesClashingPatchPriority(t *testing.T) {
	first := syntheticEditPack(t, `{"Changes":[
		{"Action":"EditImage","Target":"Maps/Test","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"Priority":"High"},
		{"Action":"EditImage","Target":"Maps/Test","ToArea":{"X":1,"Y":1,"Width":1,"Height":1},"Priority":"Low"}
	}`)
	second := syntheticEditPack(t, `{"Changes":[
		{"Action":"EditImage","Target":"Maps/Test","ToArea":{"X":1,"Y":1,"Width":1,"Height":1},"Priority":"Medium"}
	}`)
	conflicts := assetConflicts([]Installed{first, second})
	if len(conflicts) != 1 || conflicts[0].WinnerID != second.UniqueID {
		t.Fatalf("winner must be selected from clashing patches: %#v", conflicts)
	}

	base := packHit{
		id: "base", name: "Base", loads: []cpPatch{{priority: "Medium"}},
		loadClashes: map[int]bool{0: true}, dependencies: map[string]bool{},
	}
	addon := packHit{
		id: "addon", name: "Addon", loads: []cpPatch{{priority: "Medium"}},
		loadClashes: map[int]bool{0: true}, dependencies: map[string]bool{"base": true},
	}
	loadOrder := conflictOf("load", "Maps/Test", []packHit{base, addon})
	if loadOrder.WinnerID != "addon" || loadOrder.WinnerName != "by load order" {
		t.Fatalf("dependency order should decide equal-priority loads: %#v", loadOrder)
	}
	unclear := conflictOf("load", "Maps/Test", []packHit{base, packHit{
		id: "other", name: "Other", loads: []cpPatch{{priority: "Medium"}},
		loadClashes: map[int]bool{0: true}, dependencies: map[string]bool{},
	}})
	if unclear.WinnerName != "unclear" {
		t.Fatalf("unrelated equal-priority loads should be unclear: %#v", unclear)
	}
	exclusive := conflictOf("load", "Maps/Test", []packHit{
		{id: "one", name: "One", loads: []cpPatch{{priority: "Exclusive"}}, loadClashes: map[int]bool{0: true}},
		{id: "two", name: "Two", loads: []cpPatch{{priority: "Exclusive"}}, loadClashes: map[int]bool{0: true}},
	})
	if exclusive.WinnerName != "CP applies neither" {
		t.Fatalf("exclusive loads should leave the asset unchanged: %#v", exclusive)
	}
}

func syntheticEditPack(t *testing.T, content string) Installed {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return Installed{Enabled: true, Folder: root, UniqueID: filepath.Base(root), Name: filepath.Base(root), Key: filepath.Base(root)}
}

func TestSwitchOffOnlySuggestsNonClashingAllowedValue(t *testing.T) {
	first := syntheticEditPack(t, `{"ConfigSchema":{"Variant":{"Default":"Red","AllowValues":"Red, Blue, Green"}},"Changes":[
		{"Action":"EditImage","Target":"Maps/Test","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"Variant":"Red"}},
		{"Action":"EditImage","Target":"Maps/Test","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"Variant":"Blue"}}
	]}`)
	second := syntheticEditPack(t, `{"Changes":[
		{"Action":"EditImage","Target":"Maps/Test","ToArea":{"X":0,"Y":0,"Width":1,"Height":1}}
	]}`)
	conflicts := assetConflicts([]Installed{first, second})
	if len(conflicts) != 1 {
		t.Fatalf("expected one conflict, got %#v", conflicts)
	}
	for _, fix := range conflicts[0].Fixes {
		if sameID(fix.UniqueID, first.UniqueID) {
			if fix.Field != "Variant" || fix.Value != "Green" {
				t.Fatalf("expected the only safe allowed value, got %#v", fix)
			}
			return
		}
	}
	t.Fatalf("expected a safe setting fix for %s: %#v", first.UniqueID, conflicts[0].Fixes)
}
