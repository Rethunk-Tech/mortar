package share

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// A folder game's archive is one entry per file, all with the archive's source; a link names the archive once.
func TestCollectNamesAnArchiveOnceWhenItsFilesAreSeparateEntries(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	entry := func(file string) profile.Entry {
		k := "pkg-1#" + file
		return profile.Entry{Key: k, Item: "pkg-1", File: file, Source: src, Package: true,
			Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, k), Name: file, Folder: "."}}}
	}
	s, _, _ := Collect(profile.Profile{Entries: []profile.Entry{entry("a.package"), entry("b.package")}})
	if len(s.Entries) != 1 {
		t.Fatalf("the archive appears %d times in the link", len(s.Entries))
	}
}
