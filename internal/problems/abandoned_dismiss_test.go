package problems

import "testing"

func TestHideDismissedAbandoned(t *testing.T) {
	in := []Broken{
		{UniqueID: "A", Status: "abandoned"},
		{UniqueID: "B", Status: "broken"},
	}
	got := hideDismissedBroken(in, []string{dismissToken("abandoned", "a")})
	if len(got) != 1 || got[0].UniqueID != "B" {
		t.Fatalf("got %+v", got)
	}
}
