package loadorder

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func ids(rows []Row) []mod.ID {
	out := make([]mod.ID, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}

func TestDependencyLoadsBeforeDependent(t *testing.T) {
	rows := Resolve([]Mod{
		{ID: "smapi:Z.Addon", Name: "Aaa Addon", Needs: []mod.ID{"smapi:Z.Lib"}},
		{ID: "smapi:Z.Lib", Name: "Zzz Library"},
		{ID: "smapi:A.Solo", Name: "Middle"},
	})
	got := ids(rows)
	lib := slices.Index(got, "smapi:Z.Lib")
	addon := slices.Index(got, "smapi:Z.Addon")
	if lib < 0 || addon < 0 || lib > addon {
		t.Fatalf("library should load before addon: %v", got)
	}
}

func TestContentPackLoadsAfterFramework(t *testing.T) {
	rows := Resolve([]Mod{
		{ID: "smapi:Pack.A", Name: "Aaa Pack", ContentPackFor: "smapi:Pathoschild.ContentPatcher"},
		{ID: "smapi:Pathoschild.ContentPatcher", Name: "Zzz Content Patcher"},
	})
	got := ids(rows)
	if len(got) != 2 || got[0] != "smapi:Pathoschild.ContentPatcher" || got[1] != "smapi:Pack.A" {
		t.Fatalf("framework then pack: %v", got)
	}
}

func TestOptionalDependencyLoadsBeforeDependent(t *testing.T) {
	rows := Resolve([]Mod{
		{ID: "smapi:Z.Host", Name: "Aaa Host", Optional: []mod.ID{"smapi:Z.Opt"}},
		{ID: "smapi:Z.Opt", Name: "Zzz Optional"},
	})
	got := ids(rows)
	if got[0] != "smapi:Z.Opt" || got[1] != "smapi:Z.Host" {
		t.Fatalf("optional dep first: %v", got)
	}
}

func TestCycleDetection(t *testing.T) {
	rows := Resolve([]Mod{
		{ID: "smapi:A.One", Name: "One", Needs: []mod.ID{"smapi:A.Two"}},
		{ID: "smapi:A.Two", Name: "Two", Needs: []mod.ID{"smapi:A.One"}},
		{ID: "smapi:B.Free", Name: "Free"},
	})
	if len(rows) != 3 || rows[0].ID != "smapi:B.Free" || rows[0].Cycle {
		t.Fatalf("free mod first, not a cycle: %+v", rows)
	}
	if !rows[1].Cycle || !rows[2].Cycle {
		t.Fatalf("cycle pair should be flagged: %+v", rows)
	}
}

func TestMissingRequiredDependency(t *testing.T) {
	rows := Resolve([]Mod{
		{ID: "smapi:A.Mod", Name: "Mod", Needs: []mod.ID{"smapi:Missing.Lib"}, Optional: []mod.ID{"smapi:Also.Gone"}},
	})
	if len(rows) != 1 {
		t.Fatalf("still listed: %+v", rows)
	}
	if !slices.Equal(rows[0].MissingRequired, []mod.ID{"smapi:Missing.Lib"}) {
		t.Fatalf("missing required: %+v", rows[0].MissingRequired)
	}
	if slices.Contains(rows[0].MissingRequired, "smapi:Also.Gone") {
		t.Fatalf("optional must not count as missing required: %+v", rows[0])
	}
}

func TestAlphabeticalWhenIndependent(t *testing.T) {
	rows := Resolve([]Mod{
		{ID: "smapi:B.Mod", Name: "Beta"},
		{ID: "smapi:A.Mod", Name: "Alpha"},
	})
	if got := ids(rows); !slices.Equal(got, []mod.ID{"smapi:A.Mod", "smapi:B.Mod"}) {
		t.Fatalf("by name: %v", got)
	}
}
