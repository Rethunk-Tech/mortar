package browse

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

type fakeSource struct {
	got   source.Query
	calls int
}

func (*fakeSource) ID() string              { return "fake" }
func (*fakeSource) Name() string            { return "Fake" }
func (*fakeSource) Modes() []source.Acquire { return nil }
func (f *fakeSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	f.got = q
	f.calls++
	return source.Page{Total: 2, Items: []source.Item{{Source: "fake", ID: "1"}, {Source: "fake", ID: "2"}}}, nil
}

type pageOnly struct{}

func (pageOnly) ID() string              { return "pageonly" }
func (pageOnly) Name() string            { return "Page only" }
func (pageOnly) Modes() []source.Acquire { return nil }

var (
	fake    = &fakeSource{}
	_       = source.Register(fake)
	_       = source.Register(pageOnly{})
	catalog = components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "fake", Key: "gkey"}, {ID: "pageonly"}}}
)

func TestSearchDispatchesBySourceWithCatalogKey(t *testing.T) {
	c := &Client{Version: "v", Installed: func(src, id string) bool { return src == "fake" && id == "2" }}
	page, err := c.search(context.Background(), catalog, " FAKE ", "text", 0, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if want := (source.Query{Game: "g", Key: "gkey", Text: "text", Page: 1, Version: "v"}); fmt.Sprint(fake.got) != fmt.Sprint(want) {
		t.Fatalf("query %+v, want %+v", fake.got, want)
	}
	if page.Items[0].Installed || !page.Items[1].Installed {
		t.Fatalf("installed marking: %+v", page.Items)
	}
}

func TestSearchRefusesSourcesTheGameLacksOrCannotSearch(t *testing.T) {
	c := &Client{}
	for _, id := range []string{"nexus", "pageonly", "nope"} {
		if _, err := c.search(context.Background(), catalog, id, "x", 1, Filter{}); !errors.Is(err, ErrUnknownSource) {
			t.Fatalf("%s: got %v", id, err)
		}
	}
}

func TestSearchableSourcesFollowCatalogOrder(t *testing.T) {
	got := (&Service{}).SearchableSources("stardew")
	if len(got) != 3 || got[0].ID != "nexus" || got[0].Name != "Nexus Mods" || len(got[0].Sorts) == 0 || got[1].ID != "curseforge" || got[2].ID != "github" {
		t.Fatalf("got %v", got)
	}
	if got := (&Service{}).SearchableSources("nope"); got == nil || len(got) != 0 {
		t.Fatalf("unknown game: %v", got)
	}
}

type listSource struct {
	id    string
	total int
	items []string
	err   error
}

func (l listSource) ID() string            { return l.id }
func (l listSource) Name() string          { return l.id + " site" }
func (listSource) Modes() []source.Acquire { return nil }
func (l listSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	if l.err != nil {
		return source.Page{}, l.err
	}
	p := source.Page{Total: l.total}
	if q.Page > source.FirstPage {
		return p, nil
	}
	for _, id := range l.items {
		p.Items = append(p.Items, source.Item{Source: l.id, ID: id})
	}
	return p, nil
}

var (
	_ = source.Register(listSource{id: "alpha", total: 45, items: []string{"a1", "a2", "a3"}})
	_ = source.Register(listSource{id: "beta", total: 3, items: []string{"b1"}})
	_ = source.Register(listSource{id: "broken", err: errors.New("busy")})
)

func TestSearchAllInterleavesAndNamesFailedSources(t *testing.T) {
	game := components.GameInfo{ID: "m", Sources: []components.GameSource{{ID: "alpha"}, {ID: "beta"}, {ID: "broken"}}}
	page, err := (&Client{}).searchAll(context.Background(), game, "x", 1, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, it := range page.Items {
		ids = append(ids, it.ID)
	}
	if got := fmt.Sprint(ids); got != "[a1 b1 a2 a3]" {
		t.Fatalf("order %s", got)
	}
	if page.Total != 48 || page.Pages != 3 || fmt.Sprint(page.Failed) != "[broken]" {
		t.Fatalf("page %+v", page)
	}
	if _, err := (&Client{}).searchAll(context.Background(), components.GameInfo{ID: "z", Sources: []components.GameSource{{ID: "broken"}}}, "x", 1, Filter{}); err == nil {
		t.Fatal("all sources failing must be an error")
	}
	off, err := (&Client{Off: []string{"alpha", "broken"}}).searchAll(context.Background(), game, "x", 1, Filter{})
	if err != nil || len(off.Items) != 1 || off.Items[0].ID != "b1" || len(off.Failed) != 0 {
		t.Fatalf("sources turned off are still searched: %+v %v", off, err)
	}
}

type adultSource struct{}

func (adultSource) ID() string              { return "adultsrc" }
func (adultSource) Name() string            { return "Adult" }
func (adultSource) Modes() []source.Acquire { return nil }
func (adultSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	if q.Page > 1 {
		return source.Page{Total: 30}, nil
	}
	return source.Page{Total: 30, Items: []source.Item{{ID: "1"}, {ID: "2", Adult: true}, {ID: "3"}}}, nil
}

var _ = source.Register(adultSource{})

func TestAdultHitsAreDroppedUnlessOptedIn(t *testing.T) {
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "adultsrc"}}}
	for _, tc := range []struct {
		show       bool
		items, tot int
	}{{false, 2, 29}, {true, 3, 30}} {
		page, err := (&Client{ShowAdult: tc.show}).search(context.Background(), g, "adultsrc", "", 1, Filter{})
		if err != nil || len(page.Items) != tc.items || page.Total != tc.tot {
			t.Fatalf("show=%v: %+v %v", tc.show, page, err)
		}
	}
}

func TestEmptySearchListsTopModsAndIsCached(t *testing.T) {
	c := &Client{}
	fake.calls = 0
	topCache = map[string]topEntry{"expired": {}}
	for range 3 {
		if _, err := c.search(context.Background(), catalog, "fake", "", 1, Filter{}); err != nil {
			t.Fatal(err)
		}
	}
	if fake.calls != 1 {
		t.Fatalf("empty search hit the source %d times, want 1", fake.calls)
	}
	if _, kept := topCache["expired"]; kept {
		t.Fatal("an expired top answer survived an insert")
	}
	if _, err := c.search(context.Background(), catalog, "fake", "x", 1, Filter{}); err != nil || fake.calls != 2 {
		t.Fatalf("a text search must not be cached: calls %d err %v", fake.calls, err)
	}
}

type catSource struct{ got source.Query }

func (*catSource) ID() string              { return "catsrc" }
func (*catSource) Name() string            { return "Cat" }
func (*catSource) Modes() []source.Acquire { return nil }
func (c *catSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	c.got = q
	return source.Page{Total: 1, Items: []source.Item{{Source: "catsrc", ID: "c1"}}}, nil
}

func (*catSource) Categories(context.Context, string) ([]string, error) {
	return []string{"Maps", "audio"}, nil
}

var (
	cat = &catSource{}
	_   = source.Register(cat)
)

func TestFilterReachesSourcesAndIncludeSkipsSourcesWithoutCategories(t *testing.T) {
	game := components.GameInfo{ID: "cf", Sources: []components.GameSource{{ID: "alpha"}, {ID: "catsrc", Key: "k"}}}
	f := Filter{Include: []string{"Maps"}, Exclude: []string{"x"}, Sort: source.SortName}
	page, err := (&Client{}).searchAll(context.Background(), game, "q", 1, f)
	if err != nil || len(page.Items) != 1 || page.Items[0].Source != "catsrc" {
		t.Fatalf("include must leave only the categorized source: %+v %v", page, err)
	}
	if fmt.Sprintf("%v %v %s", cat.got.Categories, cat.got.ExcludeCategories, cat.got.Sort) != "[Maps] [x] name" {
		t.Fatalf("query %+v", cat.got)
	}
	page, err = (&Client{}).searchAll(context.Background(), game, "q", 1, Filter{Exclude: []string{"x"}})
	if err != nil || len(page.Items) < 2 {
		t.Fatalf("exclude-only keeps every source: %+v %v", page, err)
	}
}

type lockedCats struct{ catSource }

func (*lockedCats) ID() string { return "lockedcats" }
func (*lockedCats) Categories(context.Context, string) ([]string, error) {
	return nil, errors.New("keyring is locked")
}

func TestCategoriesSayWhyOnlyWhenNoSourceGaveAny(t *testing.T) {
	game := components.GameInfo{ID: "why", Sources: []components.GameSource{{ID: "lockedcats"}, {ID: "catsrc"}}}
	locked := &lockedCats{}
	names, err := categoriesOf(context.Background(), game, []source.Source{locked, cat})
	if err != nil || len(names) != 2 {
		t.Fatalf("one source answering is enough: %v %v", names, err)
	}
	if _, err := categoriesOf(context.Background(), game, []source.Source{locked}); err == nil || !strings.Contains(err.Error(), "keyring is locked") {
		t.Fatalf("err = %v, want the failing source's reason", err)
	}
}

type signedOutCats struct{ catSource }

func (*signedOutCats) ID() string { return "signedoutcats" }
func (*signedOutCats) Categories(context.Context, string) ([]string, error) {
	return nil, fmt.Errorf("listing: %w", source.ErrSignedOut)
}

func TestCategoriesOfASignedOutSourceAreNoneAndNoError(t *testing.T) {
	game := components.GameInfo{ID: "signedout", Sources: []components.GameSource{{ID: "signedoutcats"}}}
	names, err := categoriesOf(context.Background(), game, []source.Source{&signedOutCats{}})
	if err != nil || len(names) != 0 {
		t.Fatalf("signed out must read as no categories: %v %v", names, err)
	}
}

type gatedCats struct{ lockedCats }

func (*gatedCats) ID() string          { return "gatedcats" }
func (*gatedCats) Unavailable() string { return "this build has no API key" }

func TestCategoriesSkipASourceBrowseShowsAsUnavailable(t *testing.T) {
	game := components.GameInfo{ID: "gated", Sources: []components.GameSource{{ID: "gatedcats"}}}
	names, err := categoriesOf(context.Background(), game, []source.Source{&gatedCats{}})
	if err != nil || len(names) != 0 {
		t.Fatalf("an unavailable source was asked: %v %v", names, err)
	}
}

func TestCategoryHelpers(t *testing.T) {
	if !source.CategoryMatch([]string{"Maps"}, []string{"maps"}, nil) || source.CategoryMatch([]string{"Maps"}, nil, []string{"MAPS"}) {
		t.Fatal("CategoryMatch is case-insensitive include/exclude")
	}
	if got := fmt.Sprint(source.UniqueNames([]string{"b", "A", "a", " "})); got != "[A b]" {
		t.Fatalf("got %s", got)
	}
}

type bigSource struct{}

func (bigSource) ID() string              { return "bigsrc" }
func (bigSource) Name() string            { return "Big" }
func (bigSource) Modes() []source.Acquire { return nil }
func (bigSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	p := source.Page{Total: 200}
	for i := (q.Page - 1) * source.PageSize; i < q.Page*source.PageSize; i++ {
		p.Items = append(p.Items, source.Item{Source: "bigsrc", ID: fmt.Sprint(i)})
	}
	return p, nil
}

var _ = source.Register(bigSource{})

func TestHiddenHitsDoNotShortenAPage(t *testing.T) {
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "bigsrc"}}}
	odd := func(id string) bool {
		n, _ := strconv.Atoi(id)
		return n%2 != 0
	}
	for name, tc := range map[string]struct {
		c *Client
		f Filter
	}{
		"installed hidden by the filter":       {&Client{ShowAdult: true, Installed: func(_, id string) bool { return odd(id) }}, Filter{Installed: ModeHide}},
		"hidden mods under the default filter": {&Client{Hidden: func(it Item) bool { return odd(it.ID) }}, Filter{}},
	} {
		topCache = map[string]topEntry{}
		var ids []string
		for p := 1; p <= 3; p++ {
			page, err := tc.c.search(context.Background(), g, "bigsrc", "x", p, tc.f)
			if err != nil || len(page.Items) != source.PageSize {
				t.Fatalf("%s, page %d: %d items, %+v, %v", name, p, len(page.Items), page, err)
			}
			for _, it := range page.Items {
				ids = append(ids, it.ID)
			}
		}
		for i, id := range ids {
			if id != fmt.Sprint(i*2) {
				t.Fatalf("%s: pages must continue without gaps or repeats: %v", name, ids)
			}
		}
	}
}

type partialSource struct{ id string }

func (p partialSource) ID() string            { return p.id }
func (p partialSource) Name() string          { return p.id }
func (partialSource) Modes() []source.Acquire { return nil }
func (p partialSource) Search(context.Context, source.Query) (source.Page, error) {
	return source.Page{Total: 1, Items: []source.Item{{Source: p.id, ID: "x"}}, Failed: []string{"Partial site (Create a Sim)"}}, nil
}

var _ = source.Register(partialSource{id: "partial"})

func TestAPartlyFailedSourceNamesItsFailedPartsInEverySearchShape(t *testing.T) {
	one := components.GameInfo{ID: "m", Sources: []components.GameSource{{ID: "partial"}}}
	for name, f := range map[string]Filter{"plain": {}, "filtered": {Installed: ModeHide}} {
		page, err := (&Client{}).search(context.Background(), one, "partial", "x", 1, f)
		if err != nil || fmt.Sprint(page.Failed) != "[Partial site (Create a Sim)]" {
			t.Fatalf("%s: %+v %v", name, page, err)
		}
	}
	all := components.GameInfo{ID: "m", Sources: []components.GameSource{{ID: "partial"}, {ID: "broken"}}}
	page, err := (&Client{}).searchAll(context.Background(), all, "x", 1, Filter{})
	if err != nil || fmt.Sprint(page.Failed) != "[Partial site (Create a Sim) broken]" {
		t.Fatalf("all: %+v %v", page, err)
	}
	topCache = map[string]topEntry{}
	topPage, err := searchCached(context.Background(), "partial", partialSource{id: "partial"}, source.Query{Game: "m", Key: "k", Page: 1})
	if err != nil || len(topPage.Failed) != 1 {
		t.Fatalf("%+v %v", topPage, err)
	}
	if len(topCache) != 0 {
		t.Fatal("a partial answer was cached")
	}
}

func TestHiddenHitsAreDroppedAndNotCounted(t *testing.T) {
	c := &Client{Hidden: func(it Item) bool { return it.Source == "fake" && it.ID == "1" }}
	page, err := c.search(context.Background(), catalog, "fake", "", 1, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "2" || page.Total != 1 {
		t.Fatalf("page %+v", page)
	}
}

func TestNoisyNamesKeepsModpacksTranslationsAndCheats(t *testing.T) {
	names := []string{"Mods", "Modpacks", "Translations", "Language", "Cheats and God items", "Hacks", "Tools", "Modding Tools"}
	want := []string{"Modpacks", "Translations", "Language", "Cheats and God items", "Hacks"}
	if got := noisyNames(names); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

type twentyOneSource struct{}

func (twentyOneSource) ID() string              { return "src21" }
func (twentyOneSource) Name() string            { return "21" }
func (twentyOneSource) Modes() []source.Acquire { return nil }
func (twentyOneSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	p := source.Page{Total: 21}
	for i := (q.Page - 1) * source.PageSize; i < min(q.Page*source.PageSize, 21); i++ {
		p.Items = append(p.Items, source.Item{Source: "src21", ID: fmt.Sprint(i)})
	}
	return p, nil
}

var _ = source.Register(twentyOneSource{})

func TestHitsDroppedFromPageOneDoNotEndTheSearchBeforeTheLastUpstreamPage(t *testing.T) {
	topCache = map[string]topEntry{}
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "src21"}}}
	c := &Client{Hidden: func(it Item) bool { return it.ID == "0" }}
	page, err := c.search(context.Background(), g, "src21", "x", 1, Filter{})
	if err != nil || len(page.Items) != 20 || page.Items[19].ID != "20" {
		t.Fatalf("the 21st upstream hit must fill page 1: %d items, %v", len(page.Items), err)
	}
}

type excludingSource struct{}

func (excludingSource) ID() string              { return "excluding" }
func (excludingSource) Name() string            { return "Excluding" }
func (excludingSource) Modes() []source.Acquire { return nil }
func (excludingSource) Search(context.Context, source.Query) (source.Page, error) {
	return source.Page{Total: 3, Hidden: 2, Items: []source.Item{{Source: "excluding", ID: "1"}}}, nil
}

var _ = source.Register(excludingSource{})

func TestHitsASourceLeftOutForExcludedCategoriesStayCountedAsHidden(t *testing.T) {
	topCache = map[string]topEntry{}
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "excluding"}}}
	page, err := (&Client{}).search(context.Background(), g, "excluding", "", 1, Filter{Exclude: []string{"Maps"}})
	if err != nil || page.Hidden != 2 {
		t.Fatalf("hidden = %d, %v", page.Hidden, err)
	}
}

// numberedSource has `real` hits numbered from 0; total is what each page reports, or real when nil.
type numberedSource struct {
	id    string
	real  int
	total func(q source.Query, served int) int
}

func (n numberedSource) ID() string            { return n.id }
func (n numberedSource) Name() string          { return n.id }
func (numberedSource) Modes() []source.Acquire { return nil }
func (n numberedSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	end := min(q.Page*q.Len(), n.real)
	p := source.Page{Total: n.real}
	if n.total != nil {
		p.Total = n.total(q, end)
	}
	for i := (q.Page - 1) * q.Len(); i < end; i++ {
		p.Items = append(p.Items, source.Item{Source: n.id, ID: fmt.Sprint(i)})
	}
	return p, nil
}

var (
	_ = source.Register(numberedSource{id: "deep", real: 40000})
	// Shaped like the Mod DB's API, which gives no total: a full page reports one more hit than it shows.
	_ = source.Register(numberedSource{id: "limitonly", real: 1000, total: func(q source.Query, served int) int {
		if served == q.Page*q.Len() {
			return served + 1
		}
		return served
	}})
)

func freshBrowse() {
	topCache = map[string]topEntry{}
	cursors = map[string]cursorEntry{}
}

func TestPagesPastTheScanLimitOfABigSourceAreFull(t *testing.T) {
	freshBrowse()
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "deep"}}}
	c := &Client{Hidden: func(it Item) bool { return it.ID == "3" }}
	for p := 1; p <= 12; p++ {
		page, err := c.search(context.Background(), g, "deep", "", p, Filter{})
		if err != nil || len(page.Items) != source.PageSize {
			t.Fatalf("page %d: %d items, %v", p, len(page.Items), err)
		}
		// One hit is hidden, so page p starts one past its upstream place.
		if want := fmt.Sprint((p-1)*source.PageSize + 1); p > 1 && page.Items[0].ID != want {
			t.Fatalf("page %d starts at %s, want %s", p, page.Items[0].ID, want)
		}
	}
	page, err := c.search(context.Background(), g, "deep", "", 2000, Filter{})
	if err != nil || len(page.Items) != source.PageSize || page.Total != 40000 {
		t.Fatalf("page 2000: %d items, total %d, %v", len(page.Items), page.Total, err)
	}
}

func TestALimitOnlySourceOffersPageThree(t *testing.T) {
	freshBrowse()
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "limitonly"}}}
	page, err := (&Client{}).search(context.Background(), g, "limitonly", "", 2, Filter{})
	if err != nil || page.Total <= 2*source.PageSize {
		t.Fatalf("page 2 total %d offers no page 3, %v", page.Total, err)
	}
	page, err = (&Client{}).search(context.Background(), g, "limitonly", "", 3, Filter{})
	if err != nil || len(page.Items) != source.PageSize || page.Items[0].ID != "40" {
		t.Fatalf("page 3: %+v, %v", page.Items, err)
	}
}

func TestAFixedTotalSourceCountsTheSamePagesOnEveryPage(t *testing.T) {
	freshBrowse()
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "bigsrc"}}}
	c := &Client{Hidden: func(it Item) bool { n, _ := strconv.Atoi(it.ID); return n%2 != 0 }}
	one, err1 := c.search(context.Background(), g, "bigsrc", "", 1, Filter{})
	three, err3 := c.search(context.Background(), g, "bigsrc", "", 3, Filter{})
	if err1 != nil || err3 != nil || one.Total != three.Total {
		t.Fatalf("totals %d and %d, %v %v", one.Total, three.Total, err1, err3)
	}
}

func TestTheSourcePageCacheKeepsAtMostItsCap(t *testing.T) {
	freshBrowse()
	s := numberedSource{id: "deep", real: 40000}
	for p := 1; p <= topCap+10; p++ {
		if _, err := searchCached(context.Background(), "deep", s, source.Query{Game: "g", Page: p}); err != nil {
			t.Fatal(err)
		}
	}
	if len(topCache) != topCap {
		t.Fatalf("%d pages kept, cap %d", len(topCache), topCap)
	}
	if _, kept := topCache[fmt.Sprintf("%s|%d", queryKey("deep", source.Query{Game: "g"}), topCap+10)]; !kept {
		t.Fatal("the newest page was evicted")
	}
}

type hugeSource struct{}

func (hugeSource) ID() string              { return "hugesrc" }
func (hugeSource) Name() string            { return "Huge" }
func (hugeSource) Modes() []source.Acquire { return nil }
func (hugeSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	p := source.Page{Total: 1000}
	for i := (q.Page - 1) * source.PageSize; i < q.Page*source.PageSize; i++ {
		p.Items = append(p.Items, source.Item{Source: "hugesrc", ID: fmt.Sprint(i)})
	}
	return p, nil
}

var _ = source.Register(hugeSource{})

// The window asks for a screen's source pages at once, so page 12 can arrive before page 11 has remembered its end:
// it starts from the nearest earlier end, so no hit dropped before it is served twice or skipped.
func TestAPageAskedBeforeThePageBeforeItStartsWhereThatPageEnds(t *testing.T) {
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "hugesrc"}}}
	topCache = map[string]topEntry{}
	cursors = map[string]cursorEntry{}
	c := &Client{ShowAdult: true, Hidden: func(it Item) bool { return it.ID == "0" }}
	pageIDs := func(p int) []string {
		page, err := c.search(context.Background(), g, "hugesrc", "", p, Filter{})
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, it := range page.Items {
			ids = append(ids, it.ID)
		}
		return ids
	}
	for p := 1; p <= 9; p++ {
		pageIDs(p)
	}
	ten, eleven := pageIDs(10), pageIDs(12)
	if ten[0] != "181" || eleven[0] != "221" {
		t.Fatalf("page 10 starts %s, page 12 starts %s; want 181 and 221", ten[0], eleven[0])
	}
	if first := pageIDs(11)[0]; first != "201" {
		t.Fatalf("page 11 starts %s, want 201", first)
	}
}

func TestSourcesOffNeverTurnsEveryAvailableSourceOff(t *testing.T) {
	if got := SourcesOff("stardew", []string{"github"}); fmt.Sprint(got) != "[github]" {
		t.Fatalf("one off: %v", got)
	}
	if got := SourcesOff("stardew", []string{"nexus", "curseforge", "github"}); got != nil {
		t.Fatalf("all off must turn none off: %v", got)
	}
}

func TestForgetMakesTheNextSearchReadTheSourceAgain(t *testing.T) {
	c := &Client{}
	for range 2 {
		if _, err := c.search(context.Background(), catalog, "fake", "again", 1, Filter{}); err != nil {
			t.Fatal(err)
		}
	}
	before := fake.calls
	Forget("fake")
	if _, err := c.search(context.Background(), catalog, "fake", "again", 1, Filter{}); err != nil || fake.calls != before+1 {
		t.Fatalf("calls %d after forgetting, want %d: %v", fake.calls, before+1, err)
	}
}

type refreshable struct {
	fakeSource
	keys []string
}

func (*refreshable) ID() string { return "refreshable" }
func (r *refreshable) Refresh(key, _ string) {
	r.keys = append(r.keys, key)
}

func TestRefreshDropsTheGamesPagesAndRebuildsStoredListings(t *testing.T) {
	r := &refreshable{}
	source.Register(r)
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "fake", Key: "gkey"}, {ID: "refreshable", Key: "rkey"}}}
	c := &Client{}
	if _, err := c.search(context.Background(), g, "fake", "refresh", 1, Filter{}); err != nil {
		t.Fatal(err)
	}
	before := fake.calls
	forgetGame("other")
	if _, err := c.search(context.Background(), g, "fake", "refresh", 1, Filter{}); err != nil || fake.calls != before {
		t.Fatalf("another game's refresh dropped this game's page: %d calls, want %d", fake.calls, before)
	}
	refreshGame(g, "v")
	if _, err := c.search(context.Background(), g, "fake", "refresh", 1, Filter{}); err != nil || fake.calls != before+1 {
		t.Fatalf("calls %d after the game's refresh, want %d: %v", fake.calls, before+1, err)
	}
	if fmt.Sprint(r.keys) != "[rkey]" {
		t.Fatalf("stored listings rebuilt for %v, want [rkey]", r.keys)
	}
}
