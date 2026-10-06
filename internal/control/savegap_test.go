package control

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
)

func TestSavesCheckTakesAProfileByName(t *testing.T) {
	s := services(t)
	sv, err := savessvc.NewService(t.TempDir(), s.Store, s.Settings, &meta.Client{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	s.Saves = sv
	created, err := s.Handle(t.Context(), "profile.create", Params{Game: "stardew", Name: "Spring Farm"})
	if err != nil {
		t.Fatal(err)
	}
	p, ok := created.(profile.Profile)
	if !ok {
		t.Fatalf("create returned %T", created)
	}
	sv.NotePlayed("stardew", p.ID, "Farm_1")
	if _, err := s.Handle(t.Context(), "saves.check", Params{Game: "stardew", Name: "Farm_1", Profile: "Spring Farm"}); err != nil {
		t.Fatalf("a profile named as on every other verb was refused: %v", err)
	}
}

func TestSaveGapWordingAndGroup(t *testing.T) {
	msg := launchWarningError{gap: true, save: savessvc.Fit{Folder: "Farm_1"}}.Error()
	if strings.Contains(msg, "'s farm") || !strings.Contains(msg, "The farm (Farm_1)") {
		t.Errorf("a save with no farmer name reads %q", msg)
	}
	g := saveGapGroup(savessvc.Fit{Missing: []savessvc.Lack{{Name: "File"}}, LastMissing: []savessvc.Lack{{Name: "A"}, {Name: "B"}}})
	if g.Kind != "save" || g.Count != 2 || strings.Join(g.Names, ",") != "A,B" {
		t.Errorf("save group %+v", g)
	}
}
