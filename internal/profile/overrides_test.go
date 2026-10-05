package profile

import (
	"testing"
)

func overrideStore(t *testing.T) *Store {
	t.Helper()
	return newStore(t)
}

func TestSetOverridePersists(t *testing.T) {
	t.Parallel()
	s := overrideStore(t)
	p := mustCreate(t, s, "Farm")
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

func TestSetSkipPlayCheckWritesOverride(t *testing.T) {
	t.Parallel()
	s := overrideStore(t)
	p := mustCreate(t, s, "Farm")
	got, err := s.SetSkipPlayCheck("stardew", p.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Overrides["skipPlayCheck"] != "true" {
		t.Fatalf("overrides %#v", got.Overrides)
	}
	if got.PrefOverrides()["skipPlayCheck"] != "true" {
		t.Fatalf("pref overrides = %#v", got.PrefOverrides())
	}
}

func TestSetOverrideRejectsUnknownKey(t *testing.T) {
	t.Parallel()
	s := overrideStore(t)
	p := mustCreate(t, s, "Farm")
	if _, err := s.SetOverride("stardew", p.ID, "runsKept", "3"); err == nil {
		t.Fatal("expected error")
	}
}
