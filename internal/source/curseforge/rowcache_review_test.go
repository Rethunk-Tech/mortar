package curseforge

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func cachedFor(d Driver) map[string]classRows {
	rowsMu.Lock()
	defer rowsMu.Unlock()
	out := map[string]classRows{}
	for k, v := range rowsCache {
		if strings.HasPrefix(k, d.URL+"|") {
			out[k] = v
		}
	}
	return out
}

// Every distinct search text adds a cache entry per class and none is ever dropped, so typing and paging through many
// searches grows the process by up to a thousand rows per class per query without a bound.
func TestRowCacheIsBounded(t *testing.T) {
	var requests atomic.Int64
	d := countingDriver(t, []string{"1", "2"}, 100, &requests)
	for i := range 200 {
		q := source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Text: "query " + strconv.Itoa(i)}
		if _, err := d.Search(t.Context(), q); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(cachedFor(d)); n > 100 {
		t.Fatalf("%d cache entries after 200 searches", n)
	}
}

// Reading rows must not renew their age: a page served from the cache is as old as the fetch, so rows older than the
// time limit are fetched again however often the search is paged.
func TestRowCacheAgeIsTheFetchTimeNotTheLastRead(t *testing.T) {
	var requests atomic.Int64
	d := countingDriver(t, []string{"1", "2"}, 100, &requests)
	q := source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Text: "age"}
	if _, err := d.Search(t.Context(), q); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-9 * time.Minute)
	rowsMu.Lock()
	for k, v := range rowsCache {
		if strings.HasPrefix(k, d.URL+"|") {
			v.at = old
			rowsCache[k] = v
		}
	}
	rowsMu.Unlock()
	if _, err := d.Search(t.Context(), q); err != nil {
		t.Fatal(err)
	}
	for k, v := range cachedFor(d) {
		if v.at.After(old.Add(time.Second)) {
			t.Fatalf("%s: a read renewed the age of its rows", k)
		}
	}
}

// Rows past their age are fetched again from the start of the class, not appended to.
func TestRowsPastTheirAgeAreFetchedAgain(t *testing.T) {
	var requests atomic.Int64
	d := countingDriver(t, []string{"1", "2"}, 100, &requests)
	q := source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Text: "stale"}
	if _, err := d.Search(t.Context(), q); err != nil {
		t.Fatal(err)
	}
	before := requests.Load()
	rowsMu.Lock()
	for k, v := range rowsCache {
		if strings.HasPrefix(k, d.URL+"|") {
			v.at = time.Now().Add(-rowsTTL - time.Minute)
			rowsCache[k] = v
		}
	}
	rowsMu.Unlock()
	if _, err := d.Search(t.Context(), q); err != nil {
		t.Fatal(err)
	}
	if requests.Load() == before {
		t.Fatal("stale rows were served")
	}
	for _, v := range cachedFor(d) {
		if len(v.mods) != source.PageSize {
			t.Fatalf("a refetch appended to stale rows: %d held, want %d", len(v.mods), source.PageSize)
		}
	}
}

// A class that keeps failing contributes nothing, is named on every page, and the later pages of the classes that
// answered neither repeat nor skip a row.
func TestPagingWhileAClassKeepsFailingNeitherRepeatsNorSkips(t *testing.T) {
	d := pagedDriver(t, []string{"1", "2"},
		func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Query().Get("classId") == "2" {
				http.Error(w, "down", http.StatusBadRequest)
				return true
			}
			return false
		},
		func(url.Values) int { return 200 },
		func(_ url.Values, i int) string {
			return `{"id":` + strconv.Itoa(1000+i) + `,"name":"m","downloadCount":` + strconv.Itoa(100000-i) + `}`
		})
	seen := map[string]bool{}
	next := 0
	for page := range 3 {
		p, err := d.Search(t.Context(), source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Page: source.FirstPage + page})
		if err != nil || len(p.Failed) == 0 {
			t.Fatalf("page %d: %v, failed %v", page, err, p.Failed)
		}
		for _, it := range p.Items {
			if seen[it.ID] || it.ID != strconv.Itoa(1000+next) {
				t.Fatalf("page %d: %s after %d rows (repeat or skip)", page, it.ID, next)
			}
			seen[it.ID] = true
			next++
		}
	}
}
