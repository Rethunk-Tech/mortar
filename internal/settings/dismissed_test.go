package settings

import (
	"slices"
	"testing"
)

func TestRemoveDismissedOffersTokensAgain(t *testing.T) {
	s, _ := open(t)
	for _, tok := range []string{"a", "b", "c"} {
		if err := s.AppendDismissed("gameMods:stardew", tok); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RemoveDismissed("gameMods:stardew", "a", "c"); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().Dismissed["gameMods:stardew"]; !slices.Equal(got, []string{"b"}) {
		t.Fatalf("dismissed = %v", got)
	}
}
