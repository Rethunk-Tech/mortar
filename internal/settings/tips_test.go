package settings

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestTipsSeen(t *testing.T) {
	s, dir := open(t)
	if got := s.Get().TipsSeen; got != nil {
		t.Fatalf("default = %v", got)
	}
	if _, err := s.Update(func(v *Settings) { v.TipsSeen = []string{"mods", "nope"} }); err == nil {
		t.Fatal("expected unknown tip error")
	}
	if s.Get().TipsSeen != nil {
		t.Fatal("state changed on rejected set")
	}
	next := []string{"mods", "share"}
	if _, err := s.Update(func(v *Settings) { v.TipsSeen = next }); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().TipsSeen; !slices.Equal(got, next) {
		t.Fatalf("stored = %v", got)
	}
	if _, err := s.Update(func(v *Settings) { v.TipsSeen = nil }); err != nil {
		t.Fatal(err)
	}
	if s.Get().TipsSeen != nil {
		t.Fatal("clear failed")
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`{"tipsSeen":["mods","nope","saves","mods"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get().TipsSeen; !slices.Equal(got, []string{"mods", "saves"}) {
		t.Fatalf("load = %v", got)
	}
}
