package control

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// The CLI must star the same key the window does (frontend/src/mods/favourites.ts entryFavouriteKey).
func TestEntryFavouriteKeyMatchesTheWindow(t *testing.T) {
	for src, want := range map[profile.Source]string{
		{Kind: "nexus", ModID: 42}:                  "nexus:42",
		{Kind: "nexus"}:                             "",
		{Kind: "github", Repo: "Me/Mod", Name: "x"}: "github:me/mod",
		{Kind: "thunderstore", Name: "Team-Mod"}:    "thunderstore:team-mod",
		{Kind: "local"}:                             "",
	} {
		if got := entryFavouriteKey(src); got != want {
			t.Errorf("%+v: %q, want %q", src, got, want)
		}
	}
}
