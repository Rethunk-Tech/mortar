package problems

import "testing"

func TestSupersededWhenTheNamedReplacementIsEnabled(t *testing.T) {
	mods := []Installed{
		{Key: "old", UniqueID: "A.Old", Name: "Old Clock", Enabled: true},
		{Key: "new", UniqueID: "B.Clock", Name: "24h Clock", Enabled: true, UpdateKeys: []string{"Nexus:20794"}},
		{Key: "lone", UniqueID: "C.Lone", Name: "Lone", Enabled: true},
	}
	r := Result{
		Broken: []Broken{{Key: "old", ID: "smapi:A.Old", Name: "Old Clock", Status: "obsolete", Summary: "use [24h Clock](#) instead."}},
		Compat: []Compat{
			{Key: "lone", ID: "smapi:C.Lone", Name: "Lone", Status: "broken", Summary: "use [Missing Mod](#) instead."},
			{Key: "old", ID: "smapi:A.Old", Name: "Old Clock", Status: "broken", Summary: "see https://www.nexusmods.com/stardewvalley/mods/20794"},
		},
	}
	r = superseded(r, "stardewvalley", mods)
	if len(r.Redundant) != 1 || r.Redundant[0].Key != "old" || r.Redundant[0].By[0].Key != "new" {
		t.Fatalf("redundant = %+v", r.Redundant)
	}
	if len(r.Broken) != 0 || len(r.Compat) != 1 || r.Compat[0].Key != "lone" {
		t.Fatalf("rows left: broken %+v compat %+v", r.Broken, r.Compat)
	}
}

func TestSupersededSkipsAModAlreadyRedundant(t *testing.T) {
	mods := []Installed{
		{Key: "old", UniqueID: "A.Old", Name: "Old", Enabled: true},
		{Key: "new", UniqueID: "B.New", Name: "New", Enabled: true},
	}
	r := Result{
		Redundant: []Redundant{{Kind: "shadowed", Key: "old"}},
		Broken:    []Broken{{Key: "old", ID: "smapi:A.Old", Name: "Old", Status: "obsolete", Summary: "use [New](#) instead."}},
	}
	if r = superseded(r, "stardewvalley", mods); len(r.Redundant) != 1 {
		t.Fatalf("redundant = %+v", r.Redundant)
	}
}

func TestRedundantCountsASameJobGroupOnce(t *testing.T) {
	pair := func(key string, by ...string) Redundant {
		r := Redundant{Kind: "sameJob", Key: key}
		for _, b := range by {
			r.By = append(r.By, ModRef{Key: b})
		}
		return r
	}
	rows := []Redundant{
		pair("a", "b", "c"), pair("b", "a", "c"), pair("c", "a", "b"),
		pair("x", "y"), pair("y", "x"),
		{Kind: "sameJob", Key: "small", Covered: true, By: []ModRef{{Key: "big"}}},
		{Kind: "superseded", Key: "old", By: []ModRef{{Key: "new"}}},
	}
	if got := redundantCount(rows); got != 4 {
		t.Fatalf("count = %d, want 4", got)
	}
}
