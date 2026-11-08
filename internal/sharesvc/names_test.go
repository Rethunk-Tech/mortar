package sharesvc

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
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

// A mod disabled under a case variant of its manifest id is off for the profile, so it is not named as carried.
func TestEntryNamesSkipAModDisabledByAnotherCase(t *testing.T) {
	e := profile.Entry{Key: "nexus-1-1", Disabled: []mod.ID{"smapi:a.off"}, Mods: []profile.Component{
		{ID: "smapi:a.Main", Name: "Tea Shop"},
		{ID: "smapi:A.Off", Name: "Tea Shop Extras"},
	}}
	if got := entryNames(e, false); !slices.Equal(got, []string{"Tea Shop"}) {
		t.Fatalf("names %v", got)
	}
}
