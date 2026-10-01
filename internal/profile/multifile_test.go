package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestDefaultMerge(t *testing.T) {
	if !DefaultMerge("OPTIONAL") || !DefaultMerge("miscellaneous") || DefaultMerge("MAIN") || DefaultMerge("") {
		t.Fatal("optional and miscellaneous add by default; MAIN does not")
	}
}

func TestSamePageAsk(t *testing.T) {
	p := Profile{Entries: []Entry{{
		Key: "nexus-7-1", Source: Source{Kind: KindNexus, ModID: 7, FileID: 1}, Mods: []EntryMod{{Name: "A", UniqueID: "A"}},
	}}}
	ask, ok := SamePageAsk(p, 7, 2, "OPTIONAL")
	if !ok || ask.EntryKey != "nexus-7-1" || !ask.DefaultAdd || ask.Label == "" {
		t.Fatalf("optional extra: %+v %v", ask, ok)
	}
	if _, ok := SamePageAsk(p, 7, 1, "OPTIONAL"); ok {
		t.Fatal("the same file is not a merge")
	}
	p.Entries[0].ExtraStoreKeys = []string{store.NexusKey(7, 2)}
	if _, ok := SamePageAsk(p, 7, 2, "OPTIONAL"); ok {
		t.Fatal("an extra already on the entry is not a merge")
	}
	if _, ok := SamePageAsk(p, 8, 2, "OPTIONAL"); ok {
		t.Fatal("a different page is not a merge")
	}
}

func TestAddUpdateRemoveMultiFileEntry(t *testing.T) {
	e := newEnv(t)
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	main := buildZip(t, "a.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	src := Source{Kind: KindNexus, Name: "a.zip", ModID: 7, FileID: 1, Version: "1.0"}
	res, err := e.InstallNexus("stardew", p.ID, main, src)
	if err != nil {
		t.Fatal(err)
	}
	entryKey := res.Profile.Entries[0].Key
	e.item(t, "extra-old", map[string]string{"B/manifest.json": manifestJSON("X.B")})
	got, err := e.AddExtra("stardew", p.ID, entryKey, "extra-old", Source{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || !slices.Equal(got.Entries[0].ExtraStoreKeys, []string{"extra-old"}) {
		t.Fatalf("add extras: %+v", got.Entries)
	}
	if !hasMod(got.Entries[0], "X.A") || !hasMod(got.Entries[0], "X.B") {
		t.Fatalf("add mods: %+v", got.Entries[0].Mods)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), entryKey, "extra-old", "B", "manifest.json")); err != nil {
		t.Fatalf("extra folder: %v", err)
	}

	e.item(t, "extra-new", map[string]string{"C/manifest.json": manifestJSON("X.C")})
	got, err = e.UpdateExtra("stardew", p.ID, entryKey, "extra-old", "extra-new")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Entries[0].ExtraStoreKeys, []string{"extra-new"}) || hasMod(got.Entries[0], "X.B") || !hasMod(got.Entries[0], "X.C") {
		t.Fatalf("update extra: %+v", got.Entries[0])
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), entryKey, "extra-old")); !os.IsNotExist(err) {
		t.Fatalf("old extra folder: %v", err)
	}

	next := buildZip(t, "a2.zip", map[string]string{"A/manifest.json": `{"Name":"X.A","Author":"me","Version":"2.0.0","UniqueID":"X.A"}`})
	neu := Source{Kind: KindNexus, Name: "a2.zip", ModID: 7, FileID: 3, Version: "2.0"}
	if err := e.items.AddArchiveKey("stardew", store.NexusKey(7, 3), next); err != nil {
		t.Fatal(err)
	}
	got, err = e.UpdateMultiFile("stardew", p.ID, entryKey, store.NexusKey(7, 3), &neu)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].Key != store.NexusKey(7, 3) || !slices.Equal(got.Entries[0].ExtraStoreKeys, []string{"extra-new"}) {
		t.Fatalf("update primary: %+v", got.Entries[0])
	}
	if !hasMod(got.Entries[0], "X.A") || !hasMod(got.Entries[0], "X.C") {
		t.Fatalf("update primary mods: %+v", got.Entries[0].Mods)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), store.NexusKey(7, 3), "extra-new", "C", "manifest.json")); err != nil {
		t.Fatalf("extras after primary update: %v", err)
	}

	if _, err := e.RemoveEntry("stardew", p.ID, store.NexusKey(7, 3)); err != nil {
		t.Fatal(err)
	}
	left, err := e.read("stardew", p.ID)
	if err != nil || len(left.Entries) != 0 {
		t.Fatalf("remove: %+v %v", left.Entries, err)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), store.NexusKey(7, 3))); !os.IsNotExist(err) {
		t.Fatalf("entry folder after remove: %v", err)
	}
}

func hasMod(e Entry, id string) bool {
	return slices.ContainsFunc(e.Mods, func(m EntryMod) bool { return m.UniqueID == id })
}
