package problems

import "testing"

func TestHideDismissedSoftOnly(t *testing.T) {
	in := []AssetConflict{
		{Kind: "load", Target: "a"},
		{Kind: "edit", Target: "b"},
	}
	got, dismissed := hideDismissed(in, []string{dismissToken("edit", "b"), dismissToken("load", "a")})
	if len(got) != 0 || len(dismissed) != 2 {
		t.Fatalf("got %+v dismissed %+v", got, dismissed)
	}
}
