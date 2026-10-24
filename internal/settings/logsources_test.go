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

// A default the player shows again stays shown, one they hid themselves leaves their list, and hiding a default again
// takes it off the shown list.
func TestShowingADefaultSuppressionKeepsItShown(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	door := HiddenLogSource{Source: "T::Open", Text: "Door"}
	own := HiddenLogSource{Source: "T::Grab"}
	defaults := []HiddenLogSource{door, {Source: "T::Tick"}}
	hidden := func() []HiddenLogSource { return HiddenLogSources(s.Get().GamePrefs("lc"), defaults) }
	if err := svc.HideLogSource("lc", own); err != nil {
		t.Fatal(err)
	}
	if err := svc.HideLogSource("lc", door); err != nil {
		t.Fatal(err)
	}
	if got := hidden(); !slices.Equal(got, []HiddenLogSource{{Source: "T::Tick"}, own, door}) {
		t.Fatalf("a default the player also hid, hidden = %+v", got)
	}
	for _, h := range []HiddenLogSource{door, door, door, own} {
		if err := svc.ShowLogSource("lc", h.Source, h.Text, h.Object); err != nil {
			t.Fatal(err)
		}
	}
	if got := hidden(); !slices.Equal(got, []HiddenLogSource{{Source: "T::Tick"}}) {
		t.Fatalf("after showing, hidden = %+v", got)
	}
	if got := s.Get().GamePrefs("lc").ShownLogSources; !slices.Equal(got, []HiddenLogSource{door}) {
		t.Fatalf("shown = %+v", got)
	}
	if err := svc.HideLogSource("lc", HiddenLogSource{Source: "T::Open"}); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().GamePrefs("lc").ShownLogSources; len(got) != 0 {
		t.Fatalf("hiding the method again left shown = %+v", got)
	}
}
