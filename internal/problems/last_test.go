package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// A completed check is stored for Last; Last answers from the data folder, with no scan, even from a new Service
// (a restart), and says nothing for a profile that was never checked.
func TestLastIsTheStoredCompletedCheck(t *testing.T) {
	set, profiles := testenv.Env(t)
	p := testenv.Profile(t, profiles, "stardew", "A")
	home := t.TempDir()
	s := NewService(home, set, profiles, nil)
	s.meta = fakeMeta{}
	if got := s.Last("stardew", p.ID); got != nil {
		t.Fatalf("before any check: %+v", got)
	}
	want, err := s.ProblemsNoWait(t.Context(), "stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewService(home, set, profiles, nil)
	got := restarted.Last("stardew", p.ID)
	if got == nil || got.At == 0 || len(got.Result.Broken) != len(want.Broken) || got.Result.Unknown != want.Unknown {
		t.Fatalf("Last = %+v, want the stored check %+v", got, want)
	}
	if restarted.Last("stardew", "../escape") != nil || restarted.Last("stardew", "other") != nil {
		t.Fatal("Last answered for an id that was never checked or that leaves the cache folder")
	}
}
