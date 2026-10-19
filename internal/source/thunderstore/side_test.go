package thunderstore

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestVersionsAnswerFromTheSideFileWithoutTheListing(t *testing.T) {
	f := newFake(t)
	cache := t.TempDir()
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := Driver{URL: f.srv.URL, CacheDir: cache, Now: func() time.Time { return clock }}
	dir := filepath.Join(cache, "thunderstore")

	first, err := d.Versions(t.Context(), "lethal-company", "Alice", "MoreCompany", "1.2.3")
	if err != nil || len(first) == 0 {
		t.Fatalf("first = %v, %v", first, err)
	}
	listings, _ := filepath.Glob(filepath.Join(dir, "lethal-company-"+listingTag+"-*.json"))
	if len(listings) != 1 {
		t.Fatalf("listings = %v", listings)
	}
	// With the listing gone and the index unreachable-by-count, only the side file can answer.
	if err := os.Remove(listings[0]); err != nil {
		t.Fatal(err)
	}
	memoMu.Lock()
	clear(memo)
	memoMu.Unlock()
	hits := f.indexHits.Load()
	again, err := d.Versions(t.Context(), "lethal-company", "alice", "morecompany", "1.2.3")
	if err != nil || !slices.Equal(again, first) {
		t.Fatalf("from the side file = %v, %v; want %v", again, err, first)
	}
	if f.indexHits.Load() != hits {
		t.Fatal("the side file answer asked the site")
	}
	if _, err := os.Stat(listings[0]); !os.IsNotExist(err) {
		t.Fatalf("the listing was rebuilt: %v", err)
	}

	// Older than the index's freshness: the listing is rebuilt and the side file follows it.
	clock = clock.Add(2 * refreshAfter)
	stale, err := d.Versions(t.Context(), "lethal-company", "Alice", "MoreCompany", "1.2.3")
	if err != nil || !slices.Equal(stale, first) {
		t.Fatalf("after expiry = %v, %v", stale, err)
	}
	if f.indexHits.Load() == hits {
		t.Fatal("a stale side file answered without the index")
	}
	if _, err := os.Stat(listings[0]); err != nil {
		t.Fatalf("the fallback did not rebuild the listing: %v", err)
	}
}
