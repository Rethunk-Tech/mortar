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

	// A cold start, more than an hour on: the side file still answers, and the refresh runs behind it. The site's
	// index changes meanwhile, so the new list shows once the refresh has rebuilt the listing.
	f.chunk0[0]["versions"] = []ver{{Description: "d", VersionNumber: "3.0.0"}, {Description: "d", VersionNumber: "2.0.0"}, {Description: "d", VersionNumber: "1.0.0"}}
	f.third.Store(true)
	clock = clock.Add(2 * refreshAfter)
	stale, err := d.Versions(t.Context(), "lethal-company", "Alice", "MoreCompany", "1.2.3")
	if err != nil || !slices.Equal(stale, first) {
		t.Fatalf("a stale side file = %v, %v; want %v", stale, err, first)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, err := d.Versions(t.Context(), "lethal-company", "Alice", "MoreCompany", "1.2.3")
		if err == nil && len(got) == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the refresh never reached the side file: %v, %v", got, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if f.indexHits.Load() == hits {
		t.Fatal("nothing refreshed the index")
	}
}

func TestVersionsOnAColdCacheLeavesNoListingInMemory(t *testing.T) {
	f := newFake(t)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	memoMu.Lock()
	clear(memo)
	memoMu.Unlock()
	got, err := d.Versions(t.Context(), "lethal-company", "Alice", "MoreCompany", "1.2.3")
	if err != nil || len(got) == 0 {
		t.Fatalf("versions = %v, %v", got, err)
	}
	memoMu.Lock()
	defer memoMu.Unlock()
	if _, held := memo["lethal-company"]; held {
		t.Fatal("a version lookup kept the whole listing in memory")
	}
}
