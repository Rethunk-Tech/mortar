package curseforge

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("classId") == "2" {
			http.Error(w, "down", http.StatusBadRequest)
			return
		}
		index, _ := strconv.Atoi(q.Get("index"))
		size, _ := strconv.Atoi(q.Get("pageSize"))
		out := `{"data":[`
		for i := index; i < min(index+size, 200); i++ {
			if i > index {
				out += ","
			}
			out += `{"id":` + strconv.Itoa(1000+i) + `,"name":"m","downloadCount":` + strconv.Itoa(100000-i) + `}`
		}
		_, _ = w.Write([]byte(out + `],"pagination":{"totalCount":200}}`))
	}))
	t.Cleanup(srv.Close)
	d := Driver{
		URL: srv.URL, Key: func() string { return "k" },
		GameSource: func(string) (components.GameSource, bool) {
			return components.GameSource{ID: "curseforge", Key: "1", GameID: 7, Classes: []string{"1", "2"}}, true
		},
	}
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
