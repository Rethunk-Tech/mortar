package thunderstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestABuildRemovesTheCommunitysOtherListings(t *testing.T) {
	f := newFake(t)
	cache := t.TempDir()
	dir := filepath.Join(cache, "thunderstore")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "lethal-company-c1-0123abcd.json")
	otherKey := filepath.Join(dir, "lethal-company-custom-c2-0123abcd.json")
	for _, p := range []string{stale, otherKey} {
		if err := os.WriteFile(p, []byte("[]"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	d := Driver{URL: f.srv.URL, CacheDir: cache}
	if _, err := d.Search(t.Context(), source.Query{Game: "lethal-company", Key: "lethal-company", Page: 1, Version: "1.2.3"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("a listing built under an earlier tag stayed: %v", err)
	}
	if _, err := os.Stat(otherKey); err != nil {
		t.Fatalf("another community's listing was removed: %v", err)
	}
	left, _ := filepath.Glob(filepath.Join(dir, "lethal-company-"+listingTag+"-*.json"))
	if len(left) != 1 {
		t.Fatalf("listings left: %v", left)
	}
}
