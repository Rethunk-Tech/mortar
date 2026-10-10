package curseforge

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// classFake serves, per classId, mods whose download counts are given in descending order.
func classFake(t *testing.T, downloads map[string][]int) Driver {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		counts, ok := downloads[q.Get("classId")]
		if r.URL.Path != "/mods/search" || !ok {
			http.NotFound(w, r)
			return
		}
		index, _ := strconv.Atoi(q.Get("index"))
		size, _ := strconv.Atoi(q.Get("pageSize"))
		var rows []string
		for i := index; i < min(index+size, len(counts)); i++ {
			rows = append(rows, fmt.Sprintf(`{"id":%s%d,"name":"m","downloadCount":%d}`, q.Get("classId"), i, counts[i]))
		}
		_, _ = fmt.Fprintf(w, `{"data":[%s],"pagination":{"totalCount":%d}}`, strings.Join(rows, ","), len(counts))
	}))
	t.Cleanup(srv.Close)
	return Driver{
		URL: srv.URL, Key: func() string { return "k" },
		GameSource: func(string) (components.GameSource, bool) {
			return components.GameSource{ID: "curseforge", Key: "1", GameID: 7, Classes: []string{"1", "2"}}, true
		},
	}
}

func ids(p source.Page) string {
	var out []string
	for _, it := range p.Items {
		out = append(out, it.ID+":"+strconv.Itoa(it.Downloads))
	}
	return strings.Join(out, " ")
}

func TestSearchMergesClassesBySortAndSumsTotals(t *testing.T) {
	t.Parallel()
	d := classFake(t, map[string][]int{"1": {90, 50, 10}, "2": {80, 70, 20}})
	page, err := d.Search(t.Context(), source.Query{Game: "g", Key: "1", Sort: source.SortDownloads})
	if err != nil || page.Total != 6 {
		t.Fatalf("%+v %v", page, err)
	}
	if got, want := ids(page), "10:90 20:80 21:70 11:50 22:20 12:10"; got != want {
		t.Fatalf("merge order %q, want %q", got, want)
	}
	if got := page.Items[0].URL; got != "https://www.curseforge.com/projects/10" {
		t.Fatalf("url %q", got)
	}
}

func TestSearchPagesAcrossClasses(t *testing.T) {
	t.Parallel()
	one, two := make([]int, 30), make([]int, 30)
	for i := range one {
		one[i], two[i] = 1000-2*i, 999-2*i
	}
	d := classFake(t, map[string][]int{"1": one, "2": two})
	page, err := d.Search(t.Context(), source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Page: source.FirstPage + 1})
	if err != nil || page.Total != 60 || len(page.Items) != source.PageSize {
		t.Fatalf("%+v %v", page, err)
	}
	first := page.Items[0].Downloads
	if want := 1000 - source.PageSize; first != want {
		t.Fatalf("page 2 starts at %d downloads, want %d", first, want)
	}
}
