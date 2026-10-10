package curseforge

import (
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
