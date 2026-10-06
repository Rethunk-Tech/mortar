package contentpatcher

import (
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
	if c.Kind != "edit" || !c.Cosmetic || c.WinnerName != "Edit A wins" {
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
	if c := editConflict(t, m); c.Cosmetic || c.WinnerName != "unclear" {
		t.Fatalf("unordered: %+v", c)
	}
	m[0].LoadAfter = []mod.ID{m[1].ModID()}
	m[3].LoadAfter = []mod.ID{m[2].ModID()}
	if c := editConflict(t, m); c.Cosmetic {
		t.Fatalf("one pair unordered: %+v", c)
	}
	m[4].LoadAfter = []mod.ID{m[5].ModID()}
	if c := editConflict(t, m); !c.Cosmetic || c.WinnerName != decidedPerEntry || c.WinnerID != "" {
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
	if c := editConflict(t, m); !c.Cosmetic || !mod.Equal(c.WinnerID, m[0].ModID()) || c.WinnerName != m[0].Name+" wins" {
		t.Fatalf("got %+v", c)
	}
}
