package problems

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func replacedService(t *testing.T) (*Service, string, string, string) {
	t.Helper()
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := profiles.ProfileDir("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(dir, "replaced", "BepInEx", "config", "x.cfg")
	if err := os.MkdirAll(filepath.Dir(kept), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kept, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	return NewService(t.TempDir(), set, profiles, nil), p.ID, dir, kept
}

func TestRestoreReplacedPutsTheFileBackAndForgetsTheCachedCheck(t *testing.T) {
	s, id, dir, kept := replacedService(t)
	s.cache["stale"] = cached{}

	if err := s.RestoreReplaced(t.Context(), "lethal-company", id, "BepInEx/config/x.cfg"); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "BepInEx", "config", "x.cfg")); err != nil || string(b) != "old" {
		t.Fatalf("restored file = %q, %v", b, err)
	}
	if _, err := os.Stat(kept); !os.IsNotExist(err) {
		t.Fatalf("the kept copy stayed: %v", err)
	}
	if len(s.cache) != 0 {
		t.Fatalf("the cached check survived: %v", s.cache)
	}
	if err := s.RestoreReplaced(t.Context(), "lethal-company", id, "BepInEx/config/none.cfg"); err == nil {
		t.Fatal("a path nothing replaced was restored")
	}
}

func TestDeleteReplacedRemovesEveryKeptCopyAndForgetsTheCachedCheck(t *testing.T) {
	s, id, _, kept := replacedService(t)
	s.cache["stale"] = cached{}

	if err := s.DeleteReplaced(t.Context(), "lethal-company", id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(kept); !os.IsNotExist(err) {
		t.Fatalf("the kept copy stayed: %v", err)
	}
	if len(s.cache) != 0 {
		t.Fatalf("the cached check survived: %v", s.cache)
	}
}
