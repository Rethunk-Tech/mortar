package problems

import "testing"

func TestHideDismissedAbandoned(t *testing.T) {
	in := []Broken{
		{ID: "smapi:A", Status: "abandoned"},
		{ID: "smapi:B", Status: "broken"},
	}
	got, dismissed := hideDismissedBroken(in, []string{dismissToken("broken", "smapi:a")})
	if len(got) != 1 || got[0].ID != "smapi:B" || len(dismissed) != 1 {
		t.Fatalf("got %+v dismissed %+v", got, dismissed)
	}
	leftover, extra := hideDismissedBroken(in, []string{dismissToken("abandoned", "smapi:a")})
	if len(leftover) != 2 || len(extra) != 0 {
		t.Fatalf("a token of another kind hides the row: got %+v dismissed %+v", leftover, extra)
	}
}
