package profile

import (
	"path/filepath"
	"testing"
)

func TestClearCollectionRecordsHistory(t *testing.T) {
	root := t.TempDir()
	s := &Store{root: filepath.Join(root, "profiles"), trash: filepath.Join(root, "trash")}
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	ref := CollectionRef{Domain: "stardewvalley", Slug: "cozy", Name: "Cozy", Revision: 2}
	if _, err := s.SetCollection("stardew", p.ID, ref); err != nil {
		t.Fatal(err)
	}
	got, err := s.ClearCollection("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Collection != nil {
		t.Fatalf("collection: %+v", got.Collection)
	}
	events, err := s.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Kind != historyCollectionUnlink {
		t.Fatalf("history: %+v", events)
	}
}
