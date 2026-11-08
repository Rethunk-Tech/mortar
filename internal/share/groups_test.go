package share

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestIdentityOfAcceptSetMatchesRefOf(t *testing.T) {
	local := "local-" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	inc := Include{LocalFiles: true}
	entries := []profile.Entry{
		{Key: "nexus", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2}},
		{Key: "nexus-bad", Source: profile.Source{Kind: profile.KindNexus, ModID: 0, FileID: 0}},
		{Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "owner/repo", Tag: "v1", Asset: "mod.zip"}},
		{Key: "gh-bad", Source: profile.Source{Kind: profile.KindGitHub}},
		{Key: "patreon", Source: profile.Source{Kind: profile.KindPatreon, Name: "12345"}},
		{Key: "patreon-bad", Source: profile.Source{Kind: profile.KindPatreon}},
		{Key: "itch", Source: profile.Source{Kind: profile.KindItch, Name: "user/game"}},
		{Key: "itch-bad", Source: profile.Source{Kind: profile.KindItch}},
		{Key: "cf", Source: profile.Source{Kind: profile.KindCurseForge, Name: "998265", FileID: 777}},
		{Key: "cf-bad", Source: profile.Source{Kind: profile.KindCurseForge}},
		{Key: "modio", Source: profile.Source{Kind: profile.KindModio, Name: "42", FileID: 1}},
		{Key: "modio-bad", Source: profile.Source{Kind: profile.KindModio}},
		{Key: "vs", Source: profile.Source{Kind: profile.KindVSModDB, Name: "1-2", FileID: 7, Version: "1.0.0"}},
		{Key: "vs-bad", Source: profile.Source{Kind: profile.KindVSModDB}},
		{Key: "px", Source: profile.Source{Kind: profile.KindParadox, Name: "12345", FileID: 7, Version: "1.0.0"}},
		{Key: "px-bad", Source: profile.Source{Kind: profile.KindParadox}},
		{Key: "ts", Source: profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"}},
		{Key: "ts-bad", Source: profile.Source{Kind: profile.KindThunderstore}},
		{Key: local, Source: profile.Source{Kind: profile.KindLocal, Name: "a.zip"}},
		{Key: "local-bad", Source: profile.Source{Kind: profile.KindLocal}},
		{Key: "modrinth", Source: profile.Source{Kind: profile.KindModrinth, Name: "proj"}},
		{Key: "unknown", Source: profile.Source{Kind: "nope"}},
	}
	for _, e := range entries {
		want, reason := refOf(e, inc)
		got, ok := identityOf(e)
		if ok != (reason == "") {
			t.Errorf("%s: identityOf ok=%v refOf reason=%q", e.Key, ok, reason)
			continue
		}
		if !ok {
			continue
		}
		want.Disabled, want.Overlay = nil, nil
		if got.Identity() != want.Identity() || got.LocalName != want.LocalName {
			t.Errorf("%s: identityOf=%s localName=%q refOf=%s localName=%q", e.Key, got.Identity(), got.LocalName, want.Identity(), want.LocalName)
		}
	}
}

// A group names the same identities a link carries, so an entry whose ref fails the link's own checks drops out of it
// rather than naming an identity the link would never hold.
func TestIdentityOfRejectsRefsTheLinkRejects(t *testing.T) {
	for _, e := range []profile.Entry{
		{Key: "g", Source: profile.Source{Kind: profile.KindGitHub, Repo: "owner/repo", Tag: "v1 beta", Asset: "x.zip"}},
		{Key: "p", Source: profile.Source{Kind: profile.KindParadox, Name: "not-a-number", FileID: 7, Version: "1.0"}},
		{Key: "v", Source: profile.Source{Kind: profile.KindVSModDB, Name: "not-a-number", FileID: 7, Version: "1.0"}},
	} {
		if _, ok := identityOf(e); ok {
			t.Fatalf("%s: identityOf accepted %+v", e.Key, e.Source)
		}
	}
}

// A nested group is a "/" path inside one flat name, so the link carries no new field and a Mortar that does not nest
// reads it as a plain group.
func TestNestedGroupNamesRoundTripAsFlatNames(t *testing.T) {
	e := profile.Entry{Key: "k", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2}, Mods: []profile.Component{{ID: "smapi:A.One", Folder: "A"}}}
	p := profile.Profile{Name: "P", Entries: []profile.Entry{e}, Groups: []profile.Group{{Name: "Looks/Hats", Keys: []string{"k"}}}}
	got := roundTrip(t, p)
	groups := RefGroups(got.Entries)
	if len(groups) != 1 || groups[0].Name != "Looks/Hats" {
		t.Fatalf("groups = %+v", groups)
	}
}
