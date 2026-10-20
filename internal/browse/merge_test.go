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
	got := mergeSame(items, []string{"thunderstore", "nexus", "github"}, nil, nil)
	// Nexus's "Cool Mod" by Alice joins through name and author, GitHub's through the repository.
	if len(got) != 2 {
		t.Fatalf("cards = %+v", got)
	}
	card := got[0]
	if card.Source != "thunderstore" || len(card.Alts) != 2 || card.Alts[1].Source != "github" || !card.Alts[1].Installed {
		t.Errorf("merged card = %+v", card)
	}
}

func TestMergeSameTiesANameOnlyThroughItsAuthor(t *testing.T) {
	got := mergeSame([]Item{
		{Source: "thunderstore", ID: "notnotnotswipez-MoreCompany", Name: "MoreCompany", Author: "notnotnotswipez", Downloads: 27_000_000},
		{Source: "nexus", ID: "1", Name: "More Company", Author: "Cyb3rdev", Downloads: 175_000},
		{Source: "curseforge", ID: "2", Name: "Lethal Level Loader", Author: "IAmBatby"},
		{Source: "thunderstore", ID: "IAmBatby-LethalLevelLoader", Name: "LethalLevelLoader", Author: "IAmBatby"},
		{Source: "github", ID: "EvaisaDev/LethalLib", Name: "LethalLib", Author: "EvaisaDev"},
		{Source: "thunderstore", ID: "Evaisa-LethalLib", Name: "LethalLib", Author: "Evaisa"},
	}, []string{"thunderstore", "nexus", "curseforge", "github"}, nil,
		aliasesOf([]components.AuthorAliases{{Names: []string{"Evaisa", "EvaisaDev"}}}))
	cards := map[string]Item{}
	for _, c := range got {
		cards[c.Source+":"+c.Name] = c
	}
	if len(got) != 4 || len(cards["thunderstore:LethalLevelLoader"].Alts) != 1 || len(cards["thunderstore:LethalLib"].Alts) != 1 {
		t.Fatalf("cards = %+v", got)
	}
	if cards["nexus:More Company"].ReuploadOf != "MoreCompany by notnotnotswipez" || cards["thunderstore:MoreCompany"].ReuploadOf != "" {
		t.Errorf("re-upload marks = %+v", got)
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
	}, []string{"nexus", "github"}, ident, nil)
	if len(got) != 3 || len(got[0].Alts) != 1 || got[0].Alts[0].ID != "pathoschild/cool" {
		t.Fatalf("cards = %+v", got)
	}
}

func TestMergeSameTiesBySummaryLinks(t *testing.T) {
	got := mergeSame([]Item{
		{Source: "nexus", ID: "3", Name: "Odd Title", Summary: "Source at https://github.com/Alice/Cool-Mod and more"},
		{Source: "github", ID: "alice/cool-mod", Name: "other", Repo: "alice/cool-mod"},
		{Source: "nexus", ID: "4", Name: "Another", Summary: "See https://thunderstore.io/c/lethal-company/p/Bob/Thing/"},
		{Source: "thunderstore", ID: "Bob-Thing", Name: "Something"},
	}, []string{"thunderstore", "nexus", "github"}, nil, nil)
	if len(got) != 2 {
		t.Fatalf("cards = %+v", got)
	}
}

func TestSummaryLinksNeedTheRealHost(t *testing.T) {
	for _, summary := range []string{
		"https://evilgithub.com/Alice/Cool-Mod", "https://github.com.evil.example/Alice/Cool-Mod", "github.com/Alice/Cool-Mod",
		"https://notthunderstore.io/c/lethal-company/p/Bob/Thing/", "https://thunderstore.io.evil.example/c/x/p/Bob/Thing/",
	} {
		if got := keys(Item{Source: "nexus", ID: "1", Summary: summary}, nil, nil); len(got) != 0 {
			t.Errorf("%s gave %v", summary, got)
		}
	}
	if got := keys(Item{Source: "nexus", Summary: "at https://www.github.com/Alice/Cool-Mod and more"}, nil, nil); len(got) != 1 || got[0] != "repo:alice/cool-mod" {
		t.Errorf("www link gave %v", got)
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
	}, []string{"nexus", "github"}, c.identity(t.Context()), nil)
	if len(got) != 2 || len(got[0].Alts) != 1 {
		t.Fatalf("cards = %+v", got)
	}
}

func TestKeysDropTheFullStopAfterAGitHubLink(t *testing.T) {
	for _, summary := range []string{
		"Source: https://github.com/Alice/Cool-Mod.",
		"See https://github.com/alice/cool-mod.git.",
		"https://github.com/alice/cool-mod...",
	} {
		got := keys(Item{Summary: summary}, nil, nil)
		if len(got) != 1 || got[0] != "repo:alice/cool-mod" {
			t.Errorf("keys(%q) = %v", summary, got)
		}
	}
}

func TestLinksResolveOnlyOnTheirOwnHost(t *testing.T) {
	t.Parallel()
	repo := map[string]string{
		"See https://github.com/Alice/Cool for source.":                  "Alice/Cool",
		"[src](http://www.GitHub.com/a/b)":                               "a/b",
		"ends a sentence https://github.com/a/b.":                        "a/b.",
		"https://evil.example/?next=//github.com/a/b":                    "",
		"https://evil.example/https://github.com/a/b":                    "",
		"https://evilgithub.com/a/b":                                     "",
		"https://github.com.evil.example/a/b":                            "",
		"https://api.github.com/a/b":                                     "",
		"https://user@evil.example/github.com/a/b":                       "",
		"//github.com/a/b":                                               "",
		"https://github.com/onlyowner":                                   "",
		"ftp://github.com/a/b":                                           "",
		"https://evil.example/?x=1 then https://github.com/real/repo ok": "real/repo",
	}
	for text, want := range repo {
		if got := githubRepoIn(text); got != want {
			t.Errorf("githubRepoIn(%q) = %q, want %q", text, got, want)
		}
	}
	ts := map[string]string{
		"https://thunderstore.io/c/lethal-company/p/Ns/Name/":    "Ns-Name",
		"http://www.thunderstore.io/c/valheim/p/A_b/C9":          "A_b-C9",
		"https://evil.example/?next=//thunderstore.io/c/x/p/A/B": "",
		"https://evilthunderstore.io/c/x/p/A/B":                  "",
		"https://thunderstore.io.evil.example/c/x/p/A/B":         "",
		"https://thunderstore.io/c/x/q/A/B":                      "",
		"https://thunderstore.io/c/x/p/A":                        "",
	}
	for text, want := range ts {
		if got := thunderstoreRefIn(text); got != want {
			t.Errorf("thunderstoreRefIn(%q) = %q, want %q", text, got, want)
		}
	}
}
