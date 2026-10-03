package profile

import (
	"path/filepath"
	"testing"
)

func overrideStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	return &Store{root: filepath.Join(t.TempDir(), "profiles")}
}

func TestSetOverridePersists(t *testing.T) {
	s := overrideStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.SetOverride("stardew", p.ID, "defaultLaunchMethod", "direct")
	if err != nil {
		t.Fatal(err)
	}
	if got.Overrides["defaultLaunchMethod"] != "direct" {
		t.Fatalf("overrides = %#v", got.Overrides)
	}
	s.mu.Lock()
	read, err := s.read("stardew", p.ID)
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if read.Overrides["defaultLaunchMethod"] != "direct" {
		t.Fatalf("disk overrides = %#v", read.Overrides)
	}
	cleared, err := s.SetOverride("stardew", p.ID, "defaultLaunchMethod", "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared.Overrides) != 0 {
		t.Fatalf("cleared overrides = %#v", cleared.Overrides)
	}
}

func TestSetOverrideFoldsSkipPlayCheck(t *testing.T) {
	s := overrideStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.SetSkipPlayCheck("stardew", p.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.SkipPlayCheck || got.Overrides["skipPlayCheck"] != "true" {
		t.Fatalf("fold = skip %v overrides %#v", got.SkipPlayCheck, got.Overrides)
	}
	if got.PrefOverrides()["skipPlayCheck"] != "true" {
		t.Fatalf("pref overrides = %#v", got.PrefOverrides())
	}
}

func TestSetOverrideRejectsUnknownKey(t *testing.T) {
	s := overrideStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetOverride("stardew", p.ID, "runsKept", "3"); err == nil {
		t.Fatal("expected error")
	}
}
