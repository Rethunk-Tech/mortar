package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	nexussource "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

type fakeThunderstore struct {
	id    string
	items []source.Item
}

func (f fakeThunderstore) ID() string { return f.id }
func (f fakeThunderstore) Name() string {
	return map[string]string{"thunderstore": "Thunderstore"}[f.id]
}
func (fakeThunderstore) Modes() []source.Acquire { return nil }
func (f fakeThunderstore) Search(context.Context, source.Query) (source.Page, error) {
	return source.Page{Items: f.items}, nil
}

func TestThunderstoreUpdatesOfferNewerVersionsAndMarkSourceSwitches(t *testing.T) {
	// The game's other sources answer nothing, so the test never reaches the network.
	source.Register(fakeThunderstore{id: "nexus"})
	source.Register(fakeThunderstore{id: "thunderstore", items: []source.Item{
		{ID: "Alice-Cool", Name: "Cool", Author: "Alice", Version: "2.0.0", Repo: "alice/cool", URL: "https://t/cool"},
	}})
	t.Cleanup(func() {
		source.Register(thunderstore.Driver{})
		source.Register(nexussource.Driver{})
	})
	turns, nexusTurns := 0, 0
	s := &Service{Throttle: func(_ context.Context, src string) (func(), error) {
		turns++
		if src == profile.KindNexus {
			nexusTurns++
		}
		return func() {}, nil
	}}
	same := framework.Mod{Key: "a", SourceKind: profile.KindThunderstore, SourceName: "Alice-Cool", SourceVersion: "1.0.0"}
	same.Name, same.Version = "Cool", "1.0.0"
	viaRepo := framework.Mod{Key: "b", SourceKind: profile.KindGitHub, SourceRepo: "Alice/Cool"}
	viaRepo.Name, viaRepo.Version = "Other", "1.0.0"
	current := framework.Mod{Key: "c", SourceKind: profile.KindThunderstore, SourceName: "Alice-Cool", SourceVersion: "2.0.0"}
	current.Name, current.Version = "Cool", "2.0.0"
	fromNexus := framework.Mod{Key: "d", SourceKind: profile.KindNexus}
	fromNexus.Name, fromNexus.Version = "Nexus Mod", "1.0.0"
	got := s.sourceUpdates(context.Background(), "lethal-company", []framework.Mod{same, viaRepo, current, fromNexus}, nil)
	if len(got) != 2 {
		t.Fatalf("updates = %+v", got)
	}
	if got[0].Key != "a" || got[0].Switch || got[0].Package != "Alice-Cool" || got[0].Source != "Thunderstore" {
		t.Errorf("same-source update = %+v", got[0])
	}
	if got[1].Key != "b" || !got[1].Switch {
		t.Errorf("cross-source update = %+v", got[1])
	}
	if turns == 0 {
		t.Error("searches did not wait for their source's turn")
	}
	if nexusTurns != 0 {
		t.Errorf("Nexus was searched %d times; its updates come from update keys", nexusTurns)
	}
	covered := s.sourceUpdates(context.Background(), "lethal-company", []framework.Mod{viaRepo}, []Update{{Key: "b", Version: "2.0.0"}})
	if len(covered) != 0 {
		t.Errorf("an update already offered was offered again: %+v", covered)
	}
}
