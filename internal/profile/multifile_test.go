package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/nexus"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestDefaultMerge(t *testing.T) {
	t.Parallel()
	if !DefaultMerge("OPTIONAL") || !DefaultMerge("miscellaneous") || DefaultMerge("MAIN") || DefaultMerge("") {
		t.Fatal("optional and miscellaneous add by default; MAIN does not")
	}
}

func TestSamePageAsk(t *testing.T) {
	t.Parallel()
	p := Profile{Entries: []Entry{{
		Key: "nexus-7-1", Source: Source{Kind: KindNexus, ModID: 7, FileID: 1}, Mods: []Component{{Name: "A", ID: "smapi:A"}},
	}}}
	in := func(fileID int) IncomingFile { return IncomingFile{ModID: 7, FileID: fileID, Category: "OPTIONAL"} }
	ask, _, ok := SamePageAsk(p, in(2))
	if !ok || ask.EntryKey != "nexus-7-1" || !ask.DefaultAdd || ask.Label == "" {
		t.Fatalf("optional extra: %+v %v", ask, ok)
	}
	if _, _, ok := SamePageAsk(p, in(1)); ok {
		t.Fatal("the same file is not a merge")
	}
	other := in(2)
	other.ModIDs = []mod.ID{"smapi:B"}
	if _, up, ok := SamePageAsk(p, other); !ok || up != 0 {
		t.Fatal("a file holding other mods is another file")
	}
	p.Entries[0].ExtraStoreKeys = []string{store.NexusKey(7, 2)}
	if _, _, ok := SamePageAsk(p, in(2)); ok {
		t.Fatal("an extra already on the entry is not a merge")
	}
	if _, _, ok := SamePageAsk(Profile{Entries: p.Entries}, IncomingFile{ModID: 8, FileID: 2}); ok {
		t.Fatal("a different page is not a merge")
	}
}

// The installed entries and new files are those of two real updates clicked from Nexus's Mod Manager Download.
func TestSamePageAskTakesANewerFileAsAnUpdate(t *testing.T) {
	t.Parallel()
	radiance := Entry{
		Key:    "nexus-49397-185426",
		Source: Source{Kind: KindNexus, Name: "SDV-Radiance 2.2.5 49397 2.2.5 2026-10-04T04-47Z KAEi8aby1.zip", ModID: 49397, FileID: 185426, Version: "2.2.5"},
		Mods:   []Component{{ID: "smapi:phuicmt.SDVRadiance", Name: "SDV-Radiance", Version: "2.2.5", Folder: "SDV-Radiance"}},
	}
	eac := Entry{
		Key:    "nexus-25328-177076",
		Source: Source{Kind: KindNexus, Name: "ExtraAnimalConfig 25328 1.9.14 2026-08-01T22-45Z p52oAOgbJ.zip", ModID: 25328, FileID: 177076, Version: "1.9.14"},
		Mods:   []Component{{ID: "smapi:selph.ExtraAnimalConfig", Name: "ExtraAnimalConfig", Version: "1.9.14", Folder: "ExtraAnimalConfig"}},
	}
	p := Profile{Entries: []Entry{radiance, eac}}
	for _, tc := range []struct {
		name string
		in   IncomingFile
		want int
	}{
		{"same mods inside", IncomingFile{ModID: 49397, FileID: 185494, Category: "MAIN", ModIDs: []mod.ID{"smapi:phuicmt.SDVRadiance"}}, 185426},
		{"same mods, other case", IncomingFile{ModID: 25328, FileID: 185524, Category: "MAIN", ModIDs: []mod.ID{"smapi:selph.extraanimalconfig"}}, 177076},
		{"file_updates chain", IncomingFile{ModID: 25328, FileID: 185524, Category: "MAIN", Files: []nexus.File{
			{FileID: 177076, Category: "OLD_VERSION", ReplacedBy: 180001}, {FileID: 180001, Category: "OLD_VERSION", ReplacedBy: 185524}, {FileID: 185524, Category: "MAIN"},
		}}, 177076},
		{"retired main file", IncomingFile{ModID: 49397, FileID: 185494, Category: "MAIN", Files: []nexus.File{
			{FileID: 185426, Category: "ARCHIVED"}, {FileID: 185494, Category: "MAIN"},
		}}, 185426},
	} {
		ask, up, ok := SamePageAsk(p, tc.in)
		if ok || up != tc.want {
			t.Errorf("%s: ask %+v %v, updates %d, want %d", tc.name, ask, ok, up, tc.want)
		}
	}
	if _, up, ok := SamePageAsk(p, IncomingFile{ModID: 49397, FileID: 185494, Category: "OPTIONAL", Files: []nexus.File{{FileID: 185426, Category: "MAIN"}}}); !ok || up != 0 {
		t.Error("an optional file beside a current main file is another file")
	}
}

func TestANewerMainFileReplacesItsEntryWhenItsModsWereRenamed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "P")
	old := buildZip(t, "a.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	if _, err := e.InstallSource("stardew", p.ID, old, Source{Kind: KindNexus, Name: "a.zip", ModID: 7, FileID: 1}); err != nil {
		t.Fatal(err)
	}
	next := buildZip(t, "b.zip", map[string]string{"B/manifest.json": manifestJSON("X.B")})
	got, err := e.InstallSource("stardew", p.ID, next, Source{Kind: KindNexus, Name: "b.zip", ModID: 7, FileID: 2}.WithReplacing(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Profile.Entries) != 1 || got.Profile.Entries[0].Key != store.NexusKey(7, 2) || !got.Updated || !hasMod(got.Profile.Entries[0], "smapi:X.B") {
		t.Fatalf("entries %+v updated %v", got.Profile.Entries, got.Updated)
	}
}

func TestAddUpdateRemoveMultiFileEntry(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "P")
	main := buildZip(t, "a.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	src := Source{Kind: KindNexus, Name: "a.zip", ModID: 7, FileID: 1, Version: "1.0"}
	res, err := e.InstallSource("stardew", p.ID, main, src)
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
	if !hasMod(got.Entries[0], "smapi:X.A") || !hasMod(got.Entries[0], "smapi:X.C") {
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
	if !hasMod(got.Entries[0], "smapi:X.A") || !hasMod(got.Entries[0], "smapi:X.C") {
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
	if !hasMod(got.Entries[0], "smapi:X.A") || !hasMod(got.Entries[0], "smapi:X.C") {
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
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "P")
	main := buildZip(t, "a.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	src := Source{Kind: KindNexus, Name: "a.zip", ModID: 7, FileID: 1, Version: "1.0"}
	res, err := e.InstallSource("stardew", p.ID, main, src)
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
	if len(parent.ExtraStoreKeys) != 0 || hasMod(parent, "smapi:X.B") || !hasMod(parent, "smapi:X.A") || !hasMod(extra, "smapi:X.B") {
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

func hasMod(e Entry, id mod.ID) bool {
	return slices.ContainsFunc(e.Mods, func(m Component) bool { return m.ID == id })
}
