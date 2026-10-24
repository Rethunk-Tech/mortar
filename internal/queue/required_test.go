package queue

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestMissingRequiredSkipsTheGameAndModsAlreadyThere(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Entries: []profile.Entry{
		{Mods: []profile.Component{{ModID: "carryon", Requires: []string{"game", "vsimgui", "configlib"}}}},
		{Mods: []profile.Component{{ModID: "configlib", Requires: []string{"vsimgui"}}}},
	}}
	if got := missingRequired(p); !slices.Equal(got, []string{"vsimgui"}) {
		t.Fatalf("missing = %v", got)
	}
}
