package profile

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCleanEntryNoteTags(t *testing.T) {
	t.Parallel()
	note, tags, err := CleanEntryNoteTags("  hello  ", []string{" Farm ", "farm", "", "Crops"})
	if err != nil {
		t.Fatal(err)
	}
	if note != "hello" {
		t.Fatalf("note = %q", note)
	}
	if len(tags) != 2 || tags[0] != "Farm" || tags[1] != "Crops" {
		t.Fatalf("tags = %#v", tags)
	}
	if _, _, err := CleanEntryNoteTags(strings.Repeat("n", MaxEntryNote+1), nil); err == nil {
		t.Fatal("overlong note accepted")
	}
	if n := utf8.RuneCountInString(strings.Repeat("é", MaxEntryNote)); n != MaxEntryNote {
		t.Fatal("rune fixture")
	}
	if _, _, err := CleanEntryNoteTags(strings.Repeat("é", MaxEntryNote), nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CleanEntryNoteTags("", []string{strings.Repeat("t", MaxEntryTag+1)}); err == nil {
		t.Fatal("overlong tag accepted")
	}
	tooMany := make([]string, MaxEntryTags+1)
	for i := range tooMany {
		tooMany[i] = string(rune('a' + i))
	}
	if _, _, err := CleanEntryNoteTags("", tooMany); err == nil {
		t.Fatal("too many tags accepted")
	}
}

func TestSetEntryNoteTags(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := pinTestProfile(t, s)
	p, err := s.SetEntryNoteTags("stardew", p.ID, "nexus-1-1", "  keep  ", []string{"QoL", " qol "})
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].Note != "keep" || len(p.Entries[0].Tags) != 1 || p.Entries[0].Tags[0] != "QoL" {
		t.Fatalf("entry = %+v", p.Entries[0])
	}
	loaded, err := s.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Entries[0].Note != "keep" || loaded.Entries[0].Tags[0] != "QoL" {
		t.Fatalf("persisted = %+v", loaded.Entries[0])
	}
	if _, err := s.SetEntryNoteTags("stardew", p.ID, "missing", "x", nil); err == nil {
		t.Fatal("missing key must fail")
	}
	if _, err := s.SetEntryNoteTags("stardew", p.ID, "nexus-1-1", strings.Repeat("n", MaxEntryNote+1), nil); err == nil {
		t.Fatal("overlong note must fail")
	}
	p, err = s.SetEntryNoteTags("stardew", p.ID, "nexus-1-1", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].Note != "" || p.Entries[0].Tags != nil {
		t.Fatalf("cleared = %+v", p.Entries[0])
	}
}

func TestTagEditsRecordWhatChanged(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := pinTestProfile(t, s)
	name := entryName(p.Entries[0])
	for _, c := range []struct {
		tags   []string
		change HistoryChange
		detail string
	}{
		{[]string{"QoL"}, ChangeTagged, "QoL"},
		{[]string{"QoL", "core"}, ChangeTagged, "core"},
		{[]string{"core"}, ChangeUntagged, "QoL"},
		{[]string{"art"}, ChangeTags, ""},
	} {
		if _, err := s.SetEntryNoteTags("stardew", p.ID, "nexus-1-1", "", c.tags); err != nil {
			t.Fatal(err)
		}
		events, err := s.History("stardew", p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if ev := events[0]; ev.Change != c.change || ev.Detail != c.detail || ev.Name != name {
			t.Fatalf("tags %v: event %+v", c.tags, ev)
		}
	}
}
