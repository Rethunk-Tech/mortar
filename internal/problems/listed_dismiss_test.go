package problems

import "testing"

func TestHideDismissedListed(t *testing.T) {
	in := []Missing{
		{ID: "smapi:Requirement.Mod", Listed: true},
		{ID: "smapi:Manifest.Mod"},
	}
	got, dismissed := hideDismissedListed(in, []string{dismissToken("listed", "smapi:requirement.mod")})
	if len(got) != 1 || got[0].ID != "smapi:Manifest.Mod" || len(dismissed) != 1 {
		t.Fatalf("got %+v dismissed %+v", got, dismissed)
	}
}
