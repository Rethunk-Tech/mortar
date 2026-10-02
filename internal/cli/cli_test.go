package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

type call struct {
	method string
	params control.Params
}

// fake answers each method with a canned JSON result and records what was asked.
func fake(results map[string]any, calls *[]call) caller {
	return func(method string, p control.Params, out any, _ time.Duration) error {
		*calls = append(*calls, call{method, p})
		b, err := json.Marshal(results[method])
		if err != nil {
			return err
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(b, out)
	}
}

type outcome struct {
	code        int
	out, errOut string
	calls       []call
}

func invoke(t *testing.T, results map[string]any, args ...string) outcome {
	t.Helper()
	var o, e bytes.Buffer
	var calls []call
	code := run("9.9.9", fake(results, &calls), args, &o, &e)
	return outcome{code, o.String(), e.String(), calls}
}

func TestUsageErrorsExitTwo(t *testing.T) {
	for _, args := range [][]string{{"mods"}, {"profiles"}, {"games", "--bogus"}, {"profile", "rename", "stardew", "p"}, {"logs", "stardew", "p", "--run"}} {
		r := invoke(t, nil, args...)
		if r.code != 2 || !strings.Contains(r.errOut, "Usage:") || len(r.calls) != 0 {
			t.Errorf("%v: code %d, calls %v, stderr %q", args, r.code, r.calls, r.errOut)
		}
	}
	if r := invoke(t, nil, "help"); r.code != 0 || !strings.Contains(r.out, "Usage:") {
		t.Errorf("help: %d %q", r.code, r.out)
	}
}

func TestCommandsSendTheirArguments(t *testing.T) {
	results := map[string]any{
		"profile.create": profile.Profile{ID: "abc", Name: "My Farm"},
		"mods.disable":   profile.EnableResult{},
		"mods": []control.ModRow{
			{UniqueID: "A.Mod", Name: "Alpha", Version: "1.0", Enabled: true, Source: "nexus:1/2"},
			{UniqueID: "B.Mod", Name: "Beta", Version: "2.0", Pinned: true, Source: "local"},
		},
	}
	r := invoke(t, results, "profile", "create", "stardew", "My", "Farm")
	if r.code != 0 || r.calls[0].method != "profile.create" || r.calls[0].params.Name != "My Farm" || !strings.Contains(r.out, "abc") {
		t.Fatalf("create: %+v", r)
	}
	r = invoke(t, results, "mods", "disable", "stardew", "My Farm", "A.Mod", "B.Mod")
	if c := r.calls[0]; c.method != "mods.disable" || c.params.Profile != "My Farm" || len(c.params.UniqueIDs) != 2 || !strings.HasPrefix(r.out, "Disabled ") {
		t.Fatalf("disable: %+v", r)
	}
	r = invoke(t, results, "mods", "stardew", "abc")
	if r.code != 0 || !strings.Contains(r.out, "UNIQUEID") || !strings.Contains(r.out, "off, pinned") {
		t.Fatalf("mods table: %q", r.out)
	}
	r = invoke(t, results, "mods", "stardew", "abc", "--json")
	var rows []control.ModRow
	if err := json.Unmarshal([]byte(r.out), &rows); err != nil || len(rows) != 2 {
		t.Fatalf("--json: %v %q", err, r.out)
	}
	r = invoke(t, map[string]any{"conflicts": []any{}}, "conflicts", "stardew", "abc", "--all")
	if !r.calls[0].params.All {
		t.Error("--all not sent")
	}
}

func TestProfileCompareHistoryAndRevert(t *testing.T) {
	results := map[string]any{
		"profile.compare": profile.CLICompare{
			OnlyA: []profile.DiffSide{{UniqueID: "A.Mod", Name: "Alpha", Version: "1", Enabled: true}},
		},
		"profile.history": []control.HistoryRow{{ID: "event-1", Kind: "added", Summary: "Added Alpha"}},
		"profile.revert":  profile.Profile{ID: "profile-1", Name: "Farm"},
	}
	r := invoke(t, results, "profile", "compare", "stardew", "A", "B")
	if r.code != 0 || !strings.Contains(r.out, "only-in-A") {
		t.Fatalf("compare: %+v", r)
	}
	if got := r.calls[0]; got.method != "profile.compare" || got.params.Profile != "A" || got.params.Name != "B" {
		t.Fatalf("compare params: %+v", got)
	}
	r = invoke(t, results, "profile", "history", "stardew", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "event-1") {
		t.Fatalf("history: %+v", r)
	}
	r = invoke(t, results, "profile", "revert", "stardew", "Farm", "event-1")
	if r.code != 0 || r.calls[0].method != "profile.revert" || r.calls[0].params.Name != "event-1" {
		t.Fatalf("revert: %+v", r)
	}
}

func TestProfileMatchSendsPreviewRequest(t *testing.T) {
	r := invoke(t, map[string]any{"profile.match": control.ProfileMatch{Already: 2, Missing: []string{"Missing"}}},
		"profile", "match", "stardew", "Farm", "mortar://example")
	if r.code != 0 || !strings.Contains(r.out, "2 mods already match") {
		t.Fatalf("match: %+v", r)
	}
	if got := r.calls[0]; got.method != "profile.match" || got.params.Path != "mortar://example" {
		t.Fatalf("match params: %+v", got)
	}
}

func TestNexusUntrackRequiresScope(t *testing.T) {
	r := invoke(t, nil, "nexus", "untrack", "stardew", "--yes")
	if r.code != 2 || len(r.calls) != 0 {
		t.Fatalf("untrack scope: %+v", r)
	}
}

func TestToolsCommands(t *testing.T) {
	results := map[string]any{
		"tools": []map[string]string{{"id": "smapi", "name": "SMAPI", "executable": "/bin/smapi"}},
	}
	r := invoke(t, results, "tools", "stardew")
	if r.code != 0 || !strings.Contains(r.out, "smapi") {
		t.Fatalf("tools: %+v", r)
	}
	r = invoke(t, results, "tools", "run", "stardew", "Farm", "smapi")
	if r.code != 0 || r.calls[0].method != "tools.run" || r.calls[0].params.Name != "smapi" {
		t.Fatalf("tools run: %+v", r)
	}
}

func TestCompleteOffersVerbsGamesAndProfiles(t *testing.T) {
	results := map[string]any{
		"games":    []control.GameRow{{ID: "stardew"}},
		"profiles": []profile.Profile{{ID: "1", Name: "Spring"}, {ID: "2", Name: "Winter"}},
	}
	if r := invoke(t, results, "__complete", "con"); strings.TrimSpace(r.out) != "conflicts" {
		t.Errorf("verb: %q", r.out)
	}
	if r := invoke(t, results, "__complete", "mods", ""); !strings.Contains(r.out, "disable") || !strings.Contains(r.out, "stardew") {
		t.Errorf("mods position 1 should offer subverbs and games: %q", r.out)
	}
	if r := invoke(t, results, "__complete", "conflicts", "stardew", "w"); strings.TrimSpace(r.out) != "Winter" {
		t.Errorf("profile: %q", r.out)
	}
}

func TestTrashListRestoreAndYes(t *testing.T) {
	deleted := time.Now().Add(-3 * 24 * time.Hour).UTC()
	results := map[string]any{
		"trash.list":    []profile.TrashItem{{ID: "id1", Name: "Old Farm", DeletedAt: deleted, DaysLeft: 27}},
		"trash.restore": profile.Profile{ID: "id1", Name: "Old Farm"},
		"trash.delete":  control.Removed{Mods: []string{"Old Farm"}},
	}
	r := invoke(t, results, "trash", "list")
	if r.code != 0 || !strings.Contains(r.out, "Old Farm") || !strings.Contains(r.out, "days left") {
		t.Fatalf("list: %+v", r)
	}
	if got := r.calls[0]; got.method != "trash.list" || got.params.Game != "stardew" {
		t.Fatalf("list params: %+v", got)
	}
	r = invoke(t, results, "trash", "list", "--game", "stardew")
	if r.calls[0].params.Game != "stardew" {
		t.Fatalf("list --game: %+v", r.calls[0])
	}
	r = invoke(t, map[string]any{"trash.list": []profile.TrashItem{}}, "trash", "list")
	if r.code != 0 || !strings.Contains(r.out, "Trash is empty.") {
		t.Fatalf("empty list: %+v", r)
	}
	r = invoke(t, results, "trash", "restore", "Old Farm")
	if r.code != 0 || !strings.Contains(r.out, "Restored") || r.calls[0].method != "trash.restore" || r.calls[0].params.Profile != "Old Farm" {
		t.Fatalf("restore: %+v", r)
	}
	r = invoke(t, nil, "trash", "delete", "Old Farm")
	if r.code != 2 || len(r.calls) != 0 || !strings.Contains(r.errOut, "--yes") {
		t.Fatalf("delete without yes: %+v", r)
	}
	r = invoke(t, results, "trash", "delete", "Old Farm", "--yes")
	if r.code != 0 || r.calls[0].method != "trash.delete" || !strings.Contains(r.out, "Permanently deleted") {
		t.Fatalf("delete: %+v", r)
	}
	r = invoke(t, nil, "trash", "empty")
	if r.code != 2 || !strings.Contains(r.errOut, "--yes") {
		t.Fatalf("empty without yes: %+v", r)
	}
	r = invoke(t, map[string]any{"trash.empty": nil}, "trash", "empty", "--yes")
	if r.code != 0 || r.calls[0].method != "trash.empty" || !strings.Contains(r.out, "emptied") {
		t.Fatalf("empty: %+v", r)
	}
}

func TestProblemsDismissRestoreAndDismissed(t *testing.T) {
	problemsResult := problems.Result{
		Missing: []problems.Missing{
			{DependentName: "Pack", UniqueID: "Need.Mod", Listed: true},
			{DependentName: "Other", UniqueID: "Core", Reason: "absent"},
		},
		Settings: []problems.SettingHint{{UniqueID: "A.Mod", Name: "Alpha", Field: "Enabled", Current: "false", ForNames: []string{"Beta"}}},
		Dismissed: []problems.DismissedProblem{
			{Token: strings.Join([]string{"listed", "need.mod"}, "\t"), Missing: &problems.Missing{DependentName: "Pack", UniqueID: "Need.Mod", Listed: true}},
		},
	}
	results := map[string]any{
		"problems":           problemsResult,
		"problems.dismissed": problemsResult.Dismissed,
		"problems.dismiss":   nil,
		"problems.restore":   nil,
	}
	r := invoke(t, results, "problems", "stardew", "Farm")
	if r.code != 0 || !strings.Contains(r.out, " 1  missing    Pack") || strings.Contains(r.out, " 1  missing    Other") {
		t.Fatalf("numbered listed missing only: %q", r.out)
	}
	if !strings.Contains(r.out, "missing    Other") {
		t.Fatalf("non-listed missing line: %q", r.out)
	}
	r = invoke(t, results, "problems", "dismissed", "--profile", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "listed\tneed.mod") || r.calls[0].method != "problems.dismissed" {
		t.Fatalf("dismissed: %+v", r)
	}
	r = invoke(t, results, "problems", "dismiss", "1", "--profile", "Farm")
	if r.code != 0 || r.calls[len(r.calls)-1].method != "problems.dismiss" || r.calls[len(r.calls)-1].params.ModID != 1 {
		t.Fatalf("dismiss: %+v", r)
	}
	r = invoke(t, results, "problems", "dismiss", "9", "--profile", "Farm")
	if r.code != 2 || !strings.Contains(r.errOut, "not dismissable") {
		t.Fatalf("dismiss out of range: %+v", r)
	}
	r = invoke(t, results, "problems", "restore", "1", "--profile", "Farm")
	if r.code != 0 || r.calls[len(r.calls)-1].method != "problems.restore" || r.calls[len(r.calls)-1].params.ModID != 1 {
		t.Fatalf("restore index: %+v", r)
	}
	r = invoke(t, results, "problems", "restore", "listed\tneed.mod", "--profile", "Farm")
	if got := r.calls[len(r.calls)-1]; got.method != "problems.restore" || got.params.Name != "listed\tneed.mod" {
		t.Fatalf("restore token: %+v", got)
	}
}

func TestIsTakesVerbsAndBareWordsButNotLinksOrFiles(t *testing.T) {
	for arg, want := range map[string]bool{
		"games": true, "nonsense": true, "nxm://stardewvalley/mods/1/files/2": false,
		"/home/me/farm.mortar": false, "farm.mortar": false, `C:\farm.mortar`: false, "--release-links": false,
	} {
		if got := Is([]string{arg}); got != want {
			t.Errorf("Is(%q) = %v, want %v", arg, got, want)
		}
	}
	if r := invoke(t, nil, "nonsense"); r.code != 2 || !strings.Contains(r.errOut, "unknown command") {
		t.Errorf("unknown verb: %d %q", r.code, r.errOut)
	}
}
