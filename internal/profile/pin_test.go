package profile

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
)

func pinTestProfile(t *testing.T, s *Store) Profile {
	t.Helper()
	if !game.Valid("stardew") {
		t.Fatal("stardew must be a valid game id")
	}
	p := mustCreate(t, s, "Pin test")
	var err error
	entry := Entry{Key: "nexus-1-1", Mods: []EntryMod{{UniqueID: "A.B", Version: "1.0.0"}}, Added: time.Now().UTC().Truncate(time.Second)}
	p, err = s.update("stardew", p.ID, func(cur *Profile, _ string) error {
		cur.Entries = append(cur.Entries, entry)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOffersUpdate(t *testing.T) {
	e := Entry{Key: "k"}
	if e.OffersUpdate("") {
		t.Fatal("empty newer is not an update")
	}
	if !e.OffersUpdate("2.0.0") {
		t.Fatal("unpinned entry offers a newer version")
	}
	e.Pinned = true
	if e.OffersUpdate("2.0.0") {
		t.Fatal("pinned entry offers no update")
	}
	e.Pinned = false
	e.SkipVersion = "2.0.0"
	if e.OffersUpdate("2.0.0") {
		t.Fatal("skipped version is hidden")
	}
	if !e.OffersUpdate("2.1.0") {
		t.Fatal("a later version is offered again")
	}
}

func TestSetPinnedAndSkipVersion(t *testing.T) {
	s := newStore(t)
	p := pinTestProfile(t, s)
	p, err := s.SetSkipVersion("stardew", p.ID, "nexus-1-1", "2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].SkipVersion != "2.0.0" {
		t.Fatalf("skip: %q", p.Entries[0].SkipVersion)
	}
	p, err = s.SetPinned("stardew", p.ID, "nexus-1-1", true, "stays on 1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Entries[0].Pinned {
		t.Fatal("expected pinned")
	}
	if p.Entries[0].SkipVersion != "" {
		t.Fatalf("pin should clear skip, got %q", p.Entries[0].SkipVersion)
	}
	if p.Entries[0].PinReason != "stays on 1.0" {
		t.Fatalf("pin reason: %q", p.Entries[0].PinReason)
	}
	p, err = s.SetPinned("stardew", p.ID, "nexus-1-1", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].Pinned {
		t.Fatal("expected unpinned")
	}
	p, err = s.SetSkipVersion("stardew", p.ID, "nexus-1-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].SkipVersion != "" {
		t.Fatalf("cleared skip: %q", p.Entries[0].SkipVersion)
	}
	loaded, err := s.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Entries[0].Pinned {
		t.Fatal("unpinned must persist")
	}
	if _, err := s.SetPinned("stardew", p.ID, "missing", true, ""); err == nil {
		t.Fatal("missing key must fail")
	}
}
