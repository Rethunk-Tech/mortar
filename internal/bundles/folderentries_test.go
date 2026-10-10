package bundles

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestSnapshotNamesTheStoreItemOfAPerFileEntry(t *testing.T) {
	t.Parallel()
	id := mod.NewID(mod.FormatFolder, "pkg-1#a.package")
	p := profile.Profile{Entries: []profile.Entry{{
		Key: "pkg-1#a.package", Item: "pkg-1", File: "a.package", Package: true,
		Source: profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9},
		Mods:   []profile.Component{{ID: id, Name: "a.package", Folder: "."}},
	}}}
	mods, err := snapshot(p, []mod.ID{id})
	if err != nil || len(mods) != 1 || mods[0].EntryKey != "pkg-1" {
		t.Fatalf("%+v %v", mods, err)
	}
}

func TestSnapshotKeepsTheSourceOfCurseForgeAndItchEntries(t *testing.T) {
	t.Parallel()
	for _, src := range []profile.Source{
		{Kind: profile.KindCurseForge, Name: "7", FileID: 9},
		{Kind: profile.KindItch, Name: "someone/cool-mod"},
	} {
		id := mod.NewID(mod.FormatFolder, "pkg-1")
		p := profile.Profile{Entries: []profile.Entry{{Key: "pkg-1", Source: src, Package: true, Mods: []profile.Component{{ID: id, Name: "m", Folder: "."}}}}}
		mods, err := snapshot(p, []mod.ID{id})
		if err != nil || len(mods) != 1 || mods[0].Source.Kind != src.Kind || mods[0].Source.Name != src.Name || mods[0].Source.FileID != src.FileID {
			t.Fatalf("%+v %v", mods, err)
		}
	}
}

func TestSnapshotGroupsAKeptWholeArchiveAndItsTrayEntryUnderOneStoreItem(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	a, b := mod.NewID(mod.FormatFolder, "pkg-1"), mod.NewID(mod.FormatFolder, "pkg-1#tray")
	p := profile.Profile{Entries: []profile.Entry{
		{Key: "pkg-1", Source: src, Package: true, Mods: []profile.Component{{ID: a, Name: "a", Folder: "."}}},
		{Key: "pkg-1#tray", Item: "pkg-1", Source: src, Package: true, Mods: []profile.Component{{ID: b, Name: "t", Folder: "."}}},
	}}
	mods, err := snapshot(p, []mod.ID{a, b})
	if err != nil || len(mods) != 2 || mods[0].EntryKey != "pkg-1" || mods[1].EntryKey != "pkg-1" {
		t.Fatalf("%+v %v", mods, err)
	}
}
