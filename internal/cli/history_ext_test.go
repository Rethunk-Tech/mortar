package cli

import (
	"strings"
	"testing"
	"time"

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
		"profile.good": profile.HistoryEvent{ID: "g1", Change: profile.ChangeKnownGood},
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

func TestHistorySummaryWordsEachChange(t *testing.T) {
	at := time.Date(2026, 10, 6, 4, 48, 44, 0, time.UTC)
	for _, c := range []struct {
		ev   profile.HistoryEvent
		want string
	}{
		{profile.HistoryEvent{Change: profile.ChangeDisabled, Name: "Seed Alpha"}, "Disabled Seed Alpha"},
		{profile.HistoryEvent{Change: profile.ChangeUpdated, Name: "Beta", From: "1.0", To: "1.1"}, "Updated Beta from 1.0 to 1.1"},
		{profile.HistoryEvent{Change: profile.ChangeMods, Count: 3}, "Changed 3 mods"},
		{profile.HistoryEvent{Change: profile.ChangeMods, Count: 1}, "Changed mods"},
		{profile.HistoryEvent{Change: profile.ChangeImported, Count: 1}, "Imported 1 mod"},
		{profile.HistoryEvent{Change: profile.ChangeOptionSet, Name: "demo.Mod", Detail: "Count"}, "Set Count of demo.Mod for the next start"},
		{profile.HistoryEvent{Change: profile.ChangeReverted, Target: at}, "Went back to " + at.Local().Format("2006-01-02 15:04")},
		{profile.HistoryEvent{}, "Changed the profile"},
	} {
		if got := historySummary(c.ev); got != c.want {
			t.Errorf("historySummary(%+v) = %q, want %q", c.ev, got, c.want)
		}
	}
}
