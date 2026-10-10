package curseforge

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func countingDriver(t *testing.T, classes []string, rows int, requests *atomic.Int64) Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		q := r.URL.Query()
		index, _ := strconv.Atoi(q.Get("index"))
		size, _ := strconv.Atoi(q.Get("pageSize"))
		out := `{"data":[`
		for i := index; i < min(index+size, rows); i++ {
			if i > index {
				out += ","
			}
			out += `{"id":` + q.Get("classId") + strconv.Itoa(i) + `,"name":"m","downloadCount":` + strconv.Itoa(100000-i) + `}`
		}
		_, _ = w.Write([]byte(out + `],"pagination":{"totalCount":` + strconv.Itoa(rows) + `}}`))
	}))
	t.Cleanup(srv.Close)
	return Driver{
		URL: srv.URL, Key: func() string { return "k" },
		GameSource: func(string) (components.GameSource, bool) {
			return components.GameSource{ID: "curseforge", Key: classes[0], GameID: 7, Classes: classes}, true
		},
	}
}

func TestNextMergedPageCostsAtMostOneRequestPerClass(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	classes := []string{"1", "2", "3"}
	d := countingDriver(t, classes, 900, &requests)
	q := source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Page: source.FirstPage}
	if _, err := d.Search(t.Context(), q); err != nil {
		t.Fatal(err)
	}
	first := requests.Load()
	q.Page++
	page2, err := d.Search(t.Context(), q)
	if err != nil || len(page2.Items) != source.PageSize {
		t.Fatalf("%+v %v", page2, err)
	}
	if next := requests.Load() - first; next > int64(len(classes)) {
		t.Fatalf("the next page took %d requests, want at most %d", next, len(classes))
	}
	q.Page--
	before := requests.Load()
	if _, err := d.Search(t.Context(), q); err != nil || requests.Load() != before {
		t.Fatalf("a page already held cost %d requests: %v", requests.Load()-before, err)
	}
}

func TestMergedSearchStopsAtTheDepthCap(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	d := countingDriver(t, []string{"1", "2"}, 9000, &requests)
	last := mergedCap/source.PageSize - 1
	page, err := d.Search(t.Context(), source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Page: source.FirstPage + last})
	if err != nil || len(page.Items) != source.PageSize || page.Total != mergedCap {
		t.Fatalf("last page: %d items, total %d, %v", len(page.Items), page.Total, err)
	}
	before := requests.Load()
	past, err := d.Search(t.Context(), source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Page: source.FirstPage + last + 1})
	if err != nil || len(past.Items) != 0 || past.Total != mergedCap || requests.Load() != before {
		t.Fatalf("past the cap: %+v, %d requests, %v", past, requests.Load()-before, err)
	}
}
