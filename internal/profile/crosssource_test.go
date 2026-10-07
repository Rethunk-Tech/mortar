package profile

import (
	"slices"
	"testing"
)

// A file from another site that holds the same mod replaces the entry in place: its note and tags stay and its source
// becomes the new site, so later update checks ask that site.
func TestCurseForgeFileReplacesANexusEntryKeepingItsSettings(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "P")
	v1 := buildZip(t, "a.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	res, err := e.InstallSource("stardew", p.ID, v1, Source{Kind: KindNexus, Name: "a.zip", ModID: 7, FileID: 1, Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetEntryNoteTags("stardew", p.ID, res.Profile.Entries[0].Key, "keep me", []string{"ui"}); err != nil {
		t.Fatal(err)
	}
	v2 := buildZip(t, "a2.zip", map[string]string{"A/manifest.json": `{"Name":"X.A","Author":"me","Version":"1.0.1","UniqueID":"X.A"}`})
	got, err := e.InstallSource("stardew", p.ID, v2, Source{Kind: KindCurseForge, Name: "998265", Version: "X.A 1.0.1", FileID: 777})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Profile.Entries) != 1 || !got.Updated {
		t.Fatalf("entries %+v updated %v", got.Profile.Entries, got.Updated)
	}
	en := got.Profile.Entries[0]
	if en.Source.Kind != KindCurseForge || en.Note != "keep me" || !slices.Equal(en.Tags, []string{"ui"}) || !en.Enabled("smapi:X.A") {
		t.Fatalf("entry = %+v", en)
	}
}
