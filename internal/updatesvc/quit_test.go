package updatesvc

import (
	"context"
	"testing"
)

func TestApplyOnQuitNeverReopensMortar(t *testing.T) {
	f := &fake{}
	s := &Service{u: f, info: Info{Version: "1.0.0"}, found: &Release{Version: "1.1.0", Staged: true}}
	if err := s.ApplyOnQuit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !f.appliedOnExit || f.restarted {
		t.Fatalf("appliedOnExit=%v restarted=%v; quitting must swap without relaunching", f.appliedOnExit, f.restarted)
	}
}
