package problems

import "testing"

func TestHideDismissedAbandoned(t *testing.T) {
	in := []Broken{
		{UniqueID: "A", Status: "abandoned"},
		{UniqueID: "B", Status: "broken"},
	}
	got, dismissed := hideDismissedBroken(in, []string{dismissToken("broken", "a")})
	if len(got) != 1 || got[0].UniqueID != "B" || len(dismissed) != 1 {
		t.Fatalf("got %+v dismissed %+v", got, dismissed)
	}
	leftover, extra := hideDismissedBroken(in, []string{dismissToken("abandoned", "a")})
	if len(leftover) != 2 || len(extra) != 0 {
		t.Fatalf("legacy abandoned token still hides: got %+v dismissed %+v", leftover, extra)
	}
}
