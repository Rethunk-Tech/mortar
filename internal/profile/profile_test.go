package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return &Store{root: filepath.Join(t.TempDir(), "profiles")}
}

func TestCreateListRename(t *testing.T) {
	s := newStore(t)
	if got, err := s.List("stardew"); err != nil || len(got) != 0 {
		t.Fatalf("empty list = %v, %v", got, err)
	}
	a, err := s.Create("stardew", "  Cookie farm ")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create("stardew", "Second")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "Cookie farm" || b.Order <= a.Order {
		t.Fatalf("created = %+v, %+v", a, b)
	}
	if fi, err := os.Stat(filepath.Join(s.root, "stardew", a.ID, "mods")); err != nil || !fi.IsDir() {
		t.Fatalf("mods folder: %v", err)
	}
	renamed, err := s.Rename("stardew", a.ID, "Renamed")
	if err != nil || renamed.Name != "Renamed" {
		t.Fatalf("rename = %+v, %v", renamed, err)
	}
	got, err := s.List("stardew")
	if err != nil || len(got) != 2 || got[0].ID != a.ID || got[0].Name != "Renamed" || got[1].ID != b.ID {
		t.Fatalf("list = %+v, %v", got, err)
	}
	raw, _ := os.ReadFile(filepath.Join(s.root, "stardew", a.ID, fileName))
	if !strings.Contains(string(raw), `"entries": []`) {
		t.Fatalf("json: %s", raw)
	}
}

func TestListReturnsDamagedProfiles(t *testing.T) {
	s := newStore(t)
	good, err := s.Create("stardew", "Good")
	if err != nil {
		t.Fatal(err)
	}
	badID := "0123456789abcdef"
	badDir := filepath.Join(s.root, "stardew", badID)
	if err := os.MkdirAll(filepath.Join(badDir, "mods"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, fileName), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := s.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	var damaged Profile
	foundGood := false
	for _, p := range got {
		if p.ID == good.ID {
			foundGood = true
		}
		if p.ID == badID {
			damaged = p
		}
	}
	if !foundGood || damaged.ID != badID || damaged.Error == "" {
		t.Fatalf("profiles = %+v", got)
	}
}

func TestStoreKeysSkipsDamagedAndListDamaged(t *testing.T) {
	s := newStore(t)
	s.trash = filepath.Join(t.TempDir(), "trash")
	good, err := s.Create("stardew", "Good")
	if err != nil {
		t.Fatal(err)
	}
	badID := "0123456789abcdef"
	badDir := filepath.Join(s.root, "stardew", badID)
	if err := os.MkdirAll(filepath.Join(badDir, "mods"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, fileName), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	keys, err := s.StoreKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys["stardew"]) != 0 {
		t.Fatalf("StoreKeys used a damaged profile: %v", keys)
	}
	damaged, err := s.ListDamaged("stardew")
	if err != nil || len(damaged) != 1 || damaged[0].ID != badID || damaged[0].Error == "" {
		t.Fatalf("ListDamaged = %+v, %v", damaged, err)
	}
	all, err := s.List("stardew")
	if err != nil || len(all) != 2 {
		t.Fatalf("List = %+v, %v", all, err)
	}
	if good.ID == "" {
		t.Fatal("good profile missing")
	}
}

func TestRejects(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "ok")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "   ", strings.Repeat("x", 61)} {
		if _, err := s.Create("stardew", name); err == nil {
			t.Errorf("Create(%q) accepted", name)
		}
		if _, err := s.Rename("stardew", p.ID, name); err == nil {
			t.Errorf("Rename(%q) accepted", name)
		}
	}
	if _, err := s.Create("stardew", strings.Repeat("é", 60)); err != nil {
		t.Errorf("60 runes rejected: %v", err)
	}
	if _, err := s.Create("nope", "x"); err == nil {
		t.Error("unknown game accepted")
	}
	if _, err := s.List("../etc"); err == nil {
		t.Error("traversal game accepted")
	}
	for _, id := range []string{"..", "../stardew", "../../x", "", p.ID + "/.."} {
		if _, err := s.Rename("stardew", id, "x"); err == nil {
			t.Errorf("Rename id %q accepted", id)
		}
	}
}

func TestModFolderRejectsUnknown(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ModFolder("stardew", p.ID, "", "nope.Mod"); err == nil {
		t.Fatal("unknown mod accepted")
	}
}

func TestSetNotes(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.SetNotes("stardew", p.ID, "co-op friday")
	if err != nil || got.Notes != "co-op friday" {
		t.Fatalf("set = %+v, %v", got, err)
	}
	all, err := s.List("stardew")
	if err != nil || all[0].Notes != "co-op friday" {
		t.Fatalf("list = %+v, %v", all, err)
	}
	if _, err := s.SetNotes("stardew", p.ID, strings.Repeat("é", MaxNotes+1)); err == nil {
		t.Fatal("over-cap notes accepted")
	}
	if _, err := s.SetNotes("stardew", p.ID, strings.Repeat("é", MaxNotes)); err != nil {
		t.Fatalf("notes at the cap: %v", err)
	}
	if _, err := s.SetNotes("stardew", "nope", "x"); err == nil {
		t.Fatal("unknown profile accepted")
	}
}

func TestInModsHoldsTheStoreLock(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	err = s.InMods("stardew", p.ID, func(got Profile, modsDir string) error {
		if s.mu.TryLock() {
			t.Error("the store lock is free while fn runs")
		}
		if got.ID != p.ID || !filepath.IsAbs(modsDir) || filepath.Base(modsDir) != "mods" {
			t.Errorf("fn got %+v, %q", got, modsDir)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
