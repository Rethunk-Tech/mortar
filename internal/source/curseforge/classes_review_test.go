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

// A merged page reads every class from its start to the end of the page, so request count grows with the page number
// and the class count; CurseForge's key allows a modest request rate, so deep pages of a five-class game stall.
func TestMergedSearchRequestsDoNotGrowWithThePageNumber(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		q := r.URL.Query()
		index, _ := strconv.Atoi(q.Get("index"))
		size, _ := strconv.Atoi(q.Get("pageSize"))
		out := `{"data":[`
		for i := index; i < min(index+size, 9000); i++ {
			if i > index {
				out += ","
			}
			out += `{"id":` + q.Get("classId") + strconv.Itoa(i) + `,"name":"m","downloadCount":` + strconv.Itoa(100000-i) + `}`
		}
		_, _ = w.Write([]byte(out + `],"pagination":{"totalCount":9000}}`))
	}))
	t.Cleanup(srv.Close)
	d := Driver{
		URL: srv.URL, Key: func() string { return "k" },
		GameSource: func(string) (components.GameSource, bool) {
			return components.GameSource{ID: "curseforge", Key: "1", GameID: 7, Classes: []string{"1", "2", "3", "4", "5"}}, true
		},
	}
	if _, err := d.Search(t.Context(), source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Page: source.FirstPage + 100}); err != nil {
		t.Fatal(err)
	}
	if n := requests.Load(); n > 50 {
		t.Fatalf("page 101 of 5 classes took %d requests", n)
	}
}
