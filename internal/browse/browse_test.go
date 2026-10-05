package browse

import (
	"context"
	"errors"
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
