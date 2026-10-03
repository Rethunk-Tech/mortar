package loadorder

import (
	"slices"
	"testing"
)

func ids(rows []Row) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.UniqueID
	}
	return out
}

func TestDependencyLoadsBeforeDependent(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "Z.Addon", Name: "Aaa Addon", Needs: []string{"Z.Lib"}},
		{UniqueID: "Z.Lib", Name: "Zzz Library"},
		{UniqueID: "A.Solo", Name: "Middle"},
	})
	got := ids(rows)
	lib := slices.Index(got, "Z.Lib")
	addon := slices.Index(got, "Z.Addon")
	if lib < 0 || addon < 0 || lib > addon {
		t.Fatalf("library should load before addon: %v", got)
	}
}

func TestContentPackLoadsAfterFramework(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "Pack.A", Name: "Aaa Pack", ContentPackFor: "Pathoschild.ContentPatcher"},
		{UniqueID: "Pathoschild.ContentPatcher", Name: "Zzz Content Patcher"},
	})
	got := ids(rows)
	if len(got) != 2 || got[0] != "Pathoschild.ContentPatcher" || got[1] != "Pack.A" {
		t.Fatalf("framework then pack: %v", got)
	}
}

func TestOptionalDependencyLoadsBeforeDependent(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "Z.Host", Name: "Aaa Host", Optional: []string{"Z.Opt"}},
		{UniqueID: "Z.Opt", Name: "Zzz Optional"},
	})
	got := ids(rows)
	if got[0] != "Z.Opt" || got[1] != "Z.Host" {
		t.Fatalf("optional dep first: %v", got)
	}
}

func TestCycleDetection(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "A.One", Name: "One", Needs: []string{"A.Two"}},
		{UniqueID: "A.Two", Name: "Two", Needs: []string{"A.One"}},
		{UniqueID: "B.Free", Name: "Free"},
	})
	if len(rows) != 3 || rows[0].UniqueID != "B.Free" || rows[0].Cycle {
		t.Fatalf("free mod first, not a cycle: %+v", rows)
	}
	if !rows[1].Cycle || !rows[2].Cycle {
		t.Fatalf("cycle pair should be flagged: %+v", rows)
	}
}

func TestMissingRequiredDependency(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "A.Mod", Name: "Mod", Needs: []string{"Missing.Lib"}, Optional: []string{"Also.Gone"}},
	})
	if len(rows) != 1 {
		t.Fatalf("still listed: %+v", rows)
	}
	if !slices.Equal(rows[0].MissingRequired, []string{"Missing.Lib"}) {
		t.Fatalf("missing required: %+v", rows[0].MissingRequired)
	}
	if slices.Contains(rows[0].MissingRequired, "Also.Gone") {
		t.Fatalf("optional must not count as missing required: %+v", rows[0])
	}
}

func TestAlphabeticalWhenIndependent(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "B.Mod", Name: "Beta"},
		{UniqueID: "A.Mod", Name: "Alpha"},
	})
	if got := ids(rows); !slices.Equal(got, []string{"A.Mod", "B.Mod"}) {
		t.Fatalf("by name: %v", got)
	}
}
