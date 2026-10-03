package profile

import (
	"reflect"
	"slices"
	"testing"
)

func TestRestoreEntriesReappliesEntryFields(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "Me.A", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetPinned("stardew", p.ID, "local-a", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetEntryNoteTags("stardew", p.ID, "local-a", "keep", []string{"farm"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetEntryCategory("stardew", p.ID, "local-a", "Crops"); err != nil {
		t.Fatal(err)
	}
	before, err := e.read("stardew", p.ID)
	if err != nil || len(before.Entries) != 1 {
		t.Fatalf("before = %+v, %v", before, err)
	}
	saved := before.Entries[0]
	if _, err := e.RemoveEntry("stardew", p.ID, "local-a"); err != nil {
		t.Fatal(err)
	}
	got, err := e.RestoreEntries("stardew", p.ID, []Entry{saved})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("entries = %+v", got.Entries)
	}
	restored := got.Entries[0]
	if restored.Key != saved.Key || !slices.Equal(restored.Disabled, saved.Disabled) || !restored.Pinned {
		t.Fatalf("restored = %+v, want key/disabled/pin from %+v", restored, saved)
	}
	if !slices.Equal(restored.Tags, saved.Tags) || restored.CategoryOverride != saved.CategoryOverride {
		t.Fatalf("tags/category = %+v, want %+v", restored, saved)
	}
}

func TestRestoreEntryFieldsWritesPreviousValues(t *testing.T) {
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetPinnedMany("stardew", p.ID, []string{"local-a"}, true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetEntryTagsMany("stardew", p.ID, []string{"local-b"}, "keep", true); err != nil {
		t.Fatal(err)
	}
	prev := []EntryFields{
		{Key: "local-a", Pinned: false, Tags: nil, CategoryOverride: ""},
		{Key: "local-b", Pinned: false, Tags: nil, CategoryOverride: "Crops"},
	}
	got, err := e.RestoreEntryFields("stardew", p.ID, prev)
	if err != nil {
		t.Fatal(err)
	}
	var a, b *Entry
	for i := range got.Entries {
		switch got.Entries[i].Key {
		case "local-a":
			a = &got.Entries[i]
		case "local-b":
			b = &got.Entries[i]
		}
	}
	if a == nil || b == nil || a.Pinned || b.CategoryOverride != "Crops" || len(b.Tags) != 0 {
		t.Fatalf("fields = %+v", got.Entries)
	}
	if !reflect.DeepEqual(namesForStoreKeys([]Entry{*a}, []string{"local-a"}), []string{entryLabel(*a)}) {
		t.Fatalf("names = %v", namesForStoreKeys([]Entry{*a}, []string{"local-a"}))
	}
}
