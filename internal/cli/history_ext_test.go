package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestHistoryDiffAndRevertCLI(t *testing.T) {
	results := map[string]any{
		"history.diff": profile.HistoryDiff{
			Items: []profile.HistoryItem{{Kind: "added", Name: "Alpha", Detail: "added Alpha"}},
		},
		"history.revert": profile.Profile{ID: "p1", Name: "Farm"},
	}
	r := invoke(t, results, "history", "diff", "stardew", "Farm", "a", "b")
	if r.code != 0 || !strings.Contains(r.out, "added Alpha") {
		t.Fatalf("history diff: %+v", r)
	}
	if got := r.calls[0]; got.method != "history.diff" || got.params.Name != "a" || got.params.Value != "b" {
		t.Fatalf("history diff params: %+v", got)
	}
	r = invoke(t, results, "history", "revert", "stardew", "Farm", "evt", "--item", "Me.A")
	if r.code != 0 {
		t.Fatalf("history revert: %+v", r)
	}
	if got := r.calls[0]; got.method != "history.revert" || got.params.Value != "Me.A" {
		t.Fatalf("history revert params: %+v", got)
	}
}

func TestProfileChangesAndGoodCLI(t *testing.T) {
	results := map[string]any{
		"profile.changes": profile.HistoryDiff{
			Items: []profile.HistoryItem{{Kind: "added", Name: "Beta", Detail: "added Beta"}},
		},
		"profile.good": profile.HistoryEvent{ID: "g1", Label: "Known good"},
	}
	r := invoke(t, results, "profile", "changes", "stardew", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "added Beta") {
		t.Fatalf("profile changes: %+v", r)
	}
	r = invoke(t, results, "profile", "good", "stardew", "Farm", "--mark")
	if r.code != 0 || r.calls[0].method != "profile.good" || !r.calls[0].params.All {
		t.Fatalf("profile good --mark: %+v", r)
	}
}

func TestPlayCheckIgnoresChangesForExit(t *testing.T) {
	results := map[string]any{
		"play.check": []control.PlayIssueGroup{{Kind: "changes", Count: 1, Names: []string{"added Alpha"}}},
	}
	r := invoke(t, results, "play", "stardew", "Farm", "--check")
	if r.code != 0 {
		t.Fatalf("play --check with only changes: %+v", r)
	}
}
