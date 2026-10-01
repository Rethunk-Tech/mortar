package problems

import "testing"

func TestHideDismissedListed(t *testing.T) {
	in := []Missing{
		{UniqueID: "Requirement.Mod", Listed: true},
		{UniqueID: "Manifest.Mod"},
	}
	got := hideDismissedListed(in, []string{dismissToken("listed", "requirement.mod")})
	if len(got) != 1 || got[0].UniqueID != "Manifest.Mod" {
		t.Fatalf("got %+v", got)
	}
}
