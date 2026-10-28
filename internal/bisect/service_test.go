package bisect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestServiceRefusals(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "Empty")
	s := NewService(profiles, nil)

	if _, err := s.Start(t.Context(), "", p.ID); err == nil {
		t.Error("empty game accepted")
	}
	if _, err := s.Start(t.Context(), "stardew", ""); err == nil {
		t.Error("empty profile id accepted")
	}
	if _, err := s.Start(t.Context(), "stardew", p.ID); err == nil || !strings.Contains(err.Error(), "no enabled user mods") {
		t.Errorf("profile without mods: err = %v", err)
	}
	dir, err := profiles.ProfileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(filepath.Dir(dir))
	if err != nil || len(ents) != 1 || ents[0].Name() != p.ID {
		t.Errorf("profile folders after a refused start: %v %v", ents, err)
	}

	if _, err := s.Status("bisect-9"); err == nil {
		t.Error("Status of an unknown id succeeded")
	}
	if err := s.Stop("bisect-9"); err == nil {
		t.Error("Stop of an unknown id succeeded")
	}
	if err := s.SwitchOff("stardew", p.ID, "", ""); err == nil {
		t.Error("SwitchOff with no key and no id succeeded")
	}
}
