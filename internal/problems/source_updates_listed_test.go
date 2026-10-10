package problems

import (
	"context"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	nexussource "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// fakeByID is a Thunderstore whose search page is full of namesakes, as a popular name's is, and whose listing
// still knows the package by id.
type fakeByID struct {
	fakeThunderstore
	byID     map[string]source.Item
	searches *int
}

func (f fakeByID) Search(ctx context.Context, q source.Query) (source.Page, error) {
	*f.searches++
	return f.fakeThunderstore.Search(ctx, q)
}

func (f fakeByID) Items(_ context.Context, _, _ string, ids []string) (map[string]source.Item, error) {
	out := map[string]source.Item{}
	for _, id := range ids {
		if it, ok := f.byID[strings.ToLower(id)]; ok {
			out[strings.ToLower(id)] = it
		}
	}
	return out, nil
}

func TestAThunderstorePackageIsFoundByIDHoweverManyShareItsName(t *testing.T) {
	var namesakes []source.Item
	for range source.PageSize {
		namesakes = append(namesakes, source.Item{ID: "Other-Cool", Name: "Cool", Author: "Other", Version: "9.0.0"})
	}
	searches := 0
	source.Register(fakeThunderstore{id: "nexus"})
	source.Register(fakeByID{
		id: "thunderstore", items: namesakes,
		byID:     map[string]source.Item{"alice-cool": {ID: "Alice-Cool", Name: "Cool", Author: "Alice", Version: "2.0.0", URL: "https://t/cool"}},
		searches: &searches,
	})
	t.Cleanup(func() {
		source.Register(thunderstore.Driver{})
		source.Register(nexussource.Driver{})
	})
	installed := framework.Mod{Key: "a", SourceKind: profile.KindThunderstore, SourceName: "Alice-Cool", SourceVersion: "1.0.0"}
	installed.Name, installed.Version = "Cool", "1.0.0"
	got := (&Service{}).sourceUpdates(context.Background(), "lethal-company", []framework.Mod{installed}, nil)
	if len(got) != 1 || got[0].Package != "Alice-Cool" || got[0].Version != "2.0.0" {
		t.Fatalf("updates = %+v", got)
	}
	if searches != 0 {
		t.Fatalf("a package known by id was searched for %d times", searches)
	}
}
