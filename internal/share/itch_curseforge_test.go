package share

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestAnItchEntryTravelsAsItsPageNameAlone(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "Desk", Entries: []profile.Entry{
		{Key: "local-" + strings.Repeat("cd", 32), Source: profile.Source{Kind: profile.KindItch, Name: "someone/cool-mod"}},
	}}
	res, err := Encode("stardew", p, profile.ShareFacts{})
	if err != nil || len(res.LeftOut) != 0 || len(res.Shared.Entries) != 1 {
		t.Fatalf("result = %+v, %v", res, err)
	}
	got, err := Parse(res.Payload)
	if err != nil || len(got.Entries) != 1 || got.Entries[0].Itch != "someone/cool-mod" || got.Entries[0].Local != "" {
		t.Fatalf("parsed = %+v, %v", got.Entries, err)
	}
	if !got.Entries[0].MatchesEntry(p.Entries[0]) || got.Entries[0].Identity() != "i:someone/cool-mod" {
		t.Fatal("the ref names its entry")
	}
	for _, bad := range []Ref{{Itch: "../x"}, {Itch: "nogame"}, {Itch: "a/b", ModID: 1}, {Itch: "a/b", Version: "1.0.0"}, {Itch: "a/b", CurseForge: 1, FileID: 1}} {
		if bad.valid() {
			t.Fatalf("%+v passed", bad)
		}
	}
}

func TestACurseForgeEntryTravelsAsProjectAndFile(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "Desk", Entries: []profile.Entry{
		{Key: "pkg-1", Source: profile.Source{Kind: profile.KindCurseForge, Name: "309243", Version: "Content Patcher 2.9.1", FileID: 555}},
	}}
	res, err := Encode("stardew", p, profile.ShareFacts{})
	if err != nil || len(res.LeftOut) != 0 || len(res.Shared.Entries) != 1 {
		t.Fatalf("result = %+v, %v", res, err)
	}
	got, err := Parse(res.Payload)
	if err != nil || len(got.Entries) != 1 || got.Entries[0].CurseForge != 309243 || got.Entries[0].FileID != 555 {
		t.Fatalf("parsed = %+v, %v", got.Entries, err)
	}
	if !got.Entries[0].MatchesEntry(p.Entries[0]) || got.Entries[0].Identity() != "c:309243:555" {
		t.Fatal("the ref names its entry")
	}
	for _, bad := range []Ref{{CurseForge: 1}, {CurseForge: 1, FileID: 1, ModID: 2}, {CurseForge: 1, FileID: 1, Version: "1.0.0"}, {CurseForge: -1, FileID: 1}} {
		if bad.valid() {
			t.Fatalf("%+v passed", bad)
		}
	}
}

func TestACurseForgeLinkNeedsTheGamesCurseForgeKey(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "Desk", Entries: []profile.Entry{
		{Key: "pkg-1", Source: profile.Source{Kind: profile.KindCurseForge, Name: "1", FileID: 2}},
	}}
	if _, err := Encode("lethal-company", p, profile.ShareFacts{}); err == nil {
		t.Fatal("a CurseForge ref for a game with no CurseForge source was accepted")
	}
}

func TestSplitCurseForgeArchiveIsOneRef(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindCurseForge, Name: "7", FileID: 9}
	s, _, _ := Collect(profile.Profile{Entries: []profile.Entry{splitEntry("pkg-1", "a.package", src, false), splitEntry("pkg-1", "b.package", src, true)}})
	if len(s.Entries) != 1 || s.Entries[0].CurseForge != 7 || len(s.Entries[0].Disabled) != 1 {
		t.Fatalf("%+v", s.Entries)
	}
}
