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
