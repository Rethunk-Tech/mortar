package thunderstore

import "testing"

// A package the chunked listing caught twice is kept once, as its newer entry, so Browse never shows it twice.
func TestAPackageListedTwiceIsKeptOnce(t *testing.T) {
	got := distinct([]pkg{
		{Owner: "akechii", Name: "DiscountAlert", Updated: "2025-01-01"},
		{Owner: "Other", Name: "Mod", Updated: "2025-01-01"},
		{Owner: "Akechii", Name: "discountalert", Updated: "2025-02-01"},
	})
	if len(got) != 2 || got[0].Updated != "2025-02-01" || got[1].Name != "Mod" {
		t.Fatalf("distinct = %+v", got)
	}
}
