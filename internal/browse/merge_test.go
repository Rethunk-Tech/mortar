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

func TestMergeSameTiesTheLoaderAndExactNamesAcrossSources(t *testing.T) {
	items := []Item{
		{Source: "thunderstore", ID: "BepInEx-BepInExPack", Name: "BepInExPack", Author: "BepInEx", Loader: true},
		{Source: "nexus", ID: "1", Name: "BepInEx", Author: "someuploader", Loader: true},
		{Source: "thunderstore", ID: "notnotnotswipez-MoreCompany", Name: "MoreCompany", Author: "notnotnotswipez"},
		{Source: "nexus", ID: "2", Name: "More Company", Author: "Cyb3rdev"},
	}
	got := mergeSame(items, []string{"thunderstore", "nexus"})
	if len(got) != 2 || len(got[0].Alts) != 1 || got[0].Alts[0].Source != "nexus" || len(got[1].Alts) != 1 || got[1].Source != "thunderstore" {
		t.Fatalf("cards = %+v", got)
	}
}

func TestMergeSameTiesBySummaryLinks(t *testing.T) {
	got := mergeSame([]Item{
		{Source: "nexus", ID: "3", Name: "Odd Title", Author: "u", Summary: "Source at https://github.com/Alice/Cool-Mod and more"},
		{Source: "github", ID: "alice/cool-mod", Name: "other", Author: "x", Repo: "alice/cool-mod"},
		{Source: "nexus", ID: "4", Name: "Another", Author: "u", Summary: "See https://thunderstore.io/c/lethal-company/p/Bob/Thing/"},
		{Source: "thunderstore", ID: "Bob-Thing", Name: "Something", Author: "Bob"},
	}, []string{"thunderstore", "nexus", "github"})
	if len(got) != 2 {
		t.Fatalf("cards = %+v", got)
	}
}

func TestMergeSameKeepsTwoModsOfOneSourceApart(t *testing.T) {
	got := mergeSame([]Item{
		{Source: "thunderstore", ID: "A-Radar", Name: "Radar", Author: "A"},
		{Source: "thunderstore", ID: "B-Radar", Name: "Radar", Author: "B"},
		{Source: "nexus", ID: "9", Name: "Radar", Author: "u"},
		{Source: "nexus", ID: "10", Name: "Maps", Author: "u"},
		{Source: "thunderstore", ID: "C-Maps", Name: "Maps", Author: "C"},
	}, []string{"thunderstore", "nexus"})
	if len(got) != 3 {
		t.Fatalf("cards = %+v", got)
	}
}
