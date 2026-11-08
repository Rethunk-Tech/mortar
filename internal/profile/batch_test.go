package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestBatchEntryOperationsWriteAllSelectedEntries(t *testing.T) {
	t.Parallel()
	s := newEnv(t)
	p := mustCreate(t, s, "Batch")
	var err error
	for _, key := range []string{"a-1", "b-1"} {
		s.item(t, key, map[string]string{"manifest.json": manifestJSON(key)})
		if _, err := s.AddEntry("stardew", p.ID, key, Source{}); err != nil {
			t.Fatal(err)
		}
	}

	if p, err = s.SetPinnedMany("stardew", p.ID, []string{"a-1", "b-1"}, true, ""); err != nil {
		t.Fatal(err)
	}
	if p, err = s.SetEntryTagsMany("stardew", p.ID, []string{"a-1", "b-1"}, "QoL", true); err != nil {
		t.Fatal(err)
	}
	if p, err = s.SetEntryCategoryMany("stardew", p.ID, []string{"a-1", "b-1"}, "Utilities"); err != nil {
		t.Fatal(err)
	}
	if p, err = s.SetSkipVersionMany("stardew", p.ID, []SkipVersionRef{
		{Key: "a-1", Version: "2.0"},
		{Key: "b-1", Version: "3.0"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, entry := range p.Entries {
		if !entry.Pinned || entry.CategoryOverride != "Utilities" || len(entry.Tags) != 1 || entry.Tags[0] != "QoL" {
			t.Fatalf("batch entry not updated: %+v", entry)
		}
	}
	if p.Entries[0].SkipVersion != "2.0" || p.Entries[1].SkipVersion != "3.0" {
		t.Fatalf("skip versions not updated: %+v", p.Entries)
	}

	if p, err = s.SetEntryTagsMany("stardew", p.ID, []string{"a-1", "b-1"}, "QoL", false); err != nil {
		t.Fatal(err)
	}
	for _, entry := range p.Entries {
		if len(entry.Tags) != 0 {
			t.Fatalf("tag was not removed: %+v", entry)
		}
	}
}

// Every API that names one entry reports a key it cannot find with the same words, through requireEntry.
func TestAMissingEntryIsReportedTheSameWay(t *testing.T) {
	t.Parallel()
	s := newEnv(t)
	p := mustCreate(t, s, "Missing")
	s.item(t, "a-1", map[string]string{"manifest.json": manifestJSON("a-1")})
	if _, err := s.AddEntry("stardew", p.ID, "a-1", Source{}); err != nil {
		t.Fatal(err)
	}
	const want = `"gone" is not in this profile`
	cases := map[string]func() error{
		"SetPinned": func() error { _, err := s.SetPinned("stardew", p.ID, "gone", true, ""); return err },
		"SetPinnedMany": func() error {
			_, err := s.SetPinnedMany("stardew", p.ID, []string{"a-1", "gone"}, true, "")
			return err
		},
		"SetSkipVersionMany": func() error {
			_, err := s.SetSkipVersionMany("stardew", p.ID, []SkipVersionRef{{Key: "gone"}})
			return err
		},
		"RestoreEntryFields": func() error {
			_, err := s.RestoreEntryFields("stardew", p.ID, []EntryFields{{Key: "gone"}})
			return err
		},
		"SetWinner":  func() error { _, err := s.SetWinner("stardew", p.ID, "gone", "a", "b", true); return err },
		"AddToGroup": func() error { _, err := s.AddToGroup("stardew", p.ID, "G", "gone"); return err },
	}
	for name, call := range cases {
		err := call()
		if err == nil {
			t.Errorf("%s: a missing entry was accepted", name)
			continue
		}
		if _, msg := usererr.Parse(err.Error()); msg != want {
			t.Errorf("%s: err = %q, want %q", name, msg, want)
		}
	}
}
