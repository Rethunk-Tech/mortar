package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveCustomCategoriesAndDeleteClearsOverrides(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "farm")
	var err error
	p, err = s.update("stardew", p.ID, func(prof *Profile, _ string) error {
		prof.Entries = []Entry{{Key: "k", Source: Source{Kind: KindLocal, Name: "a.zip"}, Mods: []Component{{ID: "smapi:A.Mod", Name: "A", Folder: "."}}}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	saved, err := s.SaveCustomCategories("stardew", []CustomCategory{{Name: "QoL", Color: "teal"}})
	if err != nil || len(saved) != 1 || saved[0].ID == "" {
		t.Fatalf("save = %+v, %v", saved, err)
	}
	id := saved[0].ID
	path := filepath.Join(s.dataDir, categoriesDir, "stardew.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file: %v", err)
	}
	p, err = s.SetEntryCategory("stardew", p.ID, "k", id)
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].CategoryOverride != id {
		t.Fatalf("override = %q", p.Entries[0].CategoryOverride)
	}
	if _, err := s.SaveCustomCategories("stardew", nil); err != nil {
		t.Fatal(err)
	}
	p, err = s.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].CategoryOverride != "" {
		t.Fatalf("expected cleared override, got %q", p.Entries[0].CategoryOverride)
	}
}

func TestSetEntryCategoryNexusName(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "farm")
	var err error
	p, err = s.update("stardew", p.ID, func(prof *Profile, _ string) error {
		prof.Entries = []Entry{{Key: "k", Source: Source{Kind: KindLocal, Name: "a.zip"}, Mods: []Component{{ID: "smapi:A.Mod", Name: "A", Folder: "."}}}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.SetEntryCategory("stardew", p.ID, "k", "User Interface")
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].CategoryOverride != "User Interface" {
		t.Fatalf("got %q", p.Entries[0].CategoryOverride)
	}
	if _, err := s.SetEntryCategory("stardew", p.ID, "k", strings.Repeat("x", maxCategoryName+1)); err == nil {
		t.Fatal("long name accepted")
	}
}
