package browse

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/meta"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestMergeSameFoldsHitsThatLinkEachOther(t *testing.T) {
	items := []Item{
		{Source: "nexus", ID: "1", Name: "Cool Mod", Author: "Alice"},
		{Source: "thunderstore", ID: "Alice-CoolMod", Name: "CoolMod", Author: "alice", Repo: "alice/cool"},
		{Source: "github", ID: "alice/cool", Name: "alice/cool", Author: "alice", Repo: "alice/cool", Installed: true},
		{Source: "github", ID: "bob/other", Name: "other", Author: "bob", Repo: "bob/other"},
	}
	got := mergeSame(items, []string{"thunderstore", "nexus", "github"}, nil)
	if len(got) != 3 {
		t.Fatalf("cards = %+v", got)
	}
	card := got[1]
	if card.Source != "thunderstore" || len(card.Alts) != 1 || card.Alts[0].Source != "github" || !card.Alts[0].Installed {
		t.Errorf("merged card = %+v", card)
	}
}

func TestMergeSameNeverMergesOnNames(t *testing.T) {
	got := mergeSame([]Item{
		{Source: "nexus", ID: "1", Name: "Radar Map", Author: "same"},
		{Source: "github", ID: "x/radar-map", Name: "Radar Map", Author: "same"},
		{Source: "thunderstore", ID: "A-RadarMap", Name: "RadarMap", Author: "A"},
	}, []string{"thunderstore", "nexus", "github"}, nil)
	if len(got) != 3 {
		t.Fatalf("cards = %+v", got)
	}
}

func TestMergeSameTiesHitsThatShareAPackageIdentity(t *testing.T) {
	ident := func(it Item) string {
		switch it.ID {
		case "7", "pathoschild/cool":
			return "smapi:pathoschild.cool"
		case "9":
			return "smapi:other.mod"
		}
		return ""
	}
	got := mergeSame([]Item{
		{Source: "nexus", ID: "7", Name: "Cool"},
		{Source: "github", ID: "pathoschild/cool", Name: "pathoschild/cool"},
		{Source: "nexus", ID: "9", Name: "Cool"},
		{Source: "github", ID: "nobody/unknown", Name: "Cool"},
	}, []string{"nexus", "github"}, ident)
	if len(got) != 3 || len(got[0].Alts) != 1 || got[0].Alts[0].ID != "pathoschild/cool" {
		t.Fatalf("cards = %+v", got)
	}
}

func TestMergeSameTiesTheLoader(t *testing.T) {
	got := mergeSame([]Item{
		{Source: "thunderstore", ID: "BepInEx-BepInExPack", Name: "BepInExPack", Author: "BepInEx", Loader: true},
		{Source: "nexus", ID: "1", Name: "BepInEx", Author: "u", Loader: true, Installed: true, Obsolete: true, Broken: true, Endorsements: 7, Downloads: 99},
	}, []string{"thunderstore", "nexus"}, nil)
	if len(got) != 1 || got[0].Source != "thunderstore" || !got[0].Loader || got[0].Obsolete || got[0].Broken {
		t.Fatalf("card = %+v", got)
	}
	if alt := got[0].Alts[0]; !alt.Loader || !alt.Installed || !alt.Obsolete || !alt.Broken || alt.Endorsements != 7 || alt.Downloads != 99 {
		t.Fatalf("alt = %+v", alt)
	}
}

func TestMergeSameTiesBySummaryLinks(t *testing.T) {
	got := mergeSame([]Item{
		{Source: "nexus", ID: "3", Name: "Odd Title", Summary: "Source at https://github.com/Alice/Cool-Mod and more"},
		{Source: "github", ID: "alice/cool-mod", Name: "other", Repo: "alice/cool-mod"},
		{Source: "nexus", ID: "4", Name: "Another", Summary: "See https://thunderstore.io/c/lethal-company/p/Bob/Thing/"},
		{Source: "thunderstore", ID: "Bob-Thing", Name: "Something"},
	}, []string{"thunderstore", "nexus", "github"}, nil)
	if len(got) != 2 {
		t.Fatalf("cards = %+v", got)
	}
}

func TestRankedPutsPreferredSourcesFirst(t *testing.T) {
	got := ranked([]string{"github", "bogus", "github", "nexus"}, []string{"nexus", "thunderstore", "github"})
	want := []string{"github", "nexus", "thunderstore"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("ranked = %v, want %v", got, want)
	}
}

func TestHoldingsGiveAnIdentityToTheRefsOfWhatTheProfileHolds(t *testing.T) {
	prof := profile.Profile{Entries: []profile.Entry{
		{Source: profile.Source{Kind: profile.KindNexus, ModID: 42}, Mods: []profile.Component{{ID: mod.SMAPI("Me.Cool")}}},
		{Source: profile.Source{Kind: profile.KindNexus, ModID: 50}, Mods: []profile.Component{{ID: mod.SMAPI("A.One")}, {ID: mod.SMAPI("A.Two")}}},
	}}
	installed := []profile.Installed{{UniqueID: "Me.Cool", UpdateKeys: []string{"GitHub:Me/Cool"}}}
	h := Hold(components.GameInfo{}, prof, installed)
	if h.Identity("nexus", "42") != "smapi:me.cool" || h.Identity("github", "me/cool") != "smapi:me.cool" {
		t.Fatalf("ids = %v", h.ids)
	}
	if h.Identity("nexus", "50") != "" {
		t.Fatal("an entry of two mods names no single identity")
	}
}

func TestCompatListTiesANexusAndAGitHubHitThroughTheirUniqueID(t *testing.T) {
	c := &Client{Compat: func(context.Context) (meta.CompatIndex, error) {
		return meta.CompatIndex{
			NexusToID:  map[int]string{7: "Pathoschild.Cool"},
			GitHubToID: map[string]string{"pathoschild/cool": "Pathoschild.Cool"},
		}, nil
	}}
	got := mergeSame([]Item{
		{Source: "nexus", ID: "7", Name: "Cool"},
		{Source: "github", ID: "Pathoschild/Cool", Name: "Pathoschild/Cool"},
		{Source: "github", ID: "someone/else", Name: "Cool"},
	}, []string{"nexus", "github"}, c.identity(t.Context()))
	if len(got) != 2 || len(got[0].Alts) != 1 {
		t.Fatalf("cards = %+v", got)
	}
}
