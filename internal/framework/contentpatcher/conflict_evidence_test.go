package contentpatcher

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestConflictEvidenceEditImageOverlap(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	a := imageConflictPack(t, "Pack.ImageA", 0, 0, 16, 16)
	b := imageConflictPack(t, "Pack.ImageB", 8, 8, 16, 16)
	got := check([]framework.Mod{a, b})
	if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Kind != "edit" || got.AssetConflicts[0].Target != "tilesheets/crops" {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
	ev := got.AssetConflicts[0].Evidence
	if len(ev) != 2 {
		t.Fatalf("evidence = %+v", ev)
	}
	byID := map[string]framework.ConflictEvidence{}
	for _, e := range ev {
		byID[e.PackID.Local()] = e
		if e.Action != kindEditImage || e.Source != "content.json" || e.Index != 0 || e.Target != "tilesheets/crops" {
			t.Fatalf("evidence %+v", e)
		}
	}
	left, right := byID["Pack.ImageA"], byID["Pack.ImageB"]
	if left.PackName != "Pack.ImageA" || !strings.Contains(left.ToArea, `"X":0`) || left.Priority != "Late" {
		t.Fatalf("left %+v", left)
	}
	if left.CropW != 8 || left.CropH != 8 || left.CropX != 8 || left.CropY != 8 {
		t.Fatalf("left crop %+v", left)
	}
	if right.CropW != 8 || right.CropH != 8 || right.CropX != 0 || right.CropY != 0 {
		t.Fatalf("right crop %+v", right)
	}
}

func TestConflictEvidenceEditData(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	a := dataConflictPack(t, "Pack.DataA", "alpha")
	b := dataConflictPack(t, "Pack.DataB", "beta")
	got := check([]framework.Mod{a, b})
	if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Kind != "edit" || got.AssetConflicts[0].Target != "data/objects" {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
	ev := got.AssetConflicts[0].Evidence
	if len(ev) != 2 {
		t.Fatalf("evidence = %+v", ev)
	}
	for _, e := range ev {
		if len(e.Keys) != 1 || e.Keys[0] != "Entry 123" {
			t.Fatalf("keys %q", e.Keys)
		}
		if e.Action != kindEditData || e.Source != "content.json" || e.Index != 0 || e.When == "" || e.Priority != "Late" {
			t.Fatalf("evidence %+v", e)
		}
		if !strings.Contains(e.When, "season=spring") {
			t.Fatalf("when %q", e.When)
		}
	}
}

func TestCropImageClampsToBounds(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 10), G: uint8(y * 10), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()

	url, err := cropImageDataURL(raw, -4, -4, 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	got := decodeDataURLPNG(t, url)
	if got.Bounds().Dx() != 8 || got.Bounds().Dy() != 8 {
		t.Fatalf("clamped size %v", got.Bounds())
	}

	url, err = cropImageDataURL(raw, 6, 6, 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	got = decodeDataURLPNG(t, url)
	if got.Bounds().Dx() != 2 || got.Bounds().Dy() != 2 {
		t.Fatalf("edge size %v", got.Bounds())
	}
	r, g, _, a := got.At(got.Bounds().Min.X, got.Bounds().Min.Y).RGBA()
	if r>>8 != 60 || g>>8 != 60 || a>>8 != 255 {
		t.Fatalf("edge pixel r=%d g=%d a=%d", r>>8, g>>8, a>>8)
	}
}

func imageConflictPack(t *testing.T, id string, x, y, w, h int) framework.Mod {
	t.Helper()
	root := t.TempDir()
	testfs.WriteFile(t, root, "manifest.json", packs.Manifest(id))
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	img.SetNRGBA(0, 0, color.NRGBA{A: 255, R: id[len(id)-1]})
	writePNG(t, filepath.Join(root, "patch.png"), img)
	content := `{"Changes":[{"Action":"EditImage","Target":"TileSheets/crops","FromFile":"patch.png","Priority":"Late","ToArea":{"X":` +
		strconv.Itoa(x) + `,"Y":` + strconv.Itoa(y) + `,"Width":` + strconv.Itoa(w) + `,"Height":` + strconv.Itoa(h) + `}}]}`
	testfs.WriteFile(t, root, "content.json", content)
	return packs.FromDisk(framework.Mod{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
}

func dataConflictPack(t *testing.T, id, value string) framework.Mod {
	t.Helper()
	content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","Priority":"Late","When":{"Season":"Spring"},"Entries":{"123":"` + value + `"}}]}`
	return packs.Disk(t, id, map[string]string{"manifest.json": packs.Manifest(id), "content.json": content})
}

func decodeDataURLPNG(t *testing.T, url string) image.Image {
	t.Helper()
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(url, prefix) {
		t.Fatalf("url %q", url)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(url, prefix))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestTokenDataKeysDoNotClashAcrossPacks(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(id, value string) framework.Mod {
		root := t.TempDir()
		testfs.WriteFile(t, root, "manifest.json", packs.Manifest(id))
		content := `{"Changes":[{"Action":"EditData","Target":"Data/TriggerActions","Entries":{"{{ModId}}_MigrateIds":"` + value + `"}}]}`
		testfs.WriteFile(t, root, "content.json", content)
		return packs.FromDisk(framework.Mod{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
	}
	got := check([]framework.Mod{pack("Mizu.Quail", "a"), pack("Mizu.Turkey", "b")})
	if len(got.AssetConflicts) != 0 {
		t.Fatalf("{{ModId}} keys are per pack, got %+v", got.AssetConflicts)
	}
}

func TestTargetFieldScopesDataKeys(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(id, item string) framework.Mod {
		root := t.TempDir()
		testfs.WriteFile(t, root, "manifest.json", packs.Manifest(id))
		content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","TargetField":["` + item + `"],"Entries":{"Price":"` + id + `"}}]}`
		testfs.WriteFile(t, root, "content.json", content)
		return packs.FromDisk(framework.Mod{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
	}
	got := check([]framework.Mod{pack("A.One", "301"), pack("B.Two", "302")})
	if len(got.AssetConflicts) != 0 {
		t.Fatalf("Price on different items is no conflict, got %+v", got.AssetConflicts)
	}
	same := check([]framework.Mod{pack("C.One", "301"), pack("D.Two", "301")})
	if len(same.AssetConflicts) != 1 {
		t.Fatalf("Price on the same item still clashes, got %+v", same.AssetConflicts)
	}
}

func TestListAppendsDoNotClash(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(id string) framework.Mod {
		root := t.TempDir()
		testfs.WriteFile(t, root, "manifest.json", packs.Manifest(id))
		content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","TargetField":["16","ContextTags"],"Entries":{"#-1":"` + id + `_tag"}}]}`
		testfs.WriteFile(t, root, "content.json", content)
		return packs.FromDisk(framework.Mod{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
	}
	got := check([]framework.Mod{pack("A.Tags"), pack("B.Tags")})
	if len(got.AssetConflicts) != 0 {
		t.Fatalf("list appends never clash, got %+v", got.AssetConflicts)
	}
}

func TestTextOverwritesAreShownNotCounted(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(id, target string) framework.Mod {
		root := t.TempDir()
		testfs.WriteFile(t, root, "manifest.json", packs.Manifest(id))
		content := `{"Changes":[{"Action":"EditData","Target":"` + target + `","Entries":{"Mon2":"` + id + `"}}]}`
		testfs.WriteFile(t, root, "content.json", content)
		return packs.FromDisk(framework.Mod{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
	}
	got := check([]framework.Mod{pack("A.Lines", "Characters/Dialogue/Marnie"), pack("B.Lines", "Characters/Dialogue/Marnie")})
	if len(got.AssetConflicts) != 1 || !got.AssetConflicts[0].Cosmetic {
		t.Fatalf("a dialogue overwrite is shown as harmless, got %+v", got.AssetConflicts)
	}
	data := check([]framework.Mod{pack("C.Data", "Data/Events/Mine"), pack("D.Data", "Data/Events/Mine")})
	if len(data.AssetConflicts) != 1 || data.AssetConflicts[0].Cosmetic {
		t.Fatalf("an event script overwrite still counts, got %+v", data.AssetConflicts)
	}
}

func TestKeyLabelDropsPackScope(t *testing.T) {
	if got := keyLabel("field:301/ContextTags/" + packScopedKey("/mods/A", "{{ModId}}_x") + ".Price"); got != "Field 301/ContextTags/{{ModId}}_x.Price" {
		t.Fatalf("label %q", got)
	}
}

func TestConfigTokenValuesCompareResolved(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(id, configured string) framework.Mod {
		root := t.TempDir()
		testfs.WriteFile(t, root, "manifest.json", packs.Manifest(id))
		content := `{"ConfigSchema":{"Incubation time":{"Default":"5"}},"Changes":[{"Action":"EditData","Target":"Data/FarmAnimals","TargetField":["Dinosaur"],"Entries":{"IncubationTime":"{{Incubation time}}"}}]}`
		testfs.WriteFile(t, root, "content.json", content)
		if configured != "" {
			testfs.WriteFile(t, root, "config.json", `{"Incubation time":"`+configured+`"}`)
		}
		return packs.FromDisk(framework.Mod{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id})
	}
	same := check([]framework.Mod{pack("Em.Dinos", ""), pack("Em.Animals", "5")})
	if len(same.AssetConflicts) != 0 {
		t.Fatalf("both resolve to 5, got %+v", same.AssetConflicts)
	}
	differ := check([]framework.Mod{pack("Em.Dinos2", ""), pack("Em.Animals2", "9")})
	if len(differ.AssetConflicts) != 1 {
		t.Fatalf("5 against 9 clashes, got %+v", differ.AssetConflicts)
	}
}

func TestWhenSummaryPlainDynamicValue(t *testing.T) {
	tokens := []cpTokenDefinition{{name: "livingwithsen", value: "true"}}
	w := parseWhenWithTokens(map[string]json.RawMessage{"LivingWithSen": json.RawMessage(`"true"`)}, nil, nil, tokens)
	if got := whenSummary(w); got != "livingwithsen=true" {
		t.Fatalf("summary %q", got)
	}
}

func TestWhenSummaryNamesAssumedConditions(t *testing.T) {
	w := parseWhen(map[string]json.RawMessage{
		"HasSeenEvent":                       json.RawMessage(`"12345"`),
		"HasFlag: hostPlayer":                json.RawMessage(`"beenToDesert"`),
		"Hearts:Abigail":                     json.RawMessage(`"8"`),
		"Time":                               json.RawMessage(`"{{Range: 0600, 1200}}"`),
		"HasProfession |contains=Gemologist": json.RawMessage(`false`),
		"Season":                             json.RawMessage(`"Spring"`),
	}, nil, nil)
	want := "HasFlag beentodesert; season=spring; also needs: HasSeenEvent, Hearts:Abigail, Time, not HasProfession |contains=Gemologist"
	if got := whenSummary(w); got != want {
		t.Fatalf("summary %q, want %q", got, want)
	}
}
