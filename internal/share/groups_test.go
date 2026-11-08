package share

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

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
