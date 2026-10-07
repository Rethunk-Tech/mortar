package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestScratchCopyIsNeverListedAndIsPurged(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("X.A")})
	src := mustCreate(t, e, "Main")
	if _, err := e.AddEntry("stardew", src.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	scratch, err := e.ScratchCopy("stardew", src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if mods, err := e.UserMods("stardew", scratch.ID); err != nil || len(mods) != 1 {
		t.Fatalf("scratch mods = %+v, %v", mods, err)
	}
	listed, err := e.List("stardew")
	if err != nil || len(listed) != 1 || listed[0].ID != src.ID {
		t.Fatalf("list during a check = %+v, %v", listed, err)
	}
	if trashed, err := e.trashed("stardew"); err != nil || len(trashed) != 0 {
		t.Fatalf("trash = %+v, %v", trashed, err)
	}
	dir, err := e.ProfileDir("stardew", scratch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.PurgeScratch(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("scratch left after purge: %v", err)
	}
	if listed, _ := e.List("stardew"); !slices.ContainsFunc(listed, func(p Profile) bool { return p.ID == src.ID }) {
		t.Fatal("purge removed the real profile")
	}
	if err := e.DropScratch("stardew", src.ID); err == nil {
		t.Fatal("DropScratch removed a real profile")
	}
}

func TestDroppingAScratchProfileForgetsItsParsedCopy(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "P")
	scratch, err := e.ScratchCopy("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := e.profileDir("stardew", scratch.ID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, fileName)
	if _, err := e.read("stardew", scratch.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsedProfiles.Load(path); !ok {
		t.Fatal("the read did not cache the scratch profile")
	}
	if err := e.DropScratch("stardew", scratch.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsedProfiles.Load(path); ok {
		t.Fatal("a dropped scratch profile stays cached")
	}
}
