package profile

import "testing"

func TestBatchEntryOperationsWriteAllSelectedEntries(t *testing.T) {
	s := newEnv(t)
	p, err := s.Create("stardew", "Batch")
	if err != nil {
		t.Fatal(err)
	}
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
