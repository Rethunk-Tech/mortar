package settings

import (
	"slices"
	"testing"
)

// A broader hide replaces the narrower ones it covers, and one already covered adds nothing.
func TestHidingALogSourceKeepsOnlyTheBroadestEntry(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	hide := func(src HiddenLogSource) {
		t.Helper()
		if err := svc.HideLogSource("lc", src); err != nil {
			t.Fatal(err)
		}
	}
	got := func() []HiddenLogSource { return s.Get().GamePrefs("lc").HiddenLogSources }
	box := HiddenLogSource{Source: "unity:BoxCollider", Object: "Ship/Door"}
	hide(box)
	hide(HiddenLogSource{Source: "unity:BoxCollider", Object: "Ship/Switch"})
	hide(HiddenLogSource{Source: "T::Grab", Text: "Picked up scrap Bolt"})
	hide(HiddenLogSource{Source: "unity:BoxCollider"})
	hide(box)
	hide(HiddenLogSource{Source: "T::Grab"})
	want := []HiddenLogSource{{Source: "unity:BoxCollider"}, {Source: "T::Grab"}}
	if !slices.Equal(got(), want) {
		t.Fatalf("hidden = %+v", got())
	}
}
