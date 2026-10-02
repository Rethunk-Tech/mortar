package problems

import "testing"

func TestHideDismissedAbandoned(t *testing.T) {
	in := []Broken{
		{UniqueID: "A", Status: "abandoned"},
		{UniqueID: "B", Status: "broken"},
	}
	got, dismissed := hideDismissedBroken(in, []string{dismissToken("abandoned", "a")})
	if len(got) != 1 || got[0].UniqueID != "B" || len(dismissed) != 1 {
		t.Fatalf("got %+v dismissed %+v", got, dismissed)
	}
}
