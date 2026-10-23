package thunderstore

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestFactsComeFromTheCachedIndexAlone(t *testing.T) {
	f := newFake(t)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	if _, err := d.CachedFacts(t.Context(), "lethal-company", map[string]string{"alice-morecompany": "1.0.0"}); err == nil {
		t.Fatal("a community with no cached listing answered")
	}
	if _, err := d.Search(t.Context(), source.Query{Game: "lethal-company", Key: "lethal-company", Page: 1, Version: "1.2.3"}); err != nil {
		t.Fatal(err)
	}
	f.srv.Close()
	got, err := d.CachedFacts(t.Context(), "lethal-company", map[string]string{"alice-morecompany": "v1.0.0", "dave-library": "9.9.9", "nobody-nothing": "1.0.0"})
	if err != nil || len(got) != 2 {
		t.Fatalf("facts %+v %v", got, err)
	}
	want := source.Facts{
		Released: "2025-12-01T00:00:00Z", Latest: "2.0.0", LatestReleased: "2026-01-15T10:00:00Z",
		Updated: "2026-01-01", Rating: 3, Downloads: 501, Categories: []string{"Items"},
	}
	if a := got["alice-morecompany"]; a.Released != want.Released || a.Latest != want.Latest || a.LatestReleased != want.LatestReleased ||
		a.Updated != want.Updated || a.Rating != want.Rating || a.Downloads != want.Downloads || !slices.Equal(a.Categories, want.Categories) {
		t.Fatalf("MoreCompany = %+v, want %+v", a, want)
	}
	// A version the listing lacks has no release date, but the package's newest still shows.
	if l := got["dave-library"]; l.Released != "" || l.Latest != "2.0.0" {
		t.Fatalf("Library = %+v", l)
	}
}
