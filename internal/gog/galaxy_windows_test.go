//go:build windows

package gog

import (
	"path/filepath"
	"testing"
)

func TestGalaxyRegistrySeam(t *testing.T) {
	dir := t.TempDir()
	writeGame(t, dir)
	lookupGalaxy = func() string { return dir }
	t.Cleanup(func() { lookupGalaxy = readGalaxyRegistry })
	got := Locate(t.TempDir(), Roots{})
	if len(got) == 0 || got[0].Dir != dir || got[0].Store != StoreGOG {
		t.Fatalf("galaxy = %#v", got)
	}
}

func TestWindowsOfflineProgramFiles(t *testing.T) {
	pf := t.TempDir()
	dir := filepath.Join(pf, "GOG Galaxy", "Games", "Stardew Valley")
	writeGame(t, dir)
	t.Setenv("ProgramFiles(x86)", pf)
	lookupGalaxy = func() string { return "" }
	t.Cleanup(func() { lookupGalaxy = readGalaxyRegistry })
	got := Locate(t.TempDir(), Roots{})
	found := false
	for _, in := range got {
		if in.Dir == dir && in.Store == StoreGOG {
			found = true
		}
	}
	if !found {
		t.Fatalf("offline program files missing: %#v", got)
	}
}
