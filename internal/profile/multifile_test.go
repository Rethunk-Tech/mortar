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
	e.item(t, "extra-new", map[string]string{"C/manifest.json": manifestJSON("X.C")})
	got, err := e.AddExtra("stardew", p.ID, entryKey, "extra-new", Source{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || !slices.Equal(got.Entries[0].ExtraStoreKeys, []string{"extra-new"}) {
		t.Fatalf("add extras: %+v", got.Entries)
	}
	if !hasMod(got.Entries[0], "X.A") || !hasMod(got.Entries[0], "X.C") {
		t.Fatalf("add mods: %+v", got.Entries[0].Mods)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), entryKey, "extra-new", "C", "manifest.json")); err != nil {
		t.Fatalf("extra folder: %v", err)
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

	writeFile(t, filepath.Join(e.mods(p.ID), store.NexusKey(7, 3), "extra-new", "C"), "config.json", "extra-save")
	got, err = e.RollBack("stardew", p.ID, store.NexusKey(7, 3))
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].Key != entryKey || !slices.Equal(got.Entries[0].ExtraStoreKeys, []string{"extra-new"}) {
		t.Fatalf("rollback entry: %+v", got.Entries[0])
	}
	if !hasMod(got.Entries[0], "X.A") || !hasMod(got.Entries[0], "X.C") {
		t.Fatalf("rollback mods: %+v", got.Entries[0].Mods)
	}
	if b := read(t, filepath.Join(e.mods(p.ID), entryKey, "extra-new", "C", "config.json")); b != "extra-save" {
		t.Fatalf("extra carry-over after rollback = %q", b)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), entryKey, "A", "manifest.json")); err != nil {
		t.Fatalf("primary after rollback: %v", err)
	}

	if _, err := e.RemoveEntry("stardew", p.ID, entryKey); err != nil {
		t.Fatal(err)
	}
	left, err := e.read("stardew", p.ID)
	if err != nil || len(left.Entries) != 0 {
		t.Fatalf("remove: %+v %v", left.Entries, err)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), entryKey)); !os.IsNotExist(err) {
		t.Fatalf("entry folder after remove: %v", err)
	}
}

func TestSplitAndCombineEntries(t *testing.T) {
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
	extraKey := store.NexusKey(7, 2)
	e.item(t, extraKey, map[string]string{"B/manifest.json": manifestJSON("X.B")})
	if _, err := e.AddExtra("stardew", p.ID, entryKey, extraKey, Source{Kind: KindNexus, ModID: 7, FileID: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SplitExtra("stardew", p.ID, entryKey, "missing-extra"); err == nil {
		t.Fatal("split of an unknown extra must fail")
	}
	before, err := e.History("stardew", p.ID)
	if err != nil || len(before) == 0 {
		t.Fatalf("history before split: %v %v", before, err)
	}
	got, err := e.SplitExtra("stardew", p.ID, entryKey, extraKey)
	if err != nil {
		t.Fatal(err)
	}
	parent, extra, okParent, okExtra := Entry{}, Entry{}, false, false
	for _, ent := range got.Entries {
		switch ent.Key {
		case entryKey:
			parent, okParent = ent, true
		case extraKey:
			extra, okExtra = ent, true
		}
	}
	if !okParent || !okExtra {
		t.Fatalf("split entries: %+v", got.Entries)
	}
	if len(parent.ExtraStoreKeys) != 0 || hasMod(parent, "X.B") || !hasMod(parent, "X.A") || !hasMod(extra, "X.B") {
		t.Fatalf("split mods: parent=%+v extra=%+v", parent, extra)
	}
	if extra.Source.Kind != KindNexus || extra.Source.ModID != 7 || extra.Source.FileID != 2 {
		t.Fatalf("split extra source: %+v", extra.Source)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), entryKey, extraKey)); !os.IsNotExist(err) {
		t.Fatalf("extra dir after split: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), extraKey, "B", "manifest.json")); err != nil {
		t.Fatalf("split extra folder: %v", err)
	}

	undone, err := e.Revert("stardew", p.ID, before[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	var undoneParent Entry
	for _, ent := range undone.Entries {
		if ent.Key == entryKey {
			undoneParent = ent
		}
	}
	if !slices.Equal(undoneParent.ExtraStoreKeys, []string{extraKey}) {
		t.Fatalf("undo split: %+v", undone.Entries)
	}

	if _, err = e.SplitExtra("stardew", p.ID, entryKey, extraKey); err != nil {
		t.Fatal(err)
	}
	nested := store.NexusKey(7, 4)
	e.item(t, nested, map[string]string{"D/manifest.json": manifestJSON("X.D")})
	if _, err := e.AddExtra("stardew", p.ID, extraKey, nested, Source{Kind: KindNexus, ModID: 7, FileID: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.CombineEntries("stardew", p.ID, entryKey, extraKey); err == nil {
		t.Fatal("combine when other has extras must fail")
	}
	if _, err := e.SplitExtra("stardew", p.ID, extraKey, nested); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RemoveEntry("stardew", p.ID, nested); err != nil {
		t.Fatal(err)
	}

	combined, err := e.CombineEntries("stardew", p.ID, entryKey, extraKey)
	if err != nil {
		t.Fatal(err)
	}
	var combinedParent Entry
	for _, ent := range combined.Entries {
		if ent.Key == entryKey {
			combinedParent = ent
		}
	}
	if !slices.Equal(combinedParent.ExtraStoreKeys, []string{extraKey}) {
		t.Fatalf("combine: %+v", combined.Entries)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), entryKey, extraKey, "B", "manifest.json")); err != nil {
		t.Fatalf("combine extra folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.mods(p.ID), extraKey)); !os.IsNotExist(err) {
		t.Fatalf("other entry folder after combine: %v", err)
	}

	otherZip := buildZip(t, "c.zip", map[string]string{"C/manifest.json": manifestJSON("Y.C")})
	otherSrc := Source{Kind: KindNexus, Name: "c.zip", ModID: 8, FileID: 1, Version: "1.0"}
	if err := e.items.AddArchiveKey("stardew", store.NexusKey(8, 1), otherZip); err != nil {
		t.Fatal(err)
	}
	page, err := e.AddEntry("stardew", p.ID, store.NexusKey(8, 1), otherSrc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.CombineEntries("stardew", p.ID, entryKey, store.NexusKey(8, 1)); err == nil {
		t.Fatal("combine across Nexus pages must fail")
	}
	_ = page
	_ = got
}

func hasMod(e Entry, id string) bool {
	return slices.ContainsFunc(e.Mods, func(m EntryMod) bool { return m.UniqueID == id })
}
