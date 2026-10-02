package problems

import (
	"encoding/json"
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
	return cpPatch{kind: "edit", shapes: editShapes(t.TempDir(), ch, image), spouse: spouseOf(ch.When)}
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
		{"same map property", `{"MapProperties":{"Music":"x"}}`, `{"MapProperties":{"music":"y"}}`, false, true},
		{"different spouses never apply together", `{"When":{"Query: '{{Spouse}}' = 'Sterling'":true}}`, `{"When":{"Relationship:Jasper":"Married"}}`, true, false},
		{"same spouse still clashes", `{"When":{"Relationship:Jasper":"Married"}}`, `{"When":{"Query: '{{Spouse}}' = 'Jasper'":true}}`, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, b := edit(t, c.a, c.image), edit(t, c.b, c.image)
			if got := editsClash([]cpPatch{a}, []cpPatch{b}); got != c.want {
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
