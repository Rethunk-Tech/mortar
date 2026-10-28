package control

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// The CLI must star the same key the window does (frontend/src/mods/favourites.ts entryFavouriteKey).
func TestEntryFavouriteKeyMatchesTheWindow(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		e    profile.Entry
		want string
	}{
		{profile.Entry{Source: profile.Source{Kind: "nexus", ModID: 42}}, "nexus:42"},
		{profile.Entry{Source: profile.Source{Kind: "nexus"}}, ""},
		{profile.Entry{Source: profile.Source{Kind: "github", Repo: "Me/Mod", Name: "x"}}, "github:me/mod"},
		{profile.Entry{Source: profile.Source{Kind: "thunderstore", Name: "Team-Mod"}}, "thunderstore:team-mod"},
		{profile.Entry{Key: "a.zip", Source: profile.Source{Kind: "local", Name: "A.zip"}}, "local:a.zip|a.zip"},
		{profile.Entry{Key: "b", Item: "b", Source: profile.Source{Kind: "local", Name: "A.zip"}}, "local:a.zip|b"},
		{profile.Entry{Source: profile.Source{Kind: "local"}}, ""},
	} {
		if got := entryFavouriteKey(c.e); got != c.want {
			t.Errorf("%+v: %q, want %q", c.e, got, c.want)
		}
	}
}
