package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestHasRawXNBReportsWalkErrors(t *testing.T) {
	_, err := hasRawXNB(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("hasRawXNB returned nil error for a missing root")
	}
}

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

func TestRemapNestedManifestAsks(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	z := buildZip(t, "loose.zip", map[string]string{"readme.txt": "hello", "notes.md": "x", "Mod/.hidden/manifest.json": manifestJSON("X.A")})
	res, err := e.InstallArchive("stardew", p.ID, z)
	if err != nil {
		t.Fatal(err)
	}
	if res.Remap == nil || res.Fomod != nil || len(res.Profile.Entries) != 0 {
		t.Fatalf("want remap ask, got %+v", res)
	}
	if len(res.Remap.Tree) != 3 {
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

func TestRemapVariantsAskThenUpdateReuses(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	variantZip := func(name, a, b string) string {
		return buildZip(t, name, map[string]string{
			a + "/manifest.json": manifestJSON("Mod.A"),
			b + "/manifest.json": manifestJSON("Mod.A"),
		})
	}
	install := func(fileID int, z string) InstallResult {
		t.Helper()
		res, err := e.InstallNexus("stardew", p.ID, z, Source{Kind: KindNexus, Name: "v.zip", ModID: 9, FileID: fileID})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := install(1, variantZip("v1.zip", "Mod (Spring)", "Mod (Dark)"))
	if res.Remap == nil || len(res.Remap.Variants) != 2 || len(res.Profile.Entries) != 0 {
		t.Fatalf("want variant ask, got %+v", res)
	}
	if _, err := e.InstallRemap("stardew", p.ID, res.Remap.Key, ".", res.Remap.Source); err == nil {
		t.Fatal("archive root holding both variants was accepted")
	}
	res, err := e.InstallRemap("stardew", p.ID, res.Remap.Key, "Mod (Dark)", res.Remap.Source)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 {
		t.Fatalf("remap = %+v", res)
	}

	res = install(2, variantZip("v2.zip", "Mod (Spring)", "Mod (Dark)"))
	if res.Remap != nil || !res.Updated {
		t.Fatalf("update asked again: %+v", res)
	}
	if rel, _ := e.items.Root("stardew", store.NexusKey(9, 2)); rel != "Mod (Dark)" {
		t.Fatalf("update root = %q", rel)
	}

	if res = install(3, variantZip("v3.zip", "Mod (Spring) v3", "Mod (Dark) v3")); res.Remap == nil {
		t.Fatalf("renamed variants should ask, got %+v", res)
	}
}

func TestRemapVariantsWidenToBundle(t *testing.T) {
	m := func(id, folder string) manifest.Mod {
		return manifest.Mod{UniqueID: id, Folder: folder}
	}
	got := variants([]manifest.Mod{
		m("Mod.CP", "Options/Option A/[CP] Mod"), m("Mod.JA", "Options/Option A/[JA] Mod"),
		m("Mod.CP", "Options/Option B/[CP] Mod"), m("Mod.JA", "Options/Option B/[JA] Mod"),
		m("Other", "Other"),
	})
	if len(got) != 2 || got[0].Path != "Options/Option A" || got[1].Path != "Options/Option B" {
		t.Fatalf("variants = %+v", got)
	}
}
