package sharesvc

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestEntryNamesListEachNameOnce(t *testing.T) {
	e := profile.Entry{Key: "nexus-1-1", Mods: []profile.Component{
		{ID: "smapi:a.Main", Name: "Tea Shop"},
		{ID: "smapi:a.CP", Name: "Tea Shop"},
		{ID: "smapi:a.Extra", Name: "Tea Shop Extras"},
	}}
	if got := entryNames(e, false); !slices.Equal(got, []string{"Tea Shop", "Tea Shop Extras"}) {
		t.Fatalf("names %v", got)
	}
}
