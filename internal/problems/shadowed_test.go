package problems

import (
	"context"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func shadowedNames(t *testing.T, mods ...Installed) map[string][]string {
	t.Helper()
	_, _, shadowed := assetConflictScan(mods)
	out := map[string][]string{}
	for _, r := range shadowed {
		if r.Kind != "shadowed" {
			t.Fatalf("kind = %q", r.Kind)
		}
		for _, by := range r.By {
			out[r.Key] = append(out[r.Key], by.Key)
		}
	}
	return out
}

func TestShadowedPacks(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(content string, loadAfter ...Installed) Installed {
		m := syntheticLoadPack(t, content, nil)
		for _, other := range loadAfter {
			m.LoadAfter = append(m.LoadAfter, other.ModID())
		}
		return m
	}
	const small = `{"Changes":[
		{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"a.png","ToArea":{"X":0,"Y":0,"Width":32,"Height":32}},
		{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"a"},"Fields":{"2":{"0":"b"}}}
	]}`

	t.Run("a later pack rewriting every pixel and key shadows the pack", func(t *testing.T) {
		a := pack(small)
		b := pack(`{"Changes":[
			{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"b.png","ToArea":{"X":0,"Y":0,"Width":64,"Height":64},"Priority":"Late"},
			{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"x","3":"y"},"Fields":{"2":{"0":"z"}},"Priority":"Late"}
		]}`)
		got := shadowedNames(t, a, b)
		if len(got) != 1 || !slices.Equal(got[a.Key], []string{b.Key}) {
			t.Fatalf("got %v", got)
		}
		r := Check(context.Background(), fakeMeta{}, testEnv, []Installed{a, b})
		if len(r.Redundant) != 1 || r.Redundant[0].Key != a.Key || r.Redundant[0].By[0].Name != b.Name {
			t.Fatalf("Check redundant = %+v", r.Redundant)
		}
	})

	t.Run("areas from two packs cover together", func(t *testing.T) {
		a := pack(`{"Changes":[{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"a.png","ToArea":{"X":0,"Y":0,"Width":32,"Height":32}}]}`)
		left := pack(`{"Changes":[{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"l.png","ToArea":{"X":0,"Y":0,"Width":16,"Height":32},"Priority":"Late"}]}`)
		right := pack(`{"Changes":[{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"r.png","ToArea":{"X":16,"Y":0,"Width":16,"Height":32},"Priority":"Late"}]}`)
		got := shadowedNames(t, a, left, right)
		if len(got[a.Key]) != 2 {
			t.Fatalf("got %v", got)
		}
		if got := shadowedNames(t, a, left); len(got) != 0 {
			t.Fatalf("half covered: %v", got)
		}
	})

	t.Run("equal priority needs a known load order", func(t *testing.T) {
		a := pack(`{"Changes":[{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"a"}}]}`)
		b := pack(`{"Changes":[{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"b"}}]}`)
		if got := shadowedNames(t, a, b); len(got) != 0 {
			t.Fatalf("unordered: %v", got)
		}
		after := pack(`{"Changes":[{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"b"}}]}`, a)
		if got := shadowedNames(t, a, after); !slices.Equal(got[a.Key], []string{after.Key}) {
			t.Fatalf("load after: %v", got)
		}
	})

	t.Run("map tiles are covered on their own layer only", func(t *testing.T) {
		a := pack(`{"Changes":[{"Action":"EditMap","Target":"Maps/Town","MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Buildings","SetIndex":5}]}]}`)
		same := pack(`{"Changes":[{"Action":"EditMap","Target":"Maps/Town","Priority":"Late","MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Buildings","SetIndex":6}]}]}`)
		other := pack(`{"Changes":[{"Action":"EditMap","Target":"Maps/Town","Priority":"Late","MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Front","SetIndex":6}]}]}`)
		if got := shadowedNames(t, a, same); len(got[a.Key]) != 1 {
			t.Fatalf("same layer: %v", got)
		}
		if got := shadowedNames(t, a, other); len(got) != 0 {
			t.Fatalf("other layer: %v", got)
		}
	})

	t.Run("a Load loses only to a higher priority Load", func(t *testing.T) {
		low := pack(`{"Changes":[{"Action":"Load","Target":"Data/Thing","FromFile":"a.json","Priority":"Low"}]}`)
		high := pack(`{"Changes":[{"Action":"Load","Target":"Data/Thing","FromFile":"b.json","Priority":"High"}]}`)
		if got := shadowedNames(t, low, high); !slices.Equal(got[low.Key], []string{high.Key}) {
			t.Fatalf("got %v", got)
		}
		if got := shadowedNames(t, low); len(got) != 0 {
			t.Fatalf("lone load: %v", got)
		}
	})

	cover := `{"Changes":[
		{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"b.png","ToArea":{"X":0,"Y":0,"Width":64,"Height":64},"Priority":"Late"},
		{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"x"},"Fields":{"2":{"0":"z"}},"Priority":"Late"}
	]}`
	for name, tc := range map[string]struct{ checked, coverer string }{
		"a pack with no edits":        {`{"Changes":[]}`, cover},
		"an entry nobody else writes": {`{"Changes":[{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"a","9":"new"}}]}`, cover},
		"a conditional coverer": {small, `{"Changes":[
			{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"b.png","ToArea":{"X":0,"Y":0,"Width":64,"Height":64},"Priority":"Late","When":{"Time":"0600"}},
			{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"x"},"Fields":{"2":{"0":"z"}},"Priority":"Late"}
		]}`},
		"an overlay coverer": {small, `{"Changes":[
			{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"b.png","ToArea":{"X":0,"Y":0,"Width":64,"Height":64},"Priority":"Late","PatchMode":"Overlay"},
			{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"x"},"Fields":{"2":{"0":"z"}},"Priority":"Late"}
		]}`},
		"a change that also adds warps": {
			`{"Changes":[{"Action":"EditMap","Target":"Maps/Town","AddWarps":["1 1 Farm 2 2"],"MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Buildings","SetIndex":5}]}]}`,
			`{"Changes":[{"Action":"EditMap","Target":"Maps/Town","Priority":"Late","MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Buildings","SetIndex":6}]}]}`,
		},
		"a text operation beside the covered entries": {`{"Changes":[
			{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"a"}},
			{"Action":"EditData","Target":"Data/Objects","TextOperations":[{"Operation":"Append","Target":["Entries","1"],"Value":"x"}]}
		]}`, cover},
		"a tokenized area": {`{"Changes":[{"Action":"EditImage","Target":"Maps/springobjects","FromFile":"a.png","ToArea":{"X":"{{x}}","Y":0,"Width":16,"Height":16}}]}`, cover},
	} {
		t.Run("not shadowed: "+name, func(t *testing.T) {
			a, b := pack(tc.checked), pack(tc.coverer)
			if got := shadowedNames(t, a, b); len(got[a.Key]) != 0 {
				t.Fatalf("got %v", got)
			}
		})
	}
}
