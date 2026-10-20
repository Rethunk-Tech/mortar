package browse

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// rankSource holds hits already in downloads order and pages them like a site's API; failFrom > 0 fails from that page.
type rankSource struct {
	id        string
	downloads []int
	failFrom  int
	sorts     []string
	mu        sync.Mutex
	pages     []int
}

func (r *rankSource) ID() string              { return r.id }
func (r *rankSource) Name() string            { return r.id }
func (r *rankSource) Modes() []source.Acquire { return nil }
func (r *rankSource) Sorts() []string         { return r.sorts }
func (r *rankSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	r.mu.Lock()
	r.pages = append(r.pages, q.Page)
	r.mu.Unlock()
	if r.failFrom > 0 && q.Page >= r.failFrom {
		return source.Page{}, errors.New("busy")
	}
	p := source.Page{Total: len(r.downloads), Items: []source.Item{}}
	from := (q.Page - source.FirstPage) * source.PageSize
	for i := from; i < min(from+source.PageSize, len(r.downloads)); i++ {
		p.Items = append(p.Items, source.Item{Source: r.id, ID: fmt.Sprintf("%s%d", r.id, i), Name: fmt.Sprintf("%s mod %d", r.id, i), Author: r.id, Downloads: r.downloads[i]})
	}
	return p, nil
}

func descending(from, n, step int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = from - i*step
	}
	return out
}

func rankGame(t *testing.T, sources ...*rankSource) components.GameInfo {
	t.Helper()
	topCache = map[string]topEntry{}
	game := components.GameInfo{ID: "ranked"}
	for _, s := range sources {
		if s.sorts == nil {
			s.sorts = []string{source.SortDownloads}
		}
		source.Register(s)
		game.Sources = append(game.Sources, components.GameSource{ID: s.id})
	}
	return game
}

func downloadsOf(items []Item) []int {
	out := make([]int, len(items))
	for i, it := range items {
		out[i] = it.Downloads
	}
	return out
}

func TestRankedPageOneIsTheTopHitsAcrossSources(t *testing.T) {
	a := &rankSource{id: "ra", downloads: descending(1000, 40, 1)}  // 1000..961
	b := &rankSource{id: "rb", downloads: descending(995, 40, 10)}  // 995, 985, ...
	weak := &rankSource{id: "rw", downloads: descending(50, 40, 1)} // never near the top
	game := rankGame(t, a, b, weak)
	page, err := (&Client{}).searchAll(context.Background(), game, "", 1, Filter{Sort: source.SortDownloads})
	if err != nil {
		t.Fatal(err)
	}
	got := downloadsOf(page.Items)
	want := []int{1000, 999, 998, 997, 996, 995, 995, 994, 993, 992, 991, 990, 989, 988, 987, 986, 985, 985, 984, 983}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("page 1 = %v\nwant     %v", got, want)
	}
	if page.Total != 120 || page.Pages != 6 {
		t.Fatalf("total %d pages %d", page.Total, page.Pages)
	}
	if slices.ContainsFunc(page.Items, func(it Item) bool { return it.Source == "rw" }) || len(weak.pages) != 1 {
		t.Fatalf("the weak source must contribute nothing and be read once: read %v", weak.pages)
	}
	for _, s := range []*rankSource{a, b, weak} {
		if len(s.pages) > 2 {
			t.Fatalf("%s read pages %v; page 1 needs at most 2 of each", s.id, s.pages)
		}
	}
	two, err := (&Client{}).searchAll(context.Background(), game, "", 2, Filter{Sort: source.SortDownloads})
	if err != nil || len(two.Items) != source.PageSize || two.Items[0].Downloads > page.Items[len(page.Items)-1].Downloads {
		t.Fatalf("page 2 must continue below page 1: %v %v", downloadsOf(two.Items), err)
	}
}

func TestRankedLeavesOutAndNamesSourcesThatCannotSort(t *testing.T) {
	a := &rankSource{id: "rs", downloads: descending(500, 5, 1)}
	stars := &rankSource{id: "rstars", downloads: descending(9000, 5, 1), sorts: []string{source.SortStars}}
	game := rankGame(t, a, stars)
	page, err := (&Client{}).searchAll(context.Background(), game, "", 1, Filter{Sort: source.SortDownloads})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(page.Unsorted) != "[rstars]" || len(stars.pages) != 0 || len(page.Items) != 5 {
		t.Fatalf("unsorted %v, stars read %v, items %d", page.Unsorted, stars.pages, len(page.Items))
	}
}

func TestRankedKeepsWhatAFailingSourceAlreadyReturned(t *testing.T) {
	steady := &rankSource{id: "rsteady", downloads: descending(100, 60, 1)}
	flaky := &rankSource{id: "rflaky", downloads: descending(1000, 60, 1), failFrom: 3}
	game := rankGame(t, steady, flaky)
	page, err := (&Client{}).searchAll(context.Background(), game, "", 3, Filter{Sort: source.SortDownloads})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(page.Failed) != "[rflaky]" {
		t.Fatalf("failed %v", page.Failed)
	}
	// rflaky's first two pages (40 hits, all above rsteady's best) fill pages 1 and 2; page 3 is rsteady's top.
	if len(page.Items) != source.PageSize || page.Items[0].Source != "rsteady" || page.Items[0].Downloads != 100 {
		t.Fatalf("page 3 = %v", downloadsOf(page.Items))
	}
}
