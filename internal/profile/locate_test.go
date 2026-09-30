package profile

import "testing"

func TestProfilesWithModReadsProfileJSON(t *testing.T) {
	s := newStore(t)
	farm, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	coop, err := s.Create("stardew", "Co-op")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("stardew", "Empty"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.update("stardew", farm.ID, func(p *Profile, _ string) error {
		p.Entries = []Entry{{
			Key: "nexus-1-1",
			Mods: []EntryMod{
				{UniqueID: "Pathoschild.ContentPatcher", Name: "Content Patcher", Version: "2.1.0"},
			},
			Disabled: []string{"Pathoschild.ContentPatcher"},
		}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.update("stardew", coop.ID, func(p *Profile, _ string) error {
		p.Entries = []Entry{{
			Key: "nexus-1-2",
			Mods: []EntryMod{
				{UniqueID: "pathoschild.contentpatcher", Name: "Content Patcher", Version: "2.0.0"},
			},
		}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	got, err := s.ProfilesWithMod("stardew", "Pathoschild.ContentPatcher")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].ProfileID != farm.ID || got[0].Version != "2.1.0" || got[0].Enabled {
		t.Fatalf("farm = %+v", got[0])
	}
	if got[1].ProfileID != coop.ID || got[1].Version != "2.0.0" || !got[1].Enabled {
		t.Fatalf("coop = %+v", got[1])
	}

	none, err := s.ProfilesWithMod("stardew", "Nope.Mod")
	if err != nil || len(none) != 0 {
		t.Fatalf("missing = %+v, %v", none, err)
	}
	blank, err := s.ProfilesWithMod("stardew", "")
	if err != nil || len(blank) != 0 {
		t.Fatalf("blank = %+v, %v", blank, err)
	}
}
