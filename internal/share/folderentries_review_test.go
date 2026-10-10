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

// Each file of a split archive keeps its own note and tags through a link, and the Notes switch drops them all.
func TestSplitArchiveNotesTravelPerFile(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	a, b, c := splitEntry("pkg-1", "a.package", src, false), splitEntry("pkg-1", "sub/b.package", src, false), splitEntry("pkg-1", "c.package", src, false)
	a.Note, b.Note, b.Tags = "first", "second", []string{"hair"}
	p := profile.Profile{Name: "P", Entries: []profile.Entry{a, b, c}}
	res, err := Encode("stardew", p, profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(res.Payload)
	if err != nil || len(got.Entries) != 1 {
		t.Fatalf("decode: %v, %+v", err, got.Entries)
	}
	ref := got.Entries[0]
	if note, _ := ref.EntryNote(a); note != "first" {
		t.Fatalf("a note %q", note)
	}
	if note, tags := ref.EntryNote(b); note != "second" || len(tags) != 1 || tags[0] != "hair" {
		t.Fatalf("b note %q tags %v", note, tags)
	}
	if note, tags := ref.EntryNote(c); note != "" || len(tags) != 0 || ref.Note != "" {
		t.Fatalf("c note %q tags %v, ref note %q", note, tags, ref.Note)
	}
	a.Note, b.Note, b.Tags = "", "", nil
	whole := profile.Entry{Key: "w", Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 1}}
	notes := EntryNotes([]profile.Entry{a, b, c, whole, whole}, []Ref{ref, {ModID: 7, FileID: 1, Note: "kept whole"}})
	if len(notes) != 3 || notes[a.Key].Note != "first" || notes[b.Key].Note != "second" || notes["w"].Note != "kept whole" {
		t.Fatalf("notes on receive: %+v", notes)
	}
	a.Note, b.Note, b.Tags = "first", "second", []string{"hair"}
	p.Entries = []profile.Entry{a, b, c}
	inc := DefaultInclude()
	inc.Notes = false
	if s, _, _ := Collect(p, inc); len(s.Entries[0].Files) != 0 {
		t.Fatalf("notes left in with the switch off: %+v", s.Entries[0].Files)
	}
	bad := Ref{ModID: 5, FileID: 9, Files: map[string]FileNote{"../x.package": {Note: "n"}}}
	if validDetails(bad) {
		t.Fatal("a file path that leaves the archive passed")
	}
}
