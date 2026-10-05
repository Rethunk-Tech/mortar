package profile

import "testing"

func TestModsByAuthorGroupsAcrossProfiles(t *testing.T) {
	s := newStore(t)
	game := "stardew"
	farm, err := s.Create(game, "Farm")
	if err != nil {
		t.Fatal(err)
	}
	coop, err := s.Create(game, "Co-op")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.update(game, farm.ID, func(p *Profile, _ string) error {
		p.Entries = []Entry{{
			Key: "k1",
			Mods: []Component{{
				ID:      "smapi:Author.One",
				Name:    "One",
				Version: "1.0",
				Author:  "Pathoschild, Helper",
			}},
		}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.update(game, coop.ID, func(p *Profile, _ string) error {
		p.Entries = []Entry{{
			Key: "k2",
			Mods: []Component{{
				ID:      "smapi:Author.Two",
				Name:    "Two",
				Version: "2.0",
				Author:  "pathoschild",
			}},
		}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ModsByAuthor(game, "Pathoschild")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("mods: %#v", got)
	}
	if len(got[0].Profiles) != 1 || len(got[1].Profiles) != 1 {
		t.Fatalf("profiles per mod: %#v", got)
	}
	none, err := s.ModsByAuthor(game, "Nobody")
	if err != nil || len(none) != 0 {
		t.Fatalf("none: %v %#v", err, none)
	}
}
