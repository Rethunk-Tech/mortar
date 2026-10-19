package configsvc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHitsInMatchesKeyLabelDescriptionAndSectionName(t *testing.T) {
	sc := Schema{File: ConfigFile{Name: "a.cfg"}, Sections: []Section{
		{Name: "Combat", Entries: []Entry{{Key: "Damage", Value: "1"}, {Key: "Speed", Label: "Move rate", Description: "How fast you run"}}},
		{Name: "Audio", Entries: []Entry{{Key: "Volume"}}},
	}}
	for query, want := range map[string]int{"damage": 1, "move": 1, "fast": 1, "combat": 2, "volume": 1, "zzz": 0} {
		if got := hitsIn(query, "Me.A", "A", sc); len(got) != want {
			t.Errorf("%q: %d hits, want %d: %+v", query, len(got), want, got)
		}
	}
	got := hitsIn("damage", "Me.A", "A", sc)
	if got[0].File != "a.cfg" || got[0].Section != "Combat" || got[0].ModName != "A" || got[0].Value != "1" {
		t.Errorf("hit = %+v", got[0])
	}
}

func TestSearchCoversUnownedCfgAndTextFiles(t *testing.T) {
	s, _ := newService(t)
	f, ok := s.Profiles.(fakeProfiles)
	if !ok {
		t.Fatal("newService builds a fakeProfiles")
	}
	cfg := filepath.Join(f.dir, "BepInEx", "config")
	if err := os.WriteFile(filepath.Join(cfg, "BepInEx.cfg"), []byte("[Logging]\n## Enable\n# Setting type: Boolean\n# Default value: true\nEnabled = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "notes.txt"), []byte("first\nTweak the zebra here\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.Search("g", "p", "enabled")
	if err != nil || len(got) != 1 || got[0].File != "BepInEx.cfg" || got[0].ModID != "" {
		t.Fatalf("unowned cfg: %+v, %v", got, err)
	}
	got, err = s.Search("g", "p", "ZEBRA")
	if err != nil || len(got) != 1 || got[0].File != "notes.txt" || got[0].Key != "line 2" {
		t.Fatalf("text file: %+v, %v", got, err)
	}
}
