package browse

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// A Browse hit is judged by its newest version: listed through it, it is broken; past it, it is fixed.
func TestABrowseHitIsBrokenThroughItsListedVersion(t *testing.T) {
	list := components.BrokenList{Entries: []components.BrokenEntry{
		{IDs: components.BrokenIDs{Thunderstore: "Evaisa-LethalThings"}, Through: "0.10.13", Reason: "r"},
		{IDs: components.BrokenIDs{Nexus: 7}, Through: "1.0", Reason: "r"},
	}}
	for _, c := range []struct {
		it   Item
		want bool
	}{
		{Item{Source: "thunderstore", ID: "Evaisa-LethalThings", Version: "0.10.13"}, true},
		{Item{Source: "thunderstore", ID: "Evaisa-LethalThings", Version: "0.11.0"}, false},
		{Item{Source: "nexus", ID: "7", Version: "1.0"}, true},
		{Item{Source: "nexus", ID: "8", Version: "1.0"}, false},
	} {
		if _, got := list.Find(brokenModOf(c.it), ""); got != c.want {
			t.Errorf("%+v: broken %v, want %v", c.it, got, c.want)
		}
	}
}
