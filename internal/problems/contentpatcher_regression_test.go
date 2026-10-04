package problems

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"UniqueID":"Test.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`), 0o600); err != nil {
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
	]}`)
	second := syntheticEditPack(t, `{"Changes":[
		{"Action":"EditImage","Target":"Maps/Test","ToArea":{"X":1,"Y":1,"Width":1,"Height":1},"Priority":"Medium"}
	]}`)
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
	unclear := conflictOf("load", "Maps/Test", []packHit{base, {
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
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"UniqueID":"Test.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`), 0o600); err != nil {
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
		{"Action":"EditImage","Target":"Maps/Test","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"Variant":"Blue"}}`+
		strings.Repeat(`,{"Action":"EditImage","Target":"Maps/Elsewhere","ToArea":{"X":0,"Y":0,"Width":1,"Height":1}}`, 9)+`
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

func TestIncludedBlankLoadsUsePackRootPath(t *testing.T) {
	root := t.TempDir()
	writeRegressionFile(t, root, "content.json", `{"Changes":[{"Action":"Include","FromFile":"nested/content.json"}]}`)
	writeRegressionFile(t, root, "nested/content.json", `{"Changes":[{"Action":"Load","Target":"Data/Test","FromFile":"blank.json","Priority":"low"}]}`)
	writeRegressionFile(t, root, "blank.json", "{\r\n// empty\r\n}")
	writeRegressionFile(t, root, "manifest.json", `{"UniqueID":"Test.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	mod := Installed{Key: "included", Enabled: true, Folder: root, UniqueID: "Included.Blank", Name: "Included Blank"}

	conflicts := assetConflicts([]Installed{mod, syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Data/Test","FromFile":"other.json"}]}`, map[string]string{
		"other.json": `{"value":1}`,
	})})
	if len(conflicts) != 1 || !conflicts[0].Cosmetic {
		t.Fatalf("included blank load should be cosmetic: %#v", conflicts)
	}
}

func TestEquivalentLoadsAreNotConflicts(t *testing.T) {
	first := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Data/Test","FromFile":"a.json"}]}`, map[string]string{
		"a.json": `{}`,
	})
	second := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Data/Test","FromFile":"b.json"}]}`, map[string]string{
		"b.json": "{\n}",
	})
	if conflicts := assetConflicts([]Installed{first, second}); len(conflicts) != 0 {
		t.Fatalf("equivalent loads should not conflict: %#v", conflicts)
	}
}

func TestIdenticalImageEditsAreNotConflicts(t *testing.T) {
	first := syntheticImagePack(t, "first.png", []byte("same image"))
	second := syntheticImagePack(t, "second.png", []byte("same image"))
	if conflicts := assetConflicts([]Installed{first, second}); len(conflicts) != 0 {
		t.Fatalf("identical image edits should not conflict: %#v", conflicts)
	}
}

func TestDeadLowPriorityLoadOffersDefaultSetting(t *testing.T) {
	loser := settingPack(t, `{"FarmCaveChange":{"Default":false,"AllowValues":"false, true"}}`,
		`[{"Action":"Load","Target":"Maps/FarmCave","FromFile":"loser.json","Priority":"Low","When":{"FarmCaveChange":true}}]`,
		`{"FarmCaveChange":true}`)
	writeRegressionFile(t, loser.Folder, "loser.json", `{"Tile":1}`)
	winner := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/FarmCave","FromFile":"winner.json","Priority":"High"}]}`, map[string]string{
		"winner.json": `{"Tile":2}`,
	})
	conflicts, settings := assetConflictResults([]Installed{loser, winner})
	if len(conflicts) != 0 || len(settings) != 1 {
		t.Fatalf("expected dead-setting hint instead of cosmetic conflict, got %#v %#v", conflicts, settings)
	}
	if settings[0].Field != "FarmCaveChange" || settings[0].Current != "true" ||
		len(settings[0].Suggested) != 1 || settings[0].Suggested[0] != "false" {
		t.Fatalf("unexpected dead-setting hint: %#v", settings[0])
	}
}

func syntheticImagePack(t *testing.T, file string, source []byte) Installed {
	t.Helper()
	root := t.TempDir()
	writeRegressionFile(t, root, "manifest.json", `{"UniqueID":"Test.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	writeRegressionFile(t, root, "content.json", `{"Changes":[{"Action":"EditImage","Target":"LooseSprites/Cursors","FromFile":"`+file+`","ToArea":{"X":2,"Y":3,"Width":18,"Height":20}}]}`)
	writeRegressionFile(t, root, file, string(source))
	return Installed{Enabled: true, Folder: root, UniqueID: filepath.Base(root), Name: filepath.Base(root), Key: filepath.Base(root)}
}

func TestBareDynamicTokenWhenMergesSpouseCondition(t *testing.T) {
	root := t.TempDir()
	writeRegressionFile(t, root, "content.json", `{"DynamicTokens":[
		{"Name":"ShadowKidsActive","Value":false},
		{"Name":"ShadowKidsActive","Value":true,"When":{"Spouse":"SenS"}}
	],"Changes":[{"Action":"EditImage","Target":"characters/toddler","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"ShadowKidsActive":true}}]}`)
	pack := cachedPack{mentions: map[string]bool{}, schema: map[string]cpSchema{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	if len(pack.patches) != 3 || pack.patches[2].spouse != "sens" {
		t.Fatalf("bare DynamicToken spouse condition was not merged: %#v", pack.patches)
	}
}

func TestOverlayMapWithUnknownLayerDoesNotClash(t *testing.T) {
	root := t.TempDir()
	writeRegressionFile(t, root, "content.json", `{"Changes":[
		{"Action":"EditMap","Target":"Maps/Test","FromFile":"fog.tmx","PatchMode":"Overlay"},
		{"Action":"EditMap","Target":"Maps/Test","ToArea":{"X":0,"Y":0,"Width":2,"Height":2}}
	]}`)
	writeRegressionFile(t, root, "fog.tmx", `<?xml version="1.0"?><map width="2" height="2"><layer name="AlwaysFront4" width="2" height="2"><data encoding="csv">1,0,0,0</data></layer></map>`)
	pack := cachedPack{mentions: map[string]bool{}, schema: map[string]cpSchema{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	if len(pack.patches) != 2 {
		t.Fatalf("expected two map patches: %#v", pack.patches)
	}
	if clash, _ := editsClash([]cpPatch{pack.patches[0]}, []cpPatch{pack.patches[1]}); clash {
		t.Fatal("overlay layer should not clash with an unknown-layer patch")
	}
}

func TestOverlayImageUsesOpaqueCells(t *testing.T) {
	opaque := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	opaque.SetNRGBA(0, 0, color.NRGBA{A: 255})
	root := t.TempDir()
	writeRegressionFile(t, root, "manifest.json", `{"UniqueID":"Test.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	writeRegressionFile(t, root, "content.json", `{"Changes":[{"Action":"EditImage","Target":"Maps/Test","FromFile":"patch.png","ToArea":{"X":0,"Y":0,"Width":32,"Height":16},"PatchMode":"Overlay"}]}`)
	writePNG(t, filepath.Join(root, "patch.png"), opaque)
	pack := readContentPack(Installed{Enabled: true, Folder: root})
	other := cpPatch{image: true, shapes: []cpShape{{kind: 'r', x: 16, y: 0, w: 16, h: 16}}}
	if clash, _ := editsClash(pack.patches, []cpPatch{other}); clash {
		t.Fatal("transparent overlay cells must not clash")
	}
	opaque.SetNRGBA(16, 0, color.NRGBA{A: 255})
	writePNG(t, filepath.Join(root, "patch.png"), opaque)
	packCache.Delete(filepath.Clean(root))
	pack = readContentPack(Installed{Enabled: true, Folder: root})
	if clash, _ := editsClash(pack.patches, []cpPatch{other}); !clash {
		t.Fatal("opaque overlay cell must clash")
	}
}

func TestTokenizedImageFromFileExpandsCaseInsensitive(t *testing.T) {
	root := t.TempDir()
	writeRegressionFile(t, root, "manifest.json", `{"UniqueID":"Test.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	writeRegressionFile(t, root, "content.json", `{"Changes":[{"Action":"EditImage","Target":"Maps/Test","FromFile":"sprites/{{season}}.png","ToArea":{"X":32,"Y":0,"Width":16,"Height":16},"PatchMode":"Overlay"}]}`)
	for _, season := range []string{"Spring", "Summer", "Fall", "Winter"} {
		img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
		if season == "Fall" {
			img.SetNRGBA(0, 0, color.NRGBA{A: 255})
		}
		writePNG(t, filepath.Join(root, "sprites", season+".png"), img)
	}
	pack := cachedPack{mentions: map[string]bool{}, schema: map[string]cpSchema{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	if len(sourceFiles(root, "sprites/{{season}}.png")) != 4 {
		t.Fatal("season token did not expand to all case-insensitive matches")
	}
	other := cpPatch{image: true, shapes: []cpShape{{kind: 'r', x: 32, y: 0, w: 16, h: 16}}}
	if clash, _ := editsClash(pack.patches, []cpPatch{other}); !clash {
		t.Fatal("opaque seasonal file must clash")
	}
}

func TestUnresolvableTokenizedImageFallsBackToWholeSheet(t *testing.T) {
	root := t.TempDir()
	writeRegressionFile(t, root, "manifest.json", `{"UniqueID":"Test.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`)
	writeRegressionFile(t, root, "content.json", `{"Changes":[{"Action":"EditImage","Target":"Maps/Test","FromFile":"sprites/{{missing}}.png","PatchMode":"Overlay"}]}`)
	pack := cachedPack{mentions: map[string]bool{}, schema: map[string]cpSchema{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	if len(pack.patches) != 1 || len(pack.patches[0].shapes) != 1 || pack.patches[0].shapes[0].kind != 'w' {
		t.Fatalf("unresolvable token should use whole-sheet fallback: %#v", pack.patches)
	}
}

func TestDifferentSpouseConditionsExcludeEdits(t *testing.T) {
	root := t.TempDir()
	writeRegressionFile(t, root, "content.json", `{"Changes":[
		{"Action":"EditImage","Target":"characters/toddler","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"Relationship:Sigurd":"Married"}},
		{"Action":"EditImage","Target":"characters/toddler","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"Spouse":"SenS"}}
	]}`)
	pack := cachedPack{mentions: map[string]bool{}, schema: map[string]cpSchema{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	if len(pack.patches) != 2 || !exclusive(pack.patches[0], pack.patches[1]) {
		t.Fatalf("different spouse conditions should be exclusive: %#v", pack.patches)
	}
}

func TestDynamicTokenReachabilityGatesConflicts(t *testing.T) {
	t.Run("last installed-mod definition masks earlier values", func(t *testing.T) {
		pack := syntheticEditPack(t, `{"DynamicTokens":[
			{"Name":"Concession","Value":"Default"},
			{"Name":"Concession","Value":"CC","When":{"HasMod |contains=FlashShifter.StardewValleyExpandedCP":false}},
			{"Name":"Concession","Value":"Default","When":{"HasMod |contains=FlashShifter.StardewValleyExpandedCP":true}}
		],"Changes":[
			{"Action":"EditImage","Target":"maps/movietheater_tilesheet","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"Concession |contains=Default":false}}
		]}`)
		peer := syntheticEditPack(t, `{"Changes":[{"Action":"EditImage","Target":"maps/movietheater_tilesheet","ToArea":{"X":0,"Y":0,"Width":1,"Height":1}}]}`)
		sve := Installed{Enabled: true, UniqueID: "FlashShifter.StardewValleyExpandedCP", Name: "SVE", Key: "sve"}
		if conflicts := assetConflicts([]Installed{pack, peer, sve}); len(conflicts) != 0 {
			t.Fatalf("masked dynamic token condition should remove the edit: %#v", conflicts)
		}
	})

	t.Run("unknown game state remains possible", func(t *testing.T) {
		pack := syntheticEditPack(t, `{"DynamicTokens":[
			{"Name":"SeasonalChoice","Value":"Spring","When":{"Season":"Spring"}}
		],"Changes":[
			{"Action":"EditImage","Target":"maps/movietheater_tilesheet","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"SeasonalChoice":"Spring"}}
		]}`)
		peer := syntheticEditPack(t, `{"Changes":[{"Action":"EditImage","Target":"maps/movietheater_tilesheet","ToArea":{"X":0,"Y":0,"Width":1,"Height":1}}]}`)
		if conflicts := assetConflicts([]Installed{pack, peer}); len(conflicts) != 1 {
			t.Fatalf("unknown game-state definition should remain possible: %#v", conflicts)
		}
	})

	t.Run("opposite HasFlag assumptions are unreachable", func(t *testing.T) {
		pack := syntheticEditPack(t, `{"DynamicTokens":[
			{"Name":"FlagChoice","Value":"Default"},
			{"Name":"FlagChoice","Value":"Flagged","When":{"HasFlag":"festival"}}
		],"Changes":[
			{"Action":"EditImage","Target":"maps/movietheater_tilesheet","ToArea":{"X":0,"Y":0,"Width":1,"Height":1},"When":{"FlagChoice":"Flagged","HasFlag |contains=festival":false}}
		]}`)
		peer := syntheticEditPack(t, `{"Changes":[{"Action":"EditImage","Target":"maps/movietheater_tilesheet","ToArea":{"X":0,"Y":0,"Width":1,"Height":1}}]}`)
		if conflicts := assetConflicts([]Installed{pack, peer}); len(conflicts) != 0 {
			t.Fatalf("opposite HasFlag definition should not apply: %#v", conflicts)
		}
	})
}

func writeRegressionFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writePNG(t *testing.T, path string, image image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err := png.Encode(&data, image); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadsWithoutPriorityAreExclusive(t *testing.T) {
	cave := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/FarmCave","FromFile":"FarmCave_New.tmx"}]}`, map[string]string{
		"FarmCave_New.tmx": `<map width="11" height="12"/>`,
	})
	farm := syntheticLoadPack(t, `{
		"ConfigSchema": {"EnableFarmCave": {"AllowValues": "true, false", "Default": "true"}},
		"Changes": [{"Action": "Load", "Target": "Maps/FarmCave", "FromFile": "FarmCave.tbin", "When": {"EnableFarmCave": "true"}}]
	}`, map[string]string{"FarmCave.tbin": "tBIN10 farm cave"})
	mods := []Installed{cave, farm}
	conflicts := assetConflicts(mods)
	if len(conflicts) != 1 || conflicts[0].WinnerName != "CP applies neither" {
		t.Fatalf("two loads without a priority are both Exclusive, got %#v", conflicts)
	}
	if target := findTarget(AssetMapOf(mods, "", 0).Targets, "maps/farmcave", ""); target.Winner != "" {
		t.Fatalf("asset map names a winner Content Patcher never applies: %+v", target)
	}
}

func TestLowLoadThatNamesTheOtherPackIsAFallback(t *testing.T) {
	aquarium := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Portraits/Curator","FromFile":"curator.png"}]}`, map[string]string{
		"curator.png": "aquarium curator",
	})
	aquarium.UniqueID = "Gervig91.StardewAquariumCP"
	portraits := func(compat string) Installed {
		return syntheticLoadPack(t, `{"Changes":[
			{"Action": "Load", "Target": "Portraits/Curator", "FromFile": "Curator.png", "Priority": "Low"}`+compat+`
		]}`, map[string]string{
			"Curator.png":   "extras curator",
			"Aquarium.json": `{"Changes":[{"Action":"EditData","Target":"Data/Shops","TargetField":["Gervig91.StardewAquariumCP_BeachShop","Owners"],"Entries":{"Default":{"Name":"AnyOrNone"}}}]}`,
		})
	}
	aware := portraits(`,{"Action": "Include", "FromFile": "Aquarium.json", "When": {"HasMod": "Gervig91.StardewAquariumCP"}}`)
	if conflicts := assetConflicts([]Installed{aware, aquarium}); len(conflicts) != 0 {
		t.Fatalf("a Low load from a pack with an Aquarium compatibility patch is its fallback, got %#v", conflicts)
	}
	unaware := portraits("")
	if conflicts := assetConflicts([]Installed{unaware, aquarium}); len(conflicts) != 1 {
		t.Fatalf("a Low load from a pack that never names the other still conflicts, got %#v", conflicts)
	}
}

func TestConfigTokenInConditionIsItsConfiguredValue(t *testing.T) {
	blackberry := func(config string) Installed {
		files := map[string]string{"Farm_Greenhouse_Dirt.tbin": "tBIN10 blackberry"}
		if config != "" {
			files["config.json"] = config
		}
		return syntheticLoadPack(t, `{
			"ConfigSchema": {"FarmToReplace": {"AllowValues": "Standard, Forest, Riverland, Hilltop, Wilderness, FourCorners, Beach, Meadowlands", "Default": "Standard"}},
			"Changes": [{"Action": "Load", "Target": "Maps/Farm_Greenhouse_Dirt", "FromFile": "Farm_Greenhouse_Dirt.tbin", "When": {"FarmType": "{{FarmToReplace}}"}}]
		}`, files)
	}
	overgrown := syntheticLoadPack(t, `{"Changes":[{"Action":"Load","Target":"Maps/Farm_Greenhouse_Dirt","FromFile":"Farm_Greenhouse_Dirt.tbin","When":{"FarmType":"Forest"}}]}`, map[string]string{
		"Farm_Greenhouse_Dirt.tbin": "tBIN10 overgrown",
	})
	if conflicts := assetConflicts([]Installed{blackberry(""), overgrown}); len(conflicts) != 0 {
		t.Fatalf("a Standard farm replacement and a Forest one never load together, got %#v", conflicts)
	}
	if conflicts := assetConflicts([]Installed{blackberry(`{"FarmToReplace": "Forest"}`), overgrown}); len(conflicts) != 1 {
		t.Fatalf("both replace the Forest farm, got %#v", conflicts)
	}
}

func TestConfigTokenDataValueMatchesLiteral(t *testing.T) {
	betterThings := syntheticLoadPack(t, `{
		"ConfigSchema": {"MachinesCopyQuality": {"AllowValues": "true, false", "Default": "true"}},
		"Changes": [{
			"Action": "EditData", "Target": "Data/Machines",
			"TargetField": [ "(BC)12", "OutputRules", "Default_Honey", "OutputItem", "(O)459" ],
			"Entries": { "CopyQuality": true },
			"When": { "MachinesCopyQuality": true }
		}]
	}`, nil)
	mead := func(retains string) Installed {
		return syntheticLoadPack(t, `{
			"ConfigSchema": {"MeadRetainsQuality": {"AllowValues": "true, false", "Default": "true"}},
			"Changes": [{
				"Action": "EditData", "Target": "Data/Machines",
				"TargetField": [ "(BC)12", "OutputRules", "Default_Honey", "OutputItem", "(O)459" ],
				"Entries": { "CopyQuality": "{{MeadRetainsQuality}}" }
			}]
		}`, map[string]string{"config.json": `{"EnableColoredSprites": "true", "MeadRetainsQuality": "` + retains + `"}`})
	}
	if conflicts := assetConflicts([]Installed{betterThings, mead("true")}); len(conflicts) != 0 {
		t.Fatalf("both set CopyQuality to true, got %#v", conflicts)
	}
	if conflicts := assetConflicts([]Installed{betterThings, mead("false")}); len(conflicts) != 1 {
		t.Fatalf("CopyQuality true against false must conflict, got %#v", conflicts)
	}
}

func TestEngagedIsOnePartnerAtATime(t *testing.T) {
	socialPage := func(label, when string) Installed {
		return syntheticLoadPack(t, `{"Changes":[{
			"Action": "EditData", "Target": "Strings/StringsFromCSFiles",
			"Entries": {"SocialPage_Relationship_Husband": "`+label+`"},
			"When": {`+when+`}
		}]}`, nil)
	}
	agatha := socialPage("(wife)", `"Relationship:Agatha": "Married", "Language": "en"`)
	for _, when := range []string{`"Relationship:Lunna": "Engaged"`, `"Relationship:Lunna": "Engaged, Married"`} {
		if conflicts := assetConflicts([]Installed{agatha, socialPage("(fiancée)", when)}); len(conflicts) != 0 {
			t.Fatalf("%s needs a different partner than Agatha, got %#v", when, conflicts)
		}
	}
	if conflicts := assetConflicts([]Installed{agatha, socialPage("(fiancée)", `"Relationship:Lunna": "Dating, Engaged"`)}); len(conflicts) != 1 {
		t.Fatalf("dating Lunna can coincide with marriage to Agatha, got %#v", conflicts)
	}
}

func TestSwitchOffOffersTheNarrowestField(t *testing.T) {
	npc := `{"AllowValues": "enabled, disabled", "Default": "enabled"}`
	breakfast := syntheticLoadPack(t, `{
		"ConfigSchema": {
			"ImmersionMode": `+npc+`, "Gus": `+npc+`, "Lewis": `+npc+`, "Marnie": `+npc+`,
			"PlotSchedules": {"AllowValues": "enabled, plot, disabled", "Default": "enabled"}
		},
		"DynamicTokens": [
			{"Name": "ImmersionGus", "Value": "true"},
			{"Name": "ImmersionGus", "Value": "false", "When": {"ImmersionMode": "disabled", "Gus": "disabled"}}
		],
		"Changes": [
			{"Action": "Include", "FromFile": "mail.json"},
			{"Action": "Include", "FromFile": "schedules.json", "When": {"PlotSchedules": "enabled"}},
			{"Action": "Include", "FromFile": "plotschedules.json", "When": {"PlotSchedules": "plot"}}
		]
	}`, map[string]string{
		"mail.json": `{"Changes": [{"Action": "EditData", "Target": "Data/Mail", "Entries": {"BreakfastMenu": "Gus is serving breakfast."}}]}`,
		"schedules.json": `{"Changes": [
			{"Action": "EditData", "Target": "Characters/schedules/Gus", "Entries": {"Tue": "700 Saloon 10 18 2"}, "When": {"ImmersionGus": "true"}},
			{"Action": "EditData", "Target": "Characters/schedules/Lewis", "Entries": {"Tue": "800 Saloon 15 17 2"}, "When": {"Lewis": "enabled"}},
			{"Action": "EditData", "Target": "Characters/schedules/Marnie", "Entries": {"Mon": "830 Saloon 16 17 2"}, "When": {"Marnie": "enabled"}}
		]}`,
		"plotschedules.json": `{"Changes": [
			{"Action": "EditData", "Target": "Characters/schedules/Gus", "Entries": {"Tue": "700 Saloon 10 18 2/1900 Saloon 10 18 2"}, "When": {"ImmersionGus": "true"}}
		]}`,
	})
	breakfast.Name = "Part of a Saloon Breakfast"
	mayor := syntheticLoadPack(t, `{"Changes": [
		{"Action": "EditData", "Target": "Characters/schedules/Gus", "Entries": {"Tue": "900 Town 30 60 2"}},
		{"Action": "EditData", "Target": "Characters/schedules/Lewis", "Entries": {"Tue": "900 ManorHouse 4 5 2"}},
		{"Action": "EditData", "Target": "Characters/schedules/Marnie", "Entries": {"Mon": "900 AnimalShop 12 14 2"}}
	]}`, nil)
	want := map[string]string{"characters/schedules/lewis": "Lewis", "characters/schedules/marnie": "Marnie", "characters/schedules/gus": ""}
	for range 20 {
		conflicts := assetConflicts([]Installed{breakfast, mayor})
		if len(conflicts) != len(want) {
			t.Fatalf("conflicts %#v", conflicts)
		}
		for _, c := range conflicts {
			got := ""
			for _, fix := range c.Fixes {
				if fix.UniqueID == breakfast.UniqueID {
					got = fix.Field + "=" + fix.Value
				}
			}
			if field := want[c.Target]; (field == "" && got != "") || (field != "" && got != field+"=disabled") {
				t.Fatalf("%s: offered %q, want the %q field and never PlotSchedules", c.Target, got, field)
			}
		}
	}

	mail := strings.Repeat(`{"Action": "EditData", "Target": "Data/Mail", "Entries": {"DwarfLetter": "Hello."}},`, 12)
	dwarven := syntheticLoadPack(t, `{
		"ConfigSchema": {
			"EnableDwarfMagic": {"AllowValues": "true, false", "Default": "true"},
			"EnableBaseTransmutationRecipes": {"AllowValues": "true, false", "Default": "true"}
		},
		"Changes": [`+mail+`
			{"Action": "EditData", "Target": "Data/CraftingRecipes", "Entries": {"Copper Bar": "334 2/Field/378/false/null/"}, "When": {"EnableDwarfMagic": true, "EnableBaseTransmutationRecipes": true}},
			{"Action": "EditData", "Target": "Data/Objects", "Entries": {"Dwarven Rune": {"Name": "Dwarven Rune"}}, "When": {"EnableDwarfMagic": true}},
			{"Action": "EditData", "Target": "Data/Shops", "Entries": {"DwarfMagic": {"Owners": []}}, "When": {"EnableDwarfMagic": true}}
		]
	}`, nil)
	alchemistry := syntheticLoadPack(t, `{"Changes": [{"Action": "EditData", "Target": "Data/CraftingRecipes", "Entries": {"Copper Bar": "378 5/Field/334/false/null/"}}]}`, nil)
	conflicts := assetConflicts([]Installed{dwarven, alchemistry})
	if len(conflicts) != 1 || !slices.ContainsFunc(conflicts[0].Fixes, func(f ConflictFix) bool {
		return f.UniqueID == dwarven.UniqueID && f.Field == "EnableBaseTransmutationRecipes"
	}) {
		t.Fatalf("expected the recipe switch rather than all of Dwarf Magic: %#v", conflicts)
	}
}
