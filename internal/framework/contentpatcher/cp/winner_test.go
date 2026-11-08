package cp

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestLoadAfterEditConflictIsShownNotCounted(t *testing.T) {
	a := testdataPack(t, "edit_a")
	b := testdataPack(t, "edit_b")
	a.Name = "Edit A"
	a.LoadAfter = []ModID{b.UniqueID}
	a.Dependencies = []Dependency{{UniqueID: b.UniqueID, Required: false}}
	got := check([]Mod{a, b})
	if len(got.Conflicts) != 1 {
		t.Fatalf("got %+v", got.Conflicts)
	}
	c := got.Conflicts[0]
	if c.Kind != "edit" || !c.Cosmetic || c.WinnerKind != WinnerChosen || c.WinnerName != "Edit A" {
		t.Fatalf("got %+v", c)
	}
}

func editPacks(t *testing.T, target string, keys ...string) []Mod {
	t.Helper()
	out := make([]Mod, len(keys))
	for i, key := range keys {
		out[i] = contentPack(t, "Win.P"+string(rune('A'+i)),
			`{"Changes":[{"Action":"EditData","Target":"`+target+`","Entries":{"`+key+`":"`+string(rune('a'+i))+`"}}]}`)
	}
	return out
}

func editConflict(t *testing.T, mods []Mod) Conflict {
	t.Helper()
	got := check(mods)
	if len(got.Conflicts) != 1 {
		t.Fatalf("got %+v", got.Conflicts)
	}
	return got.Conflicts[0]
}

func TestEveryClashingPairOrderedDecidesTheConflict(t *testing.T) {
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)
	m := editPacks(t, "Data/CookingRecipes", "Magic Rock Candy", "Magic Rock Candy", "Stuffed Mushrooms", "Stuffed Mushrooms", "Matcha Latte", "Matcha Latte")
	if c := editConflict(t, m); c.Cosmetic || c.WinnerKind != WinnerUnclear {
		t.Fatalf("unordered: %+v", c)
	}
	m[0].LoadAfter = []ModID{m[1].UniqueID}
	m[3].LoadAfter = []ModID{m[2].UniqueID}
	if c := editConflict(t, m); c.Cosmetic {
		t.Fatalf("one pair unordered: %+v", c)
	}
	m[4].LoadAfter = []ModID{m[5].UniqueID}
	if c := editConflict(t, m); !c.Cosmetic || c.WinnerKind != WinnerPerEntry || c.WinnerID != "" {
		t.Fatalf("every pair ordered: %+v", c)
	}
}

func TestOneWinnerOverPacksThatClashAmongThemselves(t *testing.T) {
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)
	m := editPacks(t, "Data/animationDescriptions", "sleep", "sleep", "sleep", "sleep")
	m[0].LoadAfter = []ModID{m[1].UniqueID, m[2].UniqueID}
	if c := editConflict(t, m); c.Cosmetic {
		t.Fatalf("one rival left over: %+v", c)
	}
	m[0].LoadAfter = append(m[0].LoadAfter, m[3].UniqueID)
	if c := editConflict(t, m); !c.Cosmetic || !Equal(c.WinnerID, m[0].UniqueID) || c.WinnerKind != WinnerChosen || c.WinnerName != m[0].Name {
		t.Fatalf("got %+v", c)
	}
}

func TestAPackClashingOnOtherKeysDoesNotOverwriteTheContest(t *testing.T) {
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)
	pack := func(id string, keys ...string) Mod {
		entries := make([]string, len(keys))
		for n, k := range keys {
			entries[n] = `"` + k + `":"` + id + `"`
		}
		return contentPack(t, "Win.P"+strings.ToUpper(id),
			`{"Changes":[{"Action":"EditData","Target":"Data/CookingRecipes","Entries":{`+strings.Join(entries, ",")+`}}]}`)
	}
	// a and b fight over k1; w clashes with each, but on k2 and k3.
	m := []Mod{pack("a", "k1", "k2"), pack("b", "k1", "k3"), pack("c", "k2", "k3")}
	m[2].LoadAfter = []ModID{m[0].UniqueID, m[1].UniqueID}
	if c := editConflict(t, m); c.Cosmetic {
		t.Fatalf("w never touches k1, so a and b stay unsettled: %+v", c)
	}
	m = []Mod{pack("a", "k1", "k2"), pack("b", "k1", "k3"), pack("c", "k1", "k2", "k3")}
	m[2].LoadAfter = []ModID{m[0].UniqueID, m[1].UniqueID}
	if c := editConflict(t, m); !c.Cosmetic {
		t.Fatalf("w overwrites k1: %+v", c)
	}
}

func TestADependentPackWinsOverWhatItsDependencyBeat(t *testing.T) {
	dataHome(t)
	resetCaches()
	t.Cleanup(resetCaches)
	m := editPacks(t, "Data/animationDescriptions", "sleep", "sleep", "sleep")
	m[0].LoadAfter = []ModID{m[2].UniqueID}
	m[1].Dependencies = append(m[1].Dependencies, Dependency{UniqueID: m[0].UniqueID, Required: true})
	if c := editConflict(t, m); !c.Cosmetic || !Equal(c.WinnerID, m[1].UniqueID) {
		t.Fatalf("the pack that needs the winner loads after both: %+v", c)
	}
}

func TestAnAddOnByOneAuthorStillClashesWithAStranger(t *testing.T) {
	edit := func(author, value string) Mod {
		return packBy(t, author, `{"Changes":[{"Action":"EditData","Target":"Data/Objects","Entries":{"WalkKey":{"Name":"`+value+`"}}}]}`, nil)
	}
	base, child, other := edit("Walk", "Base"), edit("Walk", "Child"), edit("Someone Else", "Other")
	child.Dependencies = []Dependency{{UniqueID: base.UniqueID, Required: true}}
	got := check([]Mod{base, child, other})
	if len(got.Shadowed) != 0 {
		t.Fatalf("neither the add-on nor the base it requires is safe to remove: %+v", got.Shadowed)
	}
	if len(got.Conflicts) != 1 || !slices.Contains(got.Conflicts[0].Keys, child.Key) || got.Conflicts[0].Cosmetic {
		t.Fatalf("the add-on has no order against the stranger, so their clash stays unsettled: %+v", got.Conflicts)
	}
	child.Dependencies[0].Required = false
	if got := check([]Mod{base, child, other}); len(got.Shadowed) != 1 || got.Shadowed[0].Key != base.Key || got.Shadowed[0].Bundled {
		t.Fatalf("an optional dependent does not keep the base it overwrites: %+v", got.Shadowed)
	}
}

func BenchmarkMarkLoadAfterWinner(b *testing.B) {
	const h = 64
	hits := make([]packHit, h)
	for i := range hits {
		hits[i] = packHit{ID: ModID(fmt.Sprintf("a.p%d", i)), Edits: []Patch{{Shapes: whole}}, LoadAfter: map[string]bool{}, Dependencies: map[string]bool{}}
	}
	for i := range hits {
		for j := range hits {
			if i != j {
				hits[i].Rivals = append(hits[i].Rivals, hits[j].ID)
			}
		}
	}
	// Only the last pack settles every pair, the slowest path through overwritten.
	for i := range hits[:h-1] {
		hits[h-1].LoadAfter[hits[i].ID.Fold()] = true
	}
	for b.Loop() {
		var c Conflict
		markLoadAfterWinner(&c, hits)
	}
}
