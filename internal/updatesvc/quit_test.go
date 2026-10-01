package updatesvc

import (
	"context"
	"testing"
)

func TestShouldApplyOnQuit(t *testing.T) {
	s := &Service{info: Info{Version: "1.0.0"}, found: &Release{Version: "1.1.0", Staged: true}}
	if !s.ShouldApplyOnQuit() {
		t.Fatal("staged update should apply on quit")
	}
	s.restartChosen = true
	if s.ShouldApplyOnQuit() {
		t.Fatal("restart already chosen")
	}
	s.restartChosen = false
	s.found.Staged = false
	if s.ShouldApplyOnQuit() {
		t.Fatal("not staged")
	}
	s.info.Off = "packaged"
	s.found.Staged = true
	if s.ShouldApplyOnQuit() {
		t.Fatal("packaged build")
	}
}

func TestApplyOnQuitCallsRestart(t *testing.T) {
	f := &fake{}
	s := &Service{u: f, info: Info{Version: "1.0.0"}, found: &Release{Version: "1.1.0", Staged: true}}
	if err := s.ApplyOnQuit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !f.restarted || !s.restartChosen {
		t.Fatalf("restart=%v chosen=%v", f.restarted, s.restartChosen)
	}
}
