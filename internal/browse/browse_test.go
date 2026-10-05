package browse

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

type fakeSource struct{ got source.Query }

func (*fakeSource) ID() string              { return "fake" }
func (*fakeSource) Name() string            { return "Fake" }
func (*fakeSource) Modes() []source.Acquire { return nil }
func (f *fakeSource) Search(_ context.Context, q source.Query) (source.Page, error) {
	f.got = q
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
	page, err := c.search(context.Background(), catalog, " FAKE ", "text", 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := (source.Query{Game: "g", Key: "gkey", Text: "text", Page: 1, Version: "v"}); fake.got != want {
		t.Fatalf("query %+v, want %+v", fake.got, want)
	}
	if page.Items[0].Installed || !page.Items[1].Installed {
		t.Fatalf("installed marking: %+v", page.Items)
	}
}

func TestSearchRefusesSourcesTheGameLacksOrCannotSearch(t *testing.T) {
	c := &Client{}
	for _, id := range []string{"nexus", "pageonly", "nope"} {
		if _, err := c.search(context.Background(), catalog, id, "x", 1); !errors.Is(err, ErrUnknownSource) {
			t.Fatalf("%s: got %v", id, err)
		}
	}
}

func TestSearchableSourcesFollowCatalogOrder(t *testing.T) {
	got := (&Service{}).SearchableSources("stardew")
	if len(got) != 2 || got[0] != (SourceInfo{ID: "nexus", Name: "Nexus Mods"}) || got[1].ID != "github" {
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
func (l listSource) Search(context.Context, source.Query) (source.Page, error) {
	if l.err != nil {
		return source.Page{}, l.err
	}
	p := source.Page{Total: l.total}
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
	page, err := (&Client{}).searchAll(context.Background(), game, "x", 1)
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
	if page.Total != 48 || page.Pages != 3 || fmt.Sprint(page.Failed) != "[broken site]" {
		t.Fatalf("page %+v", page)
	}
	if _, err := (&Client{}).searchAll(context.Background(), components.GameInfo{ID: "z", Sources: []components.GameSource{{ID: "broken"}}}, "x", 1); err == nil {
		t.Fatal("all sources failing must be an error")
	}
}

type adultSource struct{}

func (adultSource) ID() string              { return "adultsrc" }
func (adultSource) Name() string            { return "Adult" }
func (adultSource) Modes() []source.Acquire { return nil }
func (adultSource) Search(context.Context, source.Query) (source.Page, error) {
	return source.Page{Total: 30, Items: []source.Item{{ID: "1"}, {ID: "2", Adult: true}, {ID: "3"}}}, nil
}

var _ = source.Register(adultSource{})

func TestAdultHitsAreDroppedUnlessOptedIn(t *testing.T) {
	g := components.GameInfo{ID: "g", Sources: []components.GameSource{{ID: "adultsrc"}}}
	for _, tc := range []struct {
		show       bool
		items, tot int
	}{{false, 2, 29}, {true, 3, 30}} {
		page, err := (&Client{ShowAdult: tc.show}).search(context.Background(), g, "adultsrc", "", 1)
		if err != nil || len(page.Items) != tc.items || page.Total != tc.tot {
			t.Fatalf("show=%v: %+v %v", tc.show, page, err)
		}
	}
}
