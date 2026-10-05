package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

type fakeThunderstore struct{ items []source.Item }

func (fakeThunderstore) ID() string              { return "thunderstore" }
func (fakeThunderstore) Name() string            { return "Thunderstore" }
func (fakeThunderstore) Modes() []source.Acquire { return nil }
func (f fakeThunderstore) Search(context.Context, source.Query) (source.Page, error) {
	return source.Page{Items: f.items}, nil
}

func TestThunderstoreUpdatesOfferNewerVersionsAndMarkSourceSwitches(t *testing.T) {
	source.Register(fakeThunderstore{items: []source.Item{
		{ID: "Alice-Cool", Name: "Cool", Author: "Alice", Version: "2.0.0", Repo: "alice/cool", URL: "https://t/cool"},
	}})
	t.Cleanup(func() { source.Register(thunderstore.Driver{}) })
	s := &Service{}
	same := framework.Mod{Key: "a", SourceKind: profile.KindThunderstore, SourceName: "Alice-Cool", SourceVersion: "1.0.0"}
	same.Name, same.Version = "Cool", "1.0.0"
	viaRepo := framework.Mod{Key: "b", SourceKind: profile.KindGitHub, SourceRepo: "Alice/Cool"}
	viaRepo.Name, viaRepo.Version = "Other", "1.0.0"
	current := framework.Mod{Key: "c", SourceKind: profile.KindThunderstore, SourceName: "Alice-Cool", SourceVersion: "2.0.0"}
	current.Name, current.Version = "Cool", "2.0.0"
	got := s.thunderstoreUpdates(context.Background(), "lethal-company", []framework.Mod{same, viaRepo, current}, nil)
	if len(got) != 2 {
		t.Fatalf("updates = %+v", got)
	}
	if got[0].Key != "a" || got[0].Switch || got[0].Package != "Alice-Cool" || got[0].Source != "Thunderstore" {
		t.Errorf("same-source update = %+v", got[0])
	}
	if got[1].Key != "b" || !got[1].Switch {
		t.Errorf("cross-source update = %+v", got[1])
	}
	covered := s.thunderstoreUpdates(context.Background(), "lethal-company", []framework.Mod{viaRepo}, []Update{{Key: "b", Version: "2.0.0"}})
	if len(covered) != 0 {
		t.Errorf("an update already offered was offered again: %+v", covered)
	}
}
