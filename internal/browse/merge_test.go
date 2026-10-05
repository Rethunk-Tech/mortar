package browse

import "testing"

func TestMergeSameFoldsOneModAcrossSources(t *testing.T) {
	items := []Item{
		{Source: "nexus", ID: "1", Name: "Cool Mod", Author: "Alice"},
		{Source: "thunderstore", ID: "Alice-CoolMod", Name: "CoolMod", Author: "alice", Repo: "alice/cool"},
		{Source: "github", ID: "alice/cool", Name: "alice/cool", Author: "alice", Repo: "alice/cool", Installed: true},
		{Source: "github", ID: "bob/other", Name: "other", Author: "bob", Repo: "bob/other"},
	}
	got := mergeSame(items, []string{"thunderstore", "nexus", "github"})
	if len(got) != 2 {
		t.Fatalf("cards = %+v", got)
	}
	card := got[0]
	if card.Source != "thunderstore" || len(card.Alts) != 2 || !card.Alts[1].Installed {
		t.Errorf("merged card = %+v", card)
	}
	if got[1].ID != "bob/other" || len(got[1].Alts) != 0 {
		t.Errorf("unrelated card = %+v", got[1])
	}
}

func TestMergeSameKeepsSameNameDifferentAuthorApart(t *testing.T) {
	got := mergeSame([]Item{
		{Source: "nexus", Name: "Map", Author: "a"},
		{Source: "github", Name: "Map", Author: "b"},
	}, []string{"nexus", "github"})
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
