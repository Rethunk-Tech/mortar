package control

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestDependentsOfSplitsRequiredFromOptional(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Entries: []profile.Entry{{Mods: []profile.Component{
		{ID: "smapi:Me.Lib"},
		{ID: "smapi:Me.Needs", Needs: []mod.ID{"smapi:me.lib"}},
		{ID: "smapi:Me.Likes", Needs: []mod.ID{"smapi:Me.Lib"}, Optional: []mod.ID{"smapi:ME.LIB"}},
	}}}}
	required, optional := dependentsOf(p, "smapi:Me.Lib")
	if !slices.Equal(required, []mod.ID{"smapi:Me.Needs"}) || !slices.Equal(optional, []mod.ID{"smapi:Me.Likes"}) {
		t.Fatalf("required %v, optional %v", required, optional)
	}
}
