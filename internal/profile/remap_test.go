package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestRemapJunkWrapperSkippedSilently(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	z := buildZip(t, "junk.zip", map[string]string{
		"__MACOSX/foo/manifest.json": manifestJSON("Junk.A"),
		"Good/manifest.json":         manifestJSON("Good.A"),
	})
	res, err := e.InstallArchive("stardew", p.ID, z)
	if err != nil {
		t.Fatal(err)
	}
	if res.Remap != nil || res.Fomod != nil || len(res.Added) != 1 || res.Added[0] != "Good.A" {
		t.Fatalf("install = %+v", res)
	}
	mods := e.mods(p.ID)
	if _, err := os.Stat(filepath.Join(mods, res.Profile.Entries[0].Key, "__MACOSX")); err == nil {
		t.Fatal("junk copied into mods")
	}
}

func TestRemapLooseFilesAsks(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	z := buildZip(t, "loose.zip", map[string]string{"readme.txt": "hello", "notes.md": "x"})
	res, err := e.InstallArchive("stardew", p.ID, z)
	if err != nil {
		t.Fatal(err)
	}
	if res.Remap == nil || res.Fomod != nil || len(res.Profile.Entries) != 0 {
		t.Fatalf("want remap ask, got %+v", res)
	}
	if len(res.Remap.Tree) != 2 {
		t.Fatalf("tree = %+v", res.Remap.Tree)
	}
}

func TestRemapStoredRootReused(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	z := buildZip(t, "wrap.zip", map[string]string{
		"readme.txt":              "x",
		"Outer/noise.txt":         "n",
		"Outer/Mod/manifest.json": manifestJSON("Mod.A"),
		"Outer/Mod/extra.txt":     "keep",
	})
	src := Source{Kind: KindNexus, Name: "wrap.zip", ModID: 9, FileID: 4}
	key := store.NexusKey(9, 4)
	if err := e.items.AddArchiveKey("stardew", key, z); err != nil {
		t.Fatal(err)
	}
	if err := e.items.SetRoot("stardew", key, "Outer/Mod"); err != nil {
		t.Fatal(err)
	}
	res, err := e.InstallNexus("stardew", p.ID, z, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Remap != nil || len(res.Added) != 1 || res.Added[0] != "Mod.A" {
		t.Fatalf("first = %+v", res)
	}
	placed := filepath.Join(e.mods(p.ID), key)
	if _, err := os.Stat(filepath.Join(placed, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(placed, "noise.txt")); err == nil {
		t.Fatal("copied outside the chosen root")
	}
	if _, err := os.Stat(filepath.Join(placed, "readme.txt")); err == nil {
		t.Fatal("copied archive root files")
	}

	q, _ := e.Create("stardew", "Q")
	res, err = e.InstallNexus("stardew", q.ID, z, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Remap != nil || len(res.Added) != 1 {
		t.Fatalf("reuse asked again: %+v", res)
	}

	modDir, err := e.items.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(modDir); err != nil {
		t.Fatal(err)
	}
	r, _ := e.Create("stardew", "R")
	res, err = e.InstallNexus("stardew", r.ID, z, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Remap == nil {
		t.Fatalf("missing root should ask, got %+v", res)
	}
}
