package contentpatcher

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestLoadAfterEditConflictIsShownNotCounted(t *testing.T) {
	a := testdataPack(t, "edit_a")
	b := testdataPack(t, "edit_b")
	a.Name = "Edit A"
	a.LoadAfter = []mod.ID{b.ModID()}
	a.Dependencies = []manifest.Dependency{{UniqueID: b.UniqueID, Required: false}}
	got := check([]framework.Mod{a, b})
	if len(got.AssetConflicts) != 1 {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
	c := got.AssetConflicts[0]
	if c.Kind != "edit" || !c.Cosmetic || c.WinnerKind != framework.WinnerChosen || c.WinnerName != "Edit A" {
		t.Fatalf("got %+v", c)
	}
}

func editPacks(t *testing.T, target string, keys ...string) []framework.Mod {
	t.Helper()
	out := make([]framework.Mod, len(keys))
	for i, key := range keys {
		out[i] = tilesheetPack(t, "Win.P"+string(rune('A'+i)),
			`{"Changes":[{"Action":"EditData","Target":"`+target+`","Entries":{"`+key+`":"`+string(rune('a'+i))+`"}}]}`)
	}
	return out
}

func editConflict(t *testing.T, mods []framework.Mod) framework.AssetConflict {
	t.Helper()
	got := check(mods)
	if len(got.AssetConflicts) != 1 {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
	return got.AssetConflicts[0]
}

func TestEveryClashingPairOrderedDecidesTheConflict(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	m := editPacks(t, "Data/CookingRecipes", "Magic Rock Candy", "Magic Rock Candy", "Stuffed Mushrooms", "Stuffed Mushrooms", "Matcha Latte", "Matcha Latte")
	if c := editConflict(t, m); c.Cosmetic || c.WinnerKind != framework.WinnerUnclear {
		t.Fatalf("unordered: %+v", c)
	}
	m[0].LoadAfter = []mod.ID{m[1].ModID()}
	m[3].LoadAfter = []mod.ID{m[2].ModID()}
	if c := editConflict(t, m); c.Cosmetic {
		t.Fatalf("one pair unordered: %+v", c)
	}
	m[4].LoadAfter = []mod.ID{m[5].ModID()}
	if c := editConflict(t, m); !c.Cosmetic || c.WinnerKind != framework.WinnerPerEntry || c.WinnerID != "" {
		t.Fatalf("every pair ordered: %+v", c)
	}
}

func TestOneWinnerOverPacksThatClashAmongThemselves(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	m := editPacks(t, "Data/animationDescriptions", "sleep", "sleep", "sleep", "sleep")
	m[0].LoadAfter = []mod.ID{m[1].ModID(), m[2].ModID()}
	if c := editConflict(t, m); c.Cosmetic {
		t.Fatalf("one rival left over: %+v", c)
	}
	m[0].LoadAfter = append(m[0].LoadAfter, m[3].ModID())
	if c := editConflict(t, m); !c.Cosmetic || !mod.Equal(c.WinnerID, m[0].ModID()) || c.WinnerKind != framework.WinnerChosen || c.WinnerName != m[0].Name {
		t.Fatalf("got %+v", c)
	}
}

func TestADependentPackWinsOverWhatItsDependencyBeat(t *testing.T) {
	testfs.DataHome(t)
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)
	m := editPacks(t, "Data/animationDescriptions", "sleep", "sleep", "sleep")
	m[0].LoadAfter = []mod.ID{m[2].ModID()}
	m[1].Dependencies = append(m[1].Dependencies, manifest.Dependency{UniqueID: m[0].UniqueID, Required: true})
	if c := editConflict(t, m); !c.Cosmetic || !mod.Equal(c.WinnerID, m[1].ModID()) {
		t.Fatalf("the pack that needs the winner loads after both: %+v", c)
	}
}

func TestAnAddOnByOneAuthorStillClashesWithAStranger(t *testing.T) {
	edit := func(author, value string) framework.Mod {
		return packBy(t, author, `{"Changes":[{"Action":"EditData","Target":"Data/Objects","Entries":{"WalkKey":{"Name":"`+value+`"}}}]}`, nil)
	}
	base, child, other := edit("Walk", "Base"), edit("Walk", "Child"), edit("Someone Else", "Other")
	child.Dependencies = []manifest.Dependency{{UniqueID: base.UniqueID, Required: true}}
	got := check([]framework.Mod{base, child, other})
	if len(got.Redundant) != 0 {
		t.Fatalf("neither the add-on nor the base it requires is safe to remove: %+v", got.Redundant)
	}
	if len(got.AssetConflicts) != 1 || !slices.Contains(got.AssetConflicts[0].Keys, child.Key) || got.AssetConflicts[0].Cosmetic {
		t.Fatalf("the add-on has no order against the stranger, so their clash stays unsettled: %+v", got.AssetConflicts)
	}
	child.Dependencies[0].Required = false
	if got := check([]framework.Mod{base, child, other}); len(got.Redundant) != 1 || got.Redundant[0].Key != base.Key || got.Redundant[0].Kind != "shadowed" {
		t.Fatalf("an optional dependent does not keep the base it overwrites: %+v", got.Redundant)
	}
}

func BenchmarkMarkLoadAfterWinner(b *testing.B) {
	const h = 64
	hits := make([]packHit, h)
	for i := range hits {
		hits[i] = packHit{id: mod.ID(fmt.Sprintf("a.p%d", i)), loadAfter: map[string]bool{}, dependencies: map[string]bool{}}
	}
	for i := range hits {
		for j := range hits {
			if i != j {
				hits[i].rivals = append(hits[i].rivals, hits[j].id)
			}
		}
	}
	// Only the last pack settles every pair, the slowest path through overwritten.
	for i := range hits[:h-1] {
		hits[h-1].loadAfter[hits[i].id.Fold()] = true
	}
	for b.Loop() {
		var c framework.AssetConflict
		markLoadAfterWinner(&c, hits)
	}
}
