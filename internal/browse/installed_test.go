package browse

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func entry(src profile.Source) profile.Entry { return profile.Entry{Source: src} }

func TestHoldingsMatchEntrySourceUpdateKeyAndBundledCompanion(t *testing.T) {
	info, ok := catalogGame("stardew")
	if !ok {
		t.Fatal("no stardew in the catalog")
	}
	prof := profile.Profile{Entries: []profile.Entry{
		entry(profile.Source{Kind: profile.KindGitHub, Repo: "Owner/Mod"}),
		entry(profile.Source{Kind: profile.KindNexus, ModID: 5}),
		entry(profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Pack"}),
		entry(profile.Source{Kind: profile.SourceMortar}),
	}}
	var keyed profile.Installed
	keyed.UpdateKeys = []string{"GitHub:Other/Keyed", "Nexus:99@3"}
	h := Hold(info, prof, []profile.Installed{keyed})
	for _, c := range [][2]string{
		{"github", "owner/mod"},                        // entry source, case-insensitive
		{"github", "other/keyed"},                      // manifest update key
		{"nexus", "5"},                                 // entry source
		{"nexus", "99"},                                // manifest update key
		{"thunderstore", "ns-pack"},                    // entry source
		{"github", "rethunk-tech/mortar-smapi-bridge"}, // bundled companion
	} {
		if !h.Has(c[0], c[1]) {
			t.Errorf("%v not held", c)
		}
	}
	if h.Has("github", "nobody/none") || h.Has("nexus", "6") || h.Has("nexus", "x") {
		t.Error("an unheld mod matched")
	}
	if !h.Bundled("github", "Rethunk-Tech/mortar-smapi-bridge") || h.Bundled("github", "owner/mod") {
		t.Error("only the companion is bundled")
	}
	if Hold(info, profile.Profile{}, nil).Has("github", "rethunk-tech/mortar-smapi-bridge") {
		t.Error("a profile without the bundled bridge holds it")
	}
}
