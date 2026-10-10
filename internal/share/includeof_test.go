package share

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestIncludeOfFollowsTheShareSettings(t *testing.T) {
	t.Parallel()
	if got, want := IncludeOf(settings.Settings{}), (Include{FomodChoices: true, Notes: true, ConfigFiles: true, ProblemChoices: true}); !sameSwitches(got, want) {
		t.Fatalf("with nothing chosen = %+v, want %+v", got, want)
	}
	on, off := true, false
	set := settings.Settings{
		ShareIncludeDisabledMods:   &on,
		ShareIncludeFomodChoices:   &off,
		ShareIncludeNotes:          &off,
		ShareIncludeConfigFiles:    &off,
		ShareIncludeProblemChoices: &off,
	}
	if got, want := IncludeOf(set), (Include{DisabledMods: true}); !sameSwitches(got, want) {
		t.Fatalf("with every setting flipped = %+v, want %+v", got, want)
	}
}

func sameSwitches(a, b Include) bool {
	return a.DisabledMods == b.DisabledMods && a.FomodChoices == b.FomodChoices && a.Notes == b.Notes &&
		a.ConfigFiles == b.ConfigFiles && a.ProblemChoices == b.ProblemChoices
}
