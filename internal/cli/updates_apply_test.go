package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestUpdatesApplyEverywhere(t *testing.T) {
	results := map[string]any{
		"updates.apply": profile.EverywhereResult{
			Updated: []profile.EverywhereHit{{ProfileID: "a", Name: "A", OldKey: "a-1"}},
			Skipped: []profile.EverywhereSkip{{ProfileID: "b", Name: "B", Reason: "pinned"}},
		},
	}
	r := invoke(t, results, "updates", "apply", "--everywhere", "stardew", "me.a")
	if r.code != 0 {
		t.Fatalf("code %d stderr %q", r.code, r.errOut)
	}
	if len(r.calls) != 1 || r.calls[0].method != "updates.apply" {
		t.Fatalf("calls %+v", r.calls)
	}
	c := r.calls[0]
	if c.params.Game != "stardew" || len(c.params.IDs) != 1 || c.params.IDs[0] != "me.a" {
		t.Fatalf("params %+v", c.params)
	}
	for _, want := range []string{"A", "updated", "B", "pinned"} {
		if !strings.Contains(r.out, want) {
			t.Fatalf("out %q, want %q", r.out, want)
		}
	}
}
