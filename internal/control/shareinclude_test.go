package control

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// The command's link is the window's link: it carries a mod's note only while the "notes" share setting is on.
func TestShareCommandFollowsTheShareSettings(t *testing.T) {
	s := services(t)
	ctx := context.Background()
	p, err := s.Store.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "mod.zip"), map[string]string{
		"Mod/manifest.json": `{"Name":"M","UniqueID":"Test.M","Version":"1.0.0","MinimumApiVersion":"4.0.0"}`,
	})
	res, err := s.Store.InstallSource(ctx, "stardew", p.ID, zip, profile.Source{Kind: profile.KindNexus, Name: "mod.zip", ModID: 5, FileID: 10, Version: "1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.SetEntryNoteTags("stardew", p.ID, res.Profile.Entries[0].Key, "keep this one", nil); err != nil {
		t.Fatal(err)
	}
	note := func() string {
		t.Helper()
		out, err := s.Handle(ctx, "share", Params{Game: "stardew", Profile: p.ID})
		if err != nil {
			t.Fatal(err)
		}
		link, ok := out.(ShareLink)
		if !ok {
			t.Fatalf("share returned %T", out)
		}
		got, err := share.Parse(link.App)
		if err != nil || len(got.Entries) != 1 {
			t.Fatalf("link: %+v, %v", got, err)
		}
		return got.Entries[0].Note
	}
	if got := note(); got != "keep this one" {
		t.Fatalf("note with the setting on = %q", got)
	}
	off := false
	if _, err := s.Settings.Update(func(v *settings.Settings) { v.ShareIncludeNotes = &off }); err != nil {
		t.Fatal(err)
	}
	if got := note(); got != "" {
		t.Fatalf("note with the setting off = %q", got)
	}
}
