package profile

import "testing"

func TestValidChangedPath(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		game, rel string
		want      bool
	}{
		{game: "sims4", rel: "Mods/mc/x.cfg", want: true},
		{game: "sims4", rel: "Mods/mc/mc_state.dat", want: true},
		{game: "sims4", rel: "Mods/mc/evil.ts4script"},
		{game: "sims4", rel: "Mods/pack/evil.dll"},
		{game: "sims4", rel: "changed/Mods/old/x.cfg", want: true},
		{game: "vintage-story", rel: "Mods/pack/modinfo.json", want: true},
		// Stardew deploys by redirect, Lethal Company has no mods target: neither accepts a copy.
		{game: "stardew", rel: "Mods/mc/x.cfg"},
		{game: "lethal-company", rel: "BepInEx/plugins/x.dll"},
		{game: "no-such-game", rel: "Mods/x.cfg"},
		{game: "sims4", rel: "profile.json"},
		{game: "sims4", rel: ""},
		{game: "sims4", rel: `Mods\x.cfg`},
		{game: "sims4", rel: "../Mods/x.cfg"},
		{game: "sims4", rel: "Mods/../x.cfg"},
		{game: "sims4", rel: "Mods//x.cfg"},
		{game: "sims4", rel: "Mods"},
		{game: "sims4", rel: "changed"},
	} {
		if got := ValidChangedPath(tc.game, tc.rel); got != tc.want {
			t.Errorf("ValidChangedPath(%q, %q) = %v, want %v", tc.game, tc.rel, got, tc.want)
		}
	}
}
