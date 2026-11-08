package contentpatcher

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
)

// A data key's label is Mortar's wording, without the pack scoping a token key carries.
func TestKeyLabelDropsPackScope(t *testing.T) {
	if got := keyLabel("field:301/ContextTags/@/mods/A|{{ModId}}_x.Price"); got != "Field 301/ContextTags/{{ModId}}_x.Price" {
		t.Fatalf("label %q", got)
	}
}

func TestWhenSummaryPlainDynamicValue(t *testing.T) {
	w := cp.When{Dynamic: []cp.DynamicCondition{{Name: "livingwithsen", Values: []string{"true"}, Expected: true}}}
	if got := whenSummary(w); got != "livingwithsen=true" {
		t.Fatalf("summary %q", got)
	}
}

func TestWhenSummaryNamesAssumedConditions(t *testing.T) {
	w := cp.When{
		Places:  map[string][]string{"season": {"spring"}},
		Flags:   []cp.FlagCondition{{Name: "beentodesert", Present: true}},
		Assumed: []string{"HasSeenEvent", "Hearts:Abigail", "Time", "not HasProfession |contains=Gemologist"},
	}
	want := "HasFlag beentodesert; season=spring; also needs: HasSeenEvent, Hearts:Abigail, Time, not HasProfession |contains=Gemologist"
	if got := whenSummary(w); got != want {
		t.Fatalf("summary %q, want %q", got, want)
	}
}

// The conflict names the cells its packs share, and Mortar words them as pixel positions.
func TestItemConflictInfo(t *testing.T) {
	if got := itemConflictInfo([]string{"13,12"}); got != "cell (208, 192)" {
		t.Fatalf("item info = %q", got)
	}
	if got := itemConflictInfo(nil); got != "" {
		t.Fatalf("no cells = %q", got)
	}
}
