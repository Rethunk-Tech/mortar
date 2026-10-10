package share

import (
	"strings"
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
		return profile.Entry{
			Key: k, Item: "pkg-1", File: file, Source: src, Package: true,
			Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, k), Name: file, Folder: "."}},
		}
	}
	s, _, _ := Collect(profile.Profile{Entries: []profile.Entry{entry("a.package"), entry("b.package")}})
	if len(s.Entries) != 1 {
		t.Fatalf("the archive appears %d times in the link", len(s.Entries))
	}
}

func splitEntry(item, file string, src profile.Source, off bool) profile.Entry {
	k := item + "#" + file
	id := mod.NewID(mod.FormatFolder, k)
	e := profile.Entry{
		Key: k, Item: item, File: file, Source: src, Package: true,
		Mods: []profile.Component{{ID: id, Name: file, Folder: "."}},
	}
	if off {
		e.Disabled = []mod.ID{id}
	}
	return e
}

func TestCollectCarriesTheSwitchedOffFilesOfASplitArchive(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	p := profile.Profile{Entries: []profile.Entry{splitEntry("pkg-1", "a.package", src, false), splitEntry("pkg-1", "b.package", src, true)}}
	s, _, off := Collect(p)
	if len(s.Entries) != 1 || len(off) != 0 || len(s.Entries[0].Disabled) != 1 || s.Entries[0].Disabled[0] != p.Entries[1].Mods[0].ID {
		t.Fatalf("entries %+v, off %v", s.Entries, off)
	}
	if s.Entries[0].key != "pkg-1" {
		t.Fatalf("ref key %q", s.Entries[0].key)
	}
}

func TestCollectOmitsAFullySwitchedOffSplitArchiveOnce(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	p := profile.Profile{Entries: []profile.Entry{splitEntry("pkg-1", "a.package", src, true), splitEntry("pkg-1", "b.package", src, true)}}
	inc := DefaultInclude()
	inc.DisabledMods = false
	s, _, off := Collect(p, inc)
	if len(s.Entries) != 0 || len(off) != 1 || off[0] != "pkg-1" {
		t.Fatalf("entries %+v, off %v", s.Entries, off)
	}
}

func TestLocalRefOfASplitArchiveNamesTheStoreItem(t *testing.T) {
	t.Parallel()
	item := "local-" + strings.Repeat("a", 64)
	src := profile.Source{Kind: profile.KindLocal, Name: "pack.zip"}
	e := splitEntry(item, "a.package", src, false)
	s, _, _ := Collect(profile.Profile{Entries: []profile.Entry{e, splitEntry(item, "b.package", src, false)}}, Include{LocalFiles: true})
	if len(s.Entries) != 1 || s.Entries[0].Local != item || !s.Entries[0].MatchesEntry(e) {
		t.Fatalf("%+v", s.Entries)
	}
}

func TestFileGroupNamesASplitArchiveOnce(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	a, b := splitEntry("pkg-1", "a.package", src, false), splitEntry("pkg-1", "b.package", src, false)
	groups := collectFileGroups(profile.Profile{Entries: []profile.Entry{a, b}, Groups: []profile.Group{{Name: "g", Keys: []string{a.Key, b.Key}}}})
	if len(groups) != 1 || len(groups[0].Refs) != 1 {
		t.Fatalf("%+v", groups)
	}
}
