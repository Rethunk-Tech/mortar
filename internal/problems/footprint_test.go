package problems

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func change(t *testing.T, raw string) cpChange {
	t.Helper()
	var ch cpChange
	if err := json.Unmarshal([]byte(raw), &ch); err != nil {
		t.Fatal(err)
	}
	return ch
}

func edit(t *testing.T, raw string, image bool) cpPatch {
	t.Helper()
	ch := change(t, raw)
	return cpPatch{kind: "edit", shapes: editShapes(t.TempDir(), ch, image), spouse: spouseOf(ch.When), places: placesOf(ch.When)}
}

func TestEditsClash(t *testing.T) {
	cases := []struct {
		name  string
		a, b  string
		image bool
		want  bool
	}{
		{"disjoint image areas", `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16}}`, `{"ToArea":{"X":16,"Y":0,"Width":16,"Height":16}}`, true, false},
		{"overlapping image areas", `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16}}`, `{"ToArea":{"X":8,"Y":8,"Width":16,"Height":16}}`, true, true},
		{"FromArea sizes a patch at the top-left", `{"FromArea":{"X":64,"Y":64,"Width":16,"Height":16}}`, `{"ToArea":{"X":32,"Y":0,"Width":16,"Height":16}}`, true, false},
		{"tokenized area is the whole asset", `{"ToArea":{"X":"{{x}}","Y":0,"Width":16,"Height":16}}`, `{"ToArea":{"X":200,"Y":200,"Width":1,"Height":1}}`, true, true},
		{"map tiles on different spots", `{"MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Back","SetIndex":5}]}`, `{"MapTiles":[{"Position":{"X":2,"Y":1},"Layer":"Back","SetIndex":5}]}`, false, false},
		{"same tile, other layer", `{"MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Back","SetIndex":5}]}`, `{"MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Front","SetIndex":5}]}`, false, false},
		{"tile inside a map patch area", `{"FromFile":"p.tmx","ToArea":{"X":0,"Y":0,"Width":10,"Height":10}}`, `{"MapTiles":[{"Position":{"X":3,"Y":3},"Layer":"Buildings","SetIndex":1}]}`, false, true},
		{"property-only tile edits merge", `{"MapTiles":[{"Position":{"X":"{{Random: 1, 2}}","Y":1},"Layer":"Back","SetProperties":{"Light":"1"}}]}`, `{"FromFile":"p.tmx"}`, false, false},
		{"different map properties", `{"MapProperties":{"Music":"x"}}`, `{"MapProperties":{"Light":"y"}}`, false, false},
		{"same map property, different values", `{"MapProperties":{"Music":"x"}}`, `{"MapProperties":{"music":"y"}}`, false, true},
		{"same map property, same value", `{"MapProperties":{"AllowGiantCrops":"T"}}`, `{"MapProperties":{"allowgiantcrops":"t"}}`, false, false},
		{"different spouses never apply together", `{"When":{"Query: '{{Spouse}}' = 'Sterling'":true}}`, `{"When":{"Relationship:Jasper":"Married"}}`, true, false},
		{"same spouse still clashes", `{"When":{"Relationship:Jasper":"Married"}}`, `{"When":{"Query: '{{Spouse}}' = 'Jasper'":true}}`, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, b := edit(t, c.a, c.image), edit(t, c.b, c.image)
			if got, _ := editsClash([]cpPatch{a}, []cpPatch{b}); got != c.want {
				t.Fatalf("clash = %v, shapes %v / %v", got, a.shapes, b.shapes)
			}
		})
	}
}

func TestAdditiveEditHasNoShape(t *testing.T) {
	if s := editShapes(t.TempDir(), change(t, `{"AddWarps":["1 2 Town 3 4"],"TextOperations":[{"Operation":"Append"}]}`), false); len(s) != 0 {
		t.Fatalf("shapes %v", s)
	}
}

func TestPlacesMakeEditsExclusive(t *testing.T) {
	a := edit(t, `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16},"When":{"LocationName":"EastScarp_Village"}}`, true)
	b := edit(t, `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16},"When":{"LocationName |contains=Custom_Umuwi":true}}`, true)
	c := edit(t, `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16},"When":{"LocationName":"Custom_Umuwi, EastScarp_Village"}}`, true)
	if clash, _ := editsClash([]cpPatch{a}, []cpPatch{b}); clash {
		t.Fatal("different locations clashed")
	}
	if clash, _ := editsClash([]cpPatch{a}, []cpPatch{c}); !clash {
		t.Fatal("shared location did not clash")
	}
}

func TestMapPatchWithoutToAreaUsesTheSourceMapSize(t *testing.T) {
	dir := t.TempDir()
	tmx := `<?xml version="1.0"?><map version="1.0" orientation="orthogonal" width="10" height="8" tilewidth="16"></map>`
	if err := os.WriteFile(filepath.Join(dir, "p.tmx"), []byte(tmx), 0o600); err != nil {
		t.Fatal(err)
	}
	s := editShapes(dir, change(t, `{"FromFile":"p.tmx"}`), false)
	if len(s) != 1 || s[0].kind != 'r' || s[0].w != 10 || s[0].h != 8 {
		t.Fatalf("shapes %v", s)
	}
	if s := editShapes(dir, change(t, `{"FromFile":"p.tbin"}`), false); len(s) != 1 || s[0].kind != 'w' {
		t.Fatalf("tbin should be whole: %v", s)
	}
}

func TestHasModWithEmptyInput(t *testing.T) {
	w := parseWhen(map[string]json.RawMessage{"HasMod: |contains=Other.Mod": json.RawMessage(`false`)}, map[string]bool{}, nil)
	if len(w.noneOf) != 1 || w.noneOf[0] != "other.mod" {
		t.Fatalf("when %+v", w)
	}
}

func TestHarmlessOverlaps(t *testing.T) {
	area := `"ToArea":{"X":0,"Y":0,"Width":16,"Height":16}`
	mapEdit := func(when string) cpPatch {
		return edit(t, `{"FromFile":"p.tmx",`+area+when+`}`, false)
	}
	img := func(when string) cpPatch {
		p := edit(t, `{`+area+when+`}`, true)
		p.image = true
		return p
	}
	tiny := edit(t, `{"FromFile":"p.tmx","ToArea":{"X":"{{x}}","Y":"{{y}}","Width":1,"Height":1}}`, false)
	cases := []struct {
		name  string
		a, b  cpPatch
		minor bool
	}{
		{"two image edits are cosmetic", img(""), img(""), true},
		{"two map edits matter", mapEdit(""), mapEdit(""), false},
		{"a map edit in one location only", mapEdit(`,"When":{"LocationName":"Cave"}`), mapEdit(""), true},
		{"an image edit in one weather only", img(`,"When":{"Weather":"Storm"}`), mapEdit(""), true},
		{"one tile at a computed spot", tiny, mapEdit(""), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clash, minor := editsClash([]cpPatch{c.a}, []cpPatch{c.b})
			if !clash || minor != c.minor {
				t.Fatalf("clash=%v minor=%v", clash, minor)
			}
		})
	}
}
