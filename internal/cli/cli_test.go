package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/controlwire"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/savessvc"
	"github.com/Rethunk-AI/mortar/internal/sharesvc"
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
	if r.code != 0 || !strings.Contains(r.out, "NAME") || !strings.Contains(r.out, "disabled, pinned") || strings.Contains(r.out, "UNIQUEID") {
		t.Fatalf("mods table: %q", r.out)
	}
	r = invoke(t, results, "mods", "stardew", "abc", "-v")
	if r.code != 0 || !strings.Contains(r.out, "MOD ID") || !strings.Contains(r.out, "B.Mod") {
		t.Fatalf("mods table verbose: %q", r.out)
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
	r = invoke(t, map[string]any{"launch": nil}, "launch", "stardew", "abc", "--force")
	if r.code != 0 || len(r.calls) != 1 || r.calls[0].method != "launch" || !r.calls[0].params.Force {
		t.Fatalf("--force: %+v", r)
	}
}

func TestSettingsGetSet(t *testing.T) {
	results := map[string]any{
		"settings.get": [][2]string{{"onPlay", "stay"}, {"runsKept", "20"}},
		"settings.set": nil,
	}
	r := invoke(t, results, "settings", "get")
	if r.code != 0 || r.calls[0].method != "settings.get" || !strings.Contains(r.out, "onPlay") {
		t.Fatalf("get: %+v", r)
	}
	r = invoke(t, results, "settings", "get", "onPlay")
	if r.calls[0].params.Key != "onPlay" {
		t.Fatalf("get key: %+v", r.calls[0])
	}
	r = invoke(t, results, "settings", "set", "onPlay", "hide")
	if r.code != 0 || r.calls[0].method != "settings.set" || r.calls[0].params.Key != "onPlay" || r.calls[0].params.Value != "hide" {
		t.Fatalf("set %+v", r)
	}
	r = invoke(t, results, "settings", "get", "--game", "stardew", "smapiBuilds")
	if r.code != 0 || r.calls[0].params.Game != "stardew" || r.calls[0].params.Key != "smapiBuilds" {
		t.Fatalf("get --game %+v", r)
	}
	r = invoke(t, results, "settings", "set", "--game", "stardew", "smapiBuilds", "include")
	if r.code != 0 || r.calls[0].params.Game != "stardew" || r.calls[0].params.Key != "smapiBuilds" || r.calls[0].params.Value != "include" {
		t.Fatalf("set: %+v", r)
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
	if r.code != 0 || !strings.Contains(r.out, "Added Alpha") || strings.Contains(r.out, "event-1") || strings.Contains(r.out, "KIND") {
		t.Fatalf("history: %+v", r)
	}
	r = invoke(t, results, "profile", "history", "stardew", "Farm", "--json")
	if r.code != 0 || !strings.Contains(r.out, `"id": "event-1"`) {
		t.Fatalf("history json: %q", r.out)
	}
	r = invoke(t, results, "profile", "revert", "stardew", "Farm", "event-1")
	if r.code != 0 || r.calls[0].method != "profile.revert" || r.calls[0].params.Name != "event-1" {
		t.Fatalf("revert: %+v", r)
	}
}

func TestProfileRepair(t *testing.T) {
	results := map[string]any{
		"profile.repair": profile.Profile{ID: "0123456789abcdef", Name: "Farm"},
	}
	r := invoke(t, results, "profile", "repair", "stardew", "0123456789abcdef", "--json")
	if r.code != 0 || r.calls[0].method != "profile.repair" || r.calls[0].params.Profile != "0123456789abcdef" {
		t.Fatalf("repair: %+v", r)
	}
	if !strings.Contains(r.out, `"id": "0123456789abcdef"`) {
		t.Fatalf("repair json: %q", r.out)
	}
}

func TestHistoryAll(t *testing.T) {
	results := map[string]any{
		"history.all": []profile.RecentEvent{
			{ProfileID: "p1", ProfileName: "Farm", ID: "event-2", Kind: "added", Label: "Added Beta"},
		},
	}
	r := invoke(t, results, "history", "stardew", "--all")
	if r.code != 0 || !strings.Contains(r.out, "Added Beta") || !strings.Contains(r.out, "Farm") || strings.Contains(r.out, "event-2") || strings.Contains(r.out, "KIND") {
		t.Fatalf("history --all: %+v", r)
	}
	if got := r.calls[0]; got.method != "history.all" || got.params.Game != "stardew" {
		t.Fatalf("history --all params: %+v", got)
	}
	r = invoke(t, results, "history", "--all", "--json", "stardew")
	if r.code != 0 || !strings.Contains(r.out, `"profileName": "Farm"`) {
		t.Fatalf("history --all --json: %+v", r)
	}
	if r := invoke(t, results, "history", "stardew"); r.code != 2 {
		t.Fatalf("history without --all: %+v", r)
	}
}

func TestBackupsCreate(t *testing.T) {
	results := map[string]any{"backups.create": nil}
	r := invoke(t, results, "backups", "create", "Farm_1")
	if r.code != 0 || r.calls[0].method != "backups.create" || r.calls[0].params.Name != "Farm_1" {
		t.Fatalf("create: %+v", r)
	}
	if !strings.Contains(r.out, "Backed up Farm_1") {
		t.Fatalf("create out: %q", r.out)
	}
	r = invoke(t, results, "backups", "create", "Farm_1", "--json")
	if r.code != 0 || r.calls[0].method != "backups.create" {
		t.Fatalf("create json: %+v", r)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(r.out), &body); err != nil || body["save"] != "Farm_1" {
		t.Fatalf("create json: %v %q", err, r.out)
	}
}

func TestBackupsKeepAndUnkeep(t *testing.T) {
	results := map[string]any{
		"backups.keep":   nil,
		"backups.unkeep": nil,
	}
	r := invoke(t, results, "backups", "keep", "2026-01-01.zip")
	if r.code != 0 || r.calls[0].method != "backups.keep" || r.calls[0].params.Name != "2026-01-01.zip" {
		t.Fatalf("keep: %+v", r)
	}
	if !strings.Contains(r.out, "Kept") {
		t.Fatalf("keep out: %q", r.out)
	}
	r = invoke(t, results, "backups", "unkeep", "2026-01-01.zip", "--json")
	if r.code != 0 || r.calls[0].method != "backups.unkeep" {
		t.Fatalf("unkeep: %+v", r)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(r.out), &body); err != nil || body["pinned"] != false {
		t.Fatalf("unkeep json: %v %q", err, r.out)
	}
}

func TestHelpListsProfileCollection(t *testing.T) {
	r := invoke(t, nil, "help")
	if r.code != 0 || !strings.Contains(r.out, "profile collection") {
		t.Fatalf("help: %d %q", r.code, r.out)
	}
}

func TestProfileCollectionSendsStatusRequest(t *testing.T) {
	st := sharesvc.CollectionStatus{Linked: true, Name: "Cozy Farm", URL: "https://example", Revision: 2, Latest: 4}
	r := invoke(t, map[string]any{"profile.collection": st}, "profile", "collection", "stardew", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "revision 2") || !strings.Contains(r.out, "latest 4") {
		t.Fatalf("collection: %+v", r)
	}
	if got := r.calls[0]; got.method != "profile.collection" || got.params.All {
		t.Fatalf("collection params: %+v", got)
	}
	r = invoke(t, map[string]any{"profile.collection": sharesvc.Result{Queued: 3}}, "profile", "collection", "stardew", "Farm", "--update")
	if r.code != 0 || !strings.Contains(r.out, "Queued 3") || !r.calls[0].params.All {
		t.Fatalf("collection --update: %+v", r)
	}
}

func TestProfilesMarksDamaged(t *testing.T) {
	results := map[string]any{
		"profiles": []profile.Profile{
			{ID: "1", Name: "Spring"},
			{ID: "0123456789abcdef", Error: "read profile: unexpected EOF"},
		},
	}
	r := invoke(t, results, "profiles", "stardew")
	if r.code != 0 || !strings.Contains(r.out, "Could not read this profile") || strings.Contains(r.out, "unexpected EOF") {
		t.Fatalf("profiles: %q", r.out)
	}
	r = invoke(t, results, "profiles", "stardew", "--json")
	if r.code != 0 || !strings.Contains(r.out, "0123456789abcdef") || !strings.Contains(r.out, "unexpected EOF") {
		t.Fatalf("profiles json: %q", r.out)
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
	if !strings.Contains(r.out, "2 problems: 1 missing") {
		t.Fatalf("counted summary: %q", r.out)
	}
	r = invoke(t, results, "problems", "dismissed", "--profile", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "Pack needs Need.Mod") || strings.Contains(r.out, "listed\tneed.mod") || r.calls[0].method != "problems.dismissed" {
		t.Fatalf("dismissed: %+v", r)
	}
	r = invoke(t, results, "problems", "dismissed", "--profile", "Farm", "--json")
	if r.code != 0 || !strings.Contains(r.out, `"token"`) || !strings.Contains(r.out, "need.mod") {
		t.Fatalf("dismissed json: %q", r.out)
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

func TestCacheAndDataUsageByMod(t *testing.T) {
	cacheSize := map[string]any{"path": "/data/cache", "size": 12}
	modRow := map[string]any{
		"game": "stardew", "key": "nexus-1-1", "name": "Alpha", "size": 100, "profiles": 1, "profileSize": 80, "lastUsed": "2026-01-02T03:04:05Z",
	}
	usageByMod := map[string]any{
		"total": 100,
		"items": []map[string]any{modRow},
	}
	results := map[string]any{
		"cache.size":      cacheSize,
		"cache.clear":     nil,
		"data.usageByMod": usageByMod,
	}
	r := invoke(t, results, "cache", "size")
	if r.code != 0 || r.calls[0].method != "cache.size" || !strings.Contains(r.out, "/data/cache") || !strings.Contains(r.out, "12 B") {
		t.Fatalf("cache size: %+v", r)
	}
	r = invoke(t, results, "cache", "clear")
	if r.code != 0 || r.calls[0].method != "cache.clear" {
		t.Fatalf("cache clear: %+v", r)
	}
	r = invoke(t, results, "data", "usage")
	if r.code != 2 {
		t.Fatalf("data usage without --by-mod: %+v", r)
	}
	r = invoke(t, results, "data", "usage", "--by-mod")
	if r.code != 0 || r.calls[0].method != "data.usageByMod" || !strings.Contains(r.out, "nexus-1-1") {
		t.Fatalf("data usage --by-mod: %+v", r)
	}
	r = invoke(t, results, "data", "usage", "--by-mod", "--json")
	if r.code != 0 || !strings.Contains(r.out, `"total"`) {
		t.Fatalf("data usage json: %+v", r)
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

func TestJSONErrorsUseStructuredExitCodes(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run("9.9.9", func(string, control.Params, any, time.Duration) error {
		return controlwire.ErrNotRunning
	}, []string{"games", "--json"}, &out, &errOut); code != 3 {
		t.Fatalf("not running code = %d", code)
	}
	var failure map[string]any
	if err := json.Unmarshal(errOut.Bytes(), &failure); err != nil || failure["code"] != float64(3) {
		t.Fatalf("not running JSON = %q", errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := run("9.9.9", fake(nil, new([]call)), []string{"profile", "--json"}, &out, &errOut); code != 2 {
		t.Fatalf("usage code = %d", code)
	}
	if err := json.Unmarshal(errOut.Bytes(), &failure); err != nil || failure["code"] != float64(2) {
		t.Fatalf("usage JSON = %q", errOut.String())
	}
}

func TestPlayCheck(t *testing.T) {
	groups := []control.PlayIssueGroup{
		{Kind: "missing", Count: 1, Names: []string{"Pathoschild.ContentPatcher"}},
		{Kind: "conflicts", Count: 2, Names: []string{"A, B"}},
	}
	r := invoke(t, map[string]any{"play.check": groups}, "play", "stardew", "Farm", "--check")
	if r.code != 3 || r.calls[0].method != "play.check" || !strings.Contains(r.out, "missing (1):") || !strings.Contains(r.out, "conflicts (2):") {
		t.Fatalf("play --check issues: %+v", r)
	}
	r = invoke(t, map[string]any{"play.check": []control.PlayIssueGroup{}}, "play", "stardew", "Farm", "--check")
	if r.code != 0 || !strings.Contains(r.out, "Ready to play.") {
		t.Fatalf("play --check clean: %+v", r)
	}
	r = invoke(t, map[string]any{"play.check": []control.PlayIssueGroup{}}, "play", "stardew", "Farm", "--check", "--json")
	if r.code != 0 || !strings.Contains(r.out, "[") {
		t.Fatalf("play --check json: %+v", r)
	}
	if r := invoke(t, nil, "play", "stardew", "Farm"); r.code != 2 {
		t.Fatalf("play without --check: %+v", r)
	}
}

func TestProfileListAndProblemsText(t *testing.T) {
	results := map[string]any{
		"mods": []control.ModRow{
			{UniqueID: "A.Mod", Name: "Alpha", Version: "1.0", Enabled: true, Source: "nexus:1915/2"},
			{UniqueID: "B.Mod", Name: "Beta", Version: "2.0", Enabled: false, Source: "local"},
		},
		"problems": problems.Result{
			Missing:        []problems.Missing{{UniqueID: "Need.Mod", Optional: false}, {UniqueID: "Opt", Optional: true}},
			AssetConflicts: []problems.AssetConflict{{Kind: "load", Target: "x", Names: []string{"A"}}, {Kind: "edit", Target: "y", Cosmetic: true}},
		},
	}
	r := invoke(t, results, "profile", "list", "stardew", "Farm", "--format", "text")
	if r.code != 0 || !strings.Contains(r.out, "Alpha") || !strings.Contains(r.out, "1.0") || strings.Contains(r.out, "Beta") {
		t.Fatalf("profile list text: %+v", r)
	}
	if !strings.Contains(r.out, "https://www.nexusmods.com/stardewvalley/mods/1915") {
		t.Fatalf("profile list nexus: %q", r.out)
	}
	r = invoke(t, results, "profile", "list", "stardew", "Farm", "--format", "md")
	if r.code != 0 || !strings.Contains(r.out, "- Alpha 1.0 https://") {
		t.Fatalf("profile list md: %q", r.out)
	}
	if r := invoke(t, results, "profile", "list", "stardew", "Farm"); r.code != 2 {
		t.Fatalf("profile list needs format: %+v", r)
	}
	r = invoke(t, results, "problems", "stardew", "Farm", "--format", "text")
	if r.code != 0 || !strings.Contains(r.out, "missing: 1") || !strings.Contains(r.out, "conflicts: 1") || !strings.Contains(r.out, "harmless: 2") {
		t.Fatalf("problems text: %q", r.out)
	}
}

func TestSettingsExportImportReset(t *testing.T) {
	results := map[string]any{
		"settings.export": map[string]string{"path": "/tmp/mortar-settings.json"},
		"settings.import": nil,
		"settings.reset":  nil,
	}
	r := invoke(t, results, "settings", "export", "/tmp/mortar-settings.json")
	if r.code != 0 || r.calls[0].method != "settings.export" || !strings.Contains(r.out, "Wrote") {
		t.Fatalf("export: %+v", r)
	}
	r = invoke(t, results, "settings", "import", "/tmp/mortar-settings.json")
	if r.code != 0 || r.calls[0].method != "settings.import" || !strings.Contains(r.out, "Imported") {
		t.Fatalf("import: %+v", r)
	}
	r = invoke(t, results, "settings", "reset", "onPlay")
	if r.code != 0 || r.calls[0].method != "settings.reset" || r.calls[0].params.Key != "onPlay" {
		t.Fatalf("reset key: %+v", r)
	}
	r = invoke(t, results, "settings", "reset", "--game", "stardew")
	if r.code != 0 || r.calls[0].params.Game != "stardew" || r.calls[0].params.Key != "" {
		t.Fatalf("reset all: %+v", r)
	}
}

func TestModsTagCategoryNoteSkipVersion(t *testing.T) {
	rows := []control.ModRow{{UniqueID: "A.Mod", Name: "Alpha", Version: "1.0", Enabled: true}}
	results := map[string]any{
		"mods.tag":          rows,
		"mods.untag":        rows,
		"mods.category":     rows,
		"mods.note":         rows,
		"mods.skip-version": rows,
	}
	r := invoke(t, results, "mods", "tag", "stardew", "Farm", "A.Mod", "qol")
	if r.code != 0 || r.calls[0].method != "mods.tag" || r.calls[0].params.Value != "qol" || len(r.calls[0].params.UniqueIDs) != 1 {
		t.Fatalf("tag: %+v", r)
	}
	r = invoke(t, results, "mods", "untag", "stardew", "Farm", "A.Mod", "qol")
	if r.code != 0 || r.calls[0].method != "mods.untag" {
		t.Fatalf("untag: %+v", r)
	}
	r = invoke(t, results, "mods", "category", "stardew", "Farm", "A.Mod", "Crops")
	if r.code != 0 || r.calls[0].params.Value != "Crops" {
		t.Fatalf("category: %+v", r)
	}
	r = invoke(t, results, "mods", "note", "stardew", "Farm", "A.Mod", "keep")
	if r.code != 0 || r.calls[0].method != "mods.note" || r.calls[0].params.Value != "keep" {
		t.Fatalf("note: %+v", r)
	}
	r = invoke(t, results, "mods", "skip-version", "stardew", "Farm", "A.Mod", "2.0.0")
	if r.code != 0 || r.calls[0].method != "mods.skip-version" || r.calls[0].params.Value != "2.0.0" {
		t.Fatalf("skip-version: %+v", r)
	}
	r = invoke(t, results, "mods", "skip-version", "stardew", "Farm", "A.Mod")
	if r.code != 0 || r.calls[0].params.Value != "" || len(r.calls[0].params.UniqueIDs) != 1 {
		t.Fatalf("skip-version clear: %+v", r)
	}
}

func TestQueueRetrySkipAndHumanStates(t *testing.T) {
	st := map[string]any{
		"items": []map[string]any{
			{"id": "1", "name": "Alpha", "version": "1", "state": "queued", "error": ""},
			{"id": "2", "name": "Beta", "version": "1", "state": "failed", "error": "[network] timeout"},
			{"id": "3", "name": "Gamma", "version": "1", "state": "waiting-click", "error": ""},
		},
	}
	results := map[string]any{"queue": st, "queue.retry": st, "queue.skip": st}
	r := invoke(t, results, "queue")
	if r.code != 0 || !strings.Contains(r.out, "waiting") || !strings.Contains(r.out, "failed") || !strings.Contains(r.out, "needs a click") {
		t.Fatalf("queue table: %q", r.out)
	}
	if !strings.Contains(r.out, "A network request failed.") {
		t.Fatalf("queue error sentence: %q", r.out)
	}
	r = invoke(t, results, "queue", "retry")
	if r.code != 0 || !strings.Contains(r.out, "Retried 1 failed downloads.") {
		t.Fatalf("retry: %q", r.out)
	}
	r = invoke(t, results, "queue", "skip")
	if r.code != 0 || !strings.Contains(r.out, "Skipped 2 downloads.") {
		t.Fatalf("skip: %q", r.out)
	}
}

func TestGamesProfilesRunsSavesAndEnableHuman(t *testing.T) {
	started := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)
	played := time.Now().Add(-3 * time.Hour).UnixMilli()
	results := map[string]any{
		"games": []control.GameRow{{
			ID: "stardew", Name: "Stardew Valley", Available: true, Configured: false, Profiles: 1, Store: "steam", InstallDir: "/games/Stardew Valley",
		}},
		"profiles": []profile.Profile{
			{ID: "deadbeef", Name: "Broken", Error: "open profile.json: unexpected EOF"},
			{ID: "abc", Name: "Farm"},
		},
		"mods.enable": profile.EnableResult{AlsoEnabled: []string{"Pathoschild.ContentPatcher"}},
		"mods": []control.ModRow{
			{UniqueID: "Pathoschild.ContentPatcher", Name: "Content Patcher", Enabled: true},
			{UniqueID: "A.Mod", Name: "Alpha", Enabled: true},
		},
		"runs": []launchsvc.Run{{
			ID: "run-9", Started: started, DurationMs: 1000, Outcome: "crashed", Errors: 1,
			Cause: &launchsvc.Cause{ModName: "Alpha", Detail: "[damaged] log truncated"},
		}},
		"saves": []savessvc.Fit{{
			Folder: "Farm_1", Farm: "Green Acres", Farmer: "Sam", Season: 0, Day: 1, Year: 2,
			LastProfileID: "abc", LastProfileAt: played,
		}},
	}
	r := invoke(t, results, "games")
	if r.code != 0 || !strings.Contains(r.out, "GAME") || !strings.Contains(r.out, "stardew") || !strings.Contains(r.out, "Supported") || !strings.Contains(r.out, "Not set up") {
		t.Fatalf("games: %q", r.out)
	}
	if strings.Contains(r.out, "ID\t") || strings.Contains(r.out, "\tyes") || strings.Contains(r.out, "\tno") {
		t.Fatalf("games internals: %q", r.out)
	}
	r = invoke(t, results, "games", "--json")
	if r.code != 0 || !strings.Contains(r.out, `"id": "stardew"`) {
		t.Fatalf("games json: %q", r.out)
	}
	r = invoke(t, results, "profiles", "stardew")
	if r.code != 0 || !strings.Contains(r.out, "Could not read this profile") || strings.Contains(r.out, "unexpected EOF") {
		t.Fatalf("profiles: %q", r.out)
	}
	r = invoke(t, results, "profiles", "stardew", "--json")
	if r.code != 0 || !strings.Contains(r.out, "unexpected EOF") {
		t.Fatalf("profiles json: %q", r.out)
	}
	r = invoke(t, results, "mods", "enable", "stardew", "Farm", "A.Mod")
	if r.code != 0 || !strings.Contains(r.out, "Also enabled, as required: Content Patcher.") || strings.Contains(r.out, "Pathoschild.ContentPatcher") {
		t.Fatalf("enable: %q", r.out)
	}
	r = invoke(t, results, "mods", "enable", "stardew", "Farm", "A.Mod", "--json")
	if r.code != 0 || !strings.Contains(r.out, "Pathoschild.ContentPatcher") {
		t.Fatalf("enable json: %q", r.out)
	}
	r = invoke(t, results, "runs", "stardew", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "Crashed") || !strings.Contains(r.out, "minutes ago") || !strings.Contains(r.out, "Alpha: That data could not be read.") {
		t.Fatalf("runs: %q", r.out)
	}
	if strings.Contains(r.out, "run-9") || strings.Contains(r.out, "crashed") || strings.Contains(r.out, "log truncated") {
		t.Fatalf("runs internals: %q", r.out)
	}
	r = invoke(t, results, "runs", "stardew", "Farm", "--json")
	if r.code != 0 || !strings.Contains(r.out, `"id": "run-9"`) || !strings.Contains(r.out, "log truncated") {
		t.Fatalf("runs json: %q", r.out)
	}
	r = invoke(t, results, "saves", "stardew", "Farm")
	if r.code != 0 || !strings.HasPrefix(strings.TrimSpace(r.out), "FARM") || !strings.Contains(r.out, "Green Acres") || !strings.Contains(r.out, "Farm ·") || !strings.Contains(r.out, "hours ago") {
		t.Fatalf("saves: %q", r.out)
	}
	if strings.HasPrefix(strings.TrimSpace(r.out), "FOLDER") {
		t.Fatalf("saves folder-first: %q", r.out)
	}
}
