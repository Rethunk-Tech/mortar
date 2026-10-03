package problems

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

func TestDriftChecksOffSkipsScan(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(s *settings.Settings) { v := false; s.DriftChecks = &v }); err != nil {
		t.Fatal(err)
	}
	s := &Service{settings: set}
	got, err := s.withDrift("stardew", "p", Result{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Drift == nil || len(got.Drift) != 0 {
		t.Fatalf("drift = %+v", got.Drift)
	}
}
