package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/modreport"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/shots"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/storecheck"
)

// verbCase is one thin wrapper: the command line, the answer its method gives, and what the wrapper must send and print.
type verbCase struct {
	name    string
	args    []string
	results map[string]any
	usage   string // the usage error the line must be refused with; empty means it runs
	method  string
	params  func(control.Params) bool
	out     []string
}

func TestVerbWrappersSendTheirArgumentsOrRefuse(t *testing.T) {
	cases := []verbCase{
		{
			name:    "mods group list",
			args:    []string{"mods", "group", "stardew", "Farm", "list"},
			results: map[string]any{"mods.group": []profile.Group{{Name: "Core", Keys: []string{"smapi:A"}}}},
			method:  "mods.group",
			params:  func(p control.Params) bool { return p.Game == "stardew" && p.Profile == "Farm" && p.Sub == "list" },
			out:     []string{"GROUP", "Core", "smapi:A"},
		},
		{
			name:  "mods group create without a name",
			args:  []string{"mods", "group", "stardew", "Farm", "create"},
			usage: "mods group create needs a group name",
		},
		{
			name:  "mods group rename without a new name",
			args:  []string{"mods", "group", "stardew", "Farm", "rename", "Old"},
			usage: "mods group rename needs a group name and a new name",
		},
		{
			name:    "mods group rename",
			args:    []string{"mods", "group", "stardew", "Farm", "rename", "Old", "New"},
			results: map[string]any{"mods.group": profile.Profile{}},
			method:  "mods.group",
			params: func(p control.Params) bool {
				return p.Name == "Old" && p.Value == "New" && p.Sub == "rename"
			},
			out: []string{"Group Old renamed to New."},
		},
		{
			name:  "mods group add without a mod",
			args:  []string{"mods", "group", "stardew", "Farm", "add", "Core"},
			usage: "mods group add needs a group name and a mod id",
		},
		{
			name:    "mods group add sends one id",
			args:    []string{"mods", "group", "stardew", "Farm", "add", "Core", "A.Mod", "B.Mod"},
			results: map[string]any{"mods.group": profile.Profile{}},
			method:  "mods.group",
			params: func(p control.Params) bool {
				return p.Name == "Core" && len(p.IDs) == 1 && p.IDs[0] == "A.Mod"
			},
			out: []string{"Add A.Mod Core."},
		},
		{
			name:    "mods group off",
			args:    []string{"mods", "group", "stardew", "Farm", "off", "Core"},
			results: map[string]any{"mods.group": profile.Profile{}},
			method:  "mods.group",
			params:  func(p control.Params) bool { return p.Name == "Core" && p.Sub == "off" },
			out:     []string{"Group Core off."},
		},
		{
			name:  "mods group unknown subcommand",
			args:  []string{"mods", "group", "stardew", "Farm", "shuffle"},
			usage: "mods group needs list, create, rename, delete, add, remove, on, or off",
		},
		{
			name:    "mods preset list needs no name",
			args:    []string{"mods", "preset", "stardew", "Farm", "A.Mod", "list"},
			results: map[string]any{"mods.preset": []string{"Debug"}},
			method:  "mods.preset",
			params:  func(p control.Params) bool { return p.Sub == "list" && p.Name == "" && p.IDs[0] == "A.Mod" },
			out:     []string{"Debug"},
		},
		{
			name:  "mods preset save needs a name",
			args:  []string{"mods", "preset", "stardew", "Farm", "A.Mod", "save"},
			usage: "mods preset save needs a name",
		},
		{
			name:    "mods preset apply sends its name",
			args:    []string{"mods", "preset", "stardew", "Farm", "A.Mod", "apply", "Debug"},
			results: map[string]any{"mods.preset": []string{}},
			method:  "mods.preset",
			params:  func(p control.Params) bool { return p.Sub == "apply" && p.Name == "Debug" },
		},
		{
			name:  "mods preset unknown subcommand",
			args:  []string{"mods", "preset", "stardew", "Farm", "A.Mod", "drop"},
			usage: "mods preset needs list, save, apply, or delete",
		},
		{
			name:    "profile set joins the value",
			args:    []string{"profile", "set", "stardew", "Farm", "nickname", "My", "Farm"},
			results: map[string]any{"profile.set": profile.Profile{ID: "p1"}},
			method:  "profile.set",
			params: func(p control.Params) bool {
				return p.Key == "nickname" && p.Value == "My Farm" && p.Profile == "Farm"
			},
			out: []string{"p1"},
		},
		{
			name:  "profile set without a value",
			args:  []string{"profile", "set", "stardew", "Farm", "nickname"},
			usage: "profile set needs a value",
		},
		{
			name:  "source tracked needs --missing",
			args:  []string{"source", "tracked", "stardew", "Farm", "--source", "nexus"},
			usage: "source tracked needs --missing",
		},
		{
			name:    "source tracked",
			args:    []string{"source", "tracked", "stardew", "Farm", "--missing", "--source", "nexus"},
			results: map[string]any{"source.trackedMissing": []nexusmods.TrackedMod{{ModID: 7, DomainName: "stardewvalley"}}},
			method:  "source.trackedMissing",
			params:  func(p control.Params) bool { return p.Source == "nexus" && p.Profile == "Farm" },
			out:     []string{"7\tstardewvalley"},
		},
		{
			name:    "mods channel",
			args:    []string{"mods", "channel", "stardew", "Farm", "A.Mod", "optional"},
			results: map[string]any{"mods.channel": []control.ModRow{}},
			method:  "mods.channel",
			params:  func(p control.Params) bool { return p.Value == "optional" && len(p.IDs) == 1 && p.IDs[0] == "A.Mod" },
			out:     []string{"Update channel for A.Mod is optional."},
		},
		{
			name:  "mods channel without a channel",
			args:  []string{"mods", "channel", "stardew", "Farm", "A.Mod"},
			usage: "mods channel needs main, optional, or beta",
		},
		{
			name:    "mods win",
			args:    []string{"mods", "win", "stardew", "Farm", "A.Mod", "B.Mod"},
			results: map[string]any{"mods.win": profile.Profile{}},
			method:  "mods.win",
			params: func(p control.Params) bool {
				return !p.Remove && len(p.IDs) == 2 && p.IDs[0] == "A.Mod" && p.IDs[1] == "B.Mod"
			},
			out: []string{"Made A.Mod win over B.Mod."},
		},
		{
			name:    "mods win --undo",
			args:    []string{"mods", "win", "stardew", "Farm", "A.Mod", "B.Mod", "--undo"},
			results: map[string]any{"mods.win": profile.Profile{}},
			method:  "mods.win",
			params:  func(p control.Params) bool { return p.Remove },
			out:     []string{"Undid A.Mod winning over B.Mod."},
		},
		{
			name:  "mods win without a loser",
			args:  []string{"mods", "win", "stardew", "Farm"},
			usage: "mods win needs a winner and a loser",
		},
		{
			name: "mods compat shows only non-ok rows",
			args: []string{"mods", "compat", "stardew", "Farm"},
			results: map[string]any{"compatibility": []problems.Compat{
				{Name: "Fine", Status: "ok"},
				{Name: "Broken", Status: "broken", Summary: "crashes"},
			}},
			method: "compatibility",
			params: func(p control.Params) bool { return p.Profile == "Farm" },
			out:    []string{"Broken", "crashes"},
		},
		{
			name:    "mods compat with nothing to show",
			args:    []string{"mods", "compat", "stardew", "Farm"},
			results: map[string]any{"compatibility": []problems.Compat{{Name: "Fine", Status: "ok"}}},
			method:  "compatibility",
			out:     []string{"No compatibility issues."},
		},
		{
			name:    "mods report prints its text and URL",
			args:    []string{"mods", "report", "stardew", "Farm", "A.Mod", "--run", "run-1"},
			results: map[string]any{"mods.report": modreport.Result{Text: "the report", URL: "https://example.test/r"}},
			method:  "mods.report",
			params:  func(p control.Params) bool { return p.Run == "run-1" && len(p.IDs) == 1 && p.IDs[0] == "A.Mod" },
			out:     []string{"the report", "https://example.test/r"},
		},
		{
			name:  "mods report without a mod",
			args:  []string{"mods", "report", "stardew", "Farm"},
			usage: "mods report needs a mod id (SMAPI id)",
		},
		{
			name: "mods by-author lists each profile",
			args: []string{"mods", "by-author", "stardew", "Some", "Author"},
			results: map[string]any{"mods.by-author": []profile.AuthorMod{{
				ID: mod.ID("smapi:A"), Name: "Alpha",
				Profiles: []profile.ModInProfile{{ProfileName: "Farm", Version: "1.2", Enabled: false}},
			}}},
			method: "mods.by-author",
			params: func(p control.Params) bool { return p.Query == "Some Author" },
			out:    []string{"Alpha (smapi:A)", "Farm · 1.2 · disabled"},
		},
		{
			name:    "mods by-author with no matches",
			args:    []string{"mods", "by-author", "stardew", "Nobody"},
			results: map[string]any{"mods.by-author": []profile.AuthorMod{}},
			method:  "mods.by-author",
			out:     []string{"No mods for that author."},
		},
		{
			name:    "sweep with no version change",
			args:    []string{"sweep", "stardew"},
			results: map[string]any{"sweep": launchsvc.SweepReport{GameName: "Stardew Valley"}},
			method:  "sweep",
			params:  func(p control.Params) bool { return p.Game == "stardew" && p.Install == "" },
			out:     []string{"No game or SMAPI version change."},
		},
		{
			name:    "sweep --install",
			args:    []string{"sweep", "stardew", "--install", "steam"},
			results: map[string]any{"sweep": launchsvc.SweepReport{Triggered: true, GameName: "Stardew Valley", GameVersion: "1.6", LoaderVersion: "4.0"}},
			method:  "sweep",
			params:  func(p control.Params) bool { return p.Install == "steam" },
			out:     []string{"Stardew Valley 1.6 / SMAPI 4.0: nothing to fix."},
		},
		{
			name: "sweep lists broken mods and missing deps",
			args: []string{"sweep", "stardew"},
			results: map[string]any{"sweep": launchsvc.SweepReport{Triggered: true, Profiles: []launchsvc.ProfileSweep{{
				Name:        "Farm",
				MissingDeps: 2,
				Broken: []launchsvc.SweepMod{
					{Name: "Alpha", Status: "broken", Fix: "update"},
					{Name: "Beta", Status: "obsolete", Fix: "off"},
					{Name: "Gamma", Status: "broken", Fix: "none"},
				},
			}}}},
			method: "sweep",
			out:    []string{"PROFILE", "missing deps", "2", "Alpha", "Update", "Turn off", "No fix yet"},
		},
		{
			name:    "who joins its query",
			args:    []string{"who", "stardew", "Farm", "Data", "Objects"},
			results: map[string]any{"who": cp.WhoChangesPage{Targets: assetTargets()}},
			method:  "who",
			params:  func(p control.Params) bool { return p.Query == "Data Objects" && p.Profile == "Farm" },
			out:     []string{"Data/Objects k", "Alpha* Edit", "B Replace", "Alpha"},
		},
		{
			name:  "who without a query",
			args:  []string{"who", "stardew", "Farm"},
			usage: "who needs a query",
		},
		{
			name:    "conflicts map with no assets",
			args:    []string{"conflicts", "map", "stardew", "Farm", "--filter", "Data"},
			results: map[string]any{"conflicts.map": cp.AssetMapPage{}},
			method:  "conflicts.map",
			params:  func(p control.Params) bool { return p.Query == "Data" },
			out:     []string{"No assets."},
		},
		{
			name:    "conflicts map",
			args:    []string{"conflicts", "map", "stardew", "Farm"},
			results: map[string]any{"conflicts.map": cp.AssetMapPage{Targets: assetTargets()}},
			method:  "conflicts.map",
			out:     []string{"Data/Objects k", "Alpha* Edit"},
		},
		{
			name:    "store repair prints its status",
			args:    []string{"store", "repair", "stardew", "Farm", "key-1"},
			results: map[string]any{"store.repair": storecheck.RepairResult{Status: "repaired"}},
			method:  "store.repair",
			params:  func(p control.Params) bool { return p.Key == "key-1" && p.Profile == "Farm" },
			out:     []string{"repaired"},
		},
		{
			name:  "store repair without a key",
			args:  []string{"store", "repair", "stardew", "Farm"},
			usage: "store repair needs a store key",
		},
		{
			name: "store report",
			args: []string{"store", "report", "--game", "stardew"},
			results: map[string]any{"store.report": store.Report{"stardew": {Unused: []store.Item{
				{Key: "key-1", Name: "Alpha", Version: "1.0", Size: 2048},
			}}}},
			method: "store.report",
			params: func(p control.Params) bool { return p.Game == "stardew" },
			out:    []string{"GAME", "key-1", "Alpha", "unused"},
		},
		{
			name:    "store report with nothing to clean",
			args:    []string{"store", "report"},
			results: map[string]any{"store.report": store.Report{}},
			method:  "store.report",
			params:  func(p control.Params) bool { return p.Game == "" },
			out:     []string{"No unused or duplicate store items."},
		},
		{
			name:    "logs fixes",
			args:    []string{"logs", "fixes", "stardew", "Farm"},
			results: fixes(),
			method:  "logs.fixes",
			params:  func(p control.Params) bool { return p.Game == "stardew" && p.Profile == "Farm" && p.Run == "" },
			out:     []string{"tooOld  update  Alpha (A.Mod) needs ContentPatcher", "no longer loads"},
		},
		{
			name:    "logs fixes with none recognised",
			args:    []string{"logs", "fixes", "stardew", "Farm", "--run", "run-2"},
			results: map[string]any{"logs.fixes": []launch.SMAPIProblem{}},
			method:  "logs.fixes",
			params:  func(p control.Params) bool { return p.Run == "run-2" },
			out:     []string{"No recognised problems."},
		},
		{
			name:    "launchers",
			args:    []string{"launchers"},
			results: map[string]any{"launchers": []game.StoreApp{{ID: "steam", Name: "Steam", Found: true, Roots: []string{"/steam"}}}},
			method:  "launchers",
			params:  func(p control.Params) bool { return p.Game == "" && p.Name == "" && p.Path == "" },
			out:     []string{"steam", "Steam", "yes", "/steam"},
		},
		{
			name:    "launchers add",
			args:    []string{"launchers", "add", "steam", "games"},
			results: map[string]any{"launchers.add": []game.StoreApp{}},
			method:  "launchers.add",
			params:  func(p control.Params) bool { return p.Name == "steam" && strings.HasSuffix(p.Path, "/games") },
		},
		{
			name:  "launchers add without a folder",
			args:  []string{"launchers", "add", "steam"},
			usage: "launchers add needs a folder",
		},
		{
			name:  "launchers unknown subcommand",
			args:  []string{"launchers", "reset"},
			usage: "unknown launchers command reset",
		},
		{
			name:    "screenshots",
			args:    []string{"screenshots", "stardew"},
			results: map[string]any{"screenshots": shots.Gallery{Shots: []shots.Shot{{Source: "steam", Name: "a.png", Size: 10}}}},
			method:  "screenshots",
			params:  func(p control.Params) bool { return p.Game == "stardew" },
			out:     []string{"SOURCE", "steam", "a.png", "10"},
		},
		{
			name:  "screenshots without a game",
			args:  []string{"screenshots"},
			usage: "screenshots needs a game",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := invoke(t, tc.results, tc.args...)
			if tc.usage != "" {
				if r.code != 2 || !strings.Contains(r.errOut, tc.usage) || len(r.calls) != 0 {
					t.Fatalf("%v: code %d, calls %v, stderr %q", tc.args, r.code, r.calls, r.errOut)
				}
				return
			}
			if r.code != 0 {
				t.Fatalf("%v: code %d, stderr %q", tc.args, r.code, r.errOut)
			}
			if len(r.calls) != 1 || r.calls[0].method != tc.method {
				t.Fatalf("%v: calls %v, want one %s", tc.args, r.calls, tc.method)
			}
			if tc.params != nil && !tc.params(r.calls[0].params) {
				t.Fatalf("%v: params %+v", tc.args, r.calls[0].params)
			}
			for _, want := range tc.out {
				if !strings.Contains(r.out, want) {
					t.Errorf("%v: output %q lacks %q", tc.args, r.out, want)
				}
			}
		})
	}
}

func assetTargets() []cp.AssetTarget {
	return []cp.AssetTarget{{
		Target: "Data/Objects", Key: "k", Winner: cp.ModID("smapi:Alpha"),
		Mods: []cp.AssetTouch{
			{ModName: "Alpha", Action: "Edit", Winner: true},
			{ModID: cp.ModID("smapi:B"), Action: "Replace"},
		},
	}}
}

func fixes() map[string]any {
	return map[string]any{"logs.fixes": []launch.SMAPIProblem{{
		Kind: launch.SMAPIProblemTooOld, Fix: launch.SMAPIFixUpdate,
		ModName: "Alpha", ModID: "A.Mod", Dependency: "ContentPatcher", Detail: "no longer loads",
	}}}
}

func TestRelativeDeletedNamesSingleUnits(t *testing.T) {
	for _, tc := range []struct {
		ago  time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{time.Minute, "1 minute ago"},
		{2 * time.Minute, "2 minutes ago"},
		{59 * time.Minute, "59 minutes ago"},
		{time.Hour, "1 hour ago"},
		{2 * time.Hour, "2 hours ago"},
		{24 * time.Hour, "1 day ago"},
		{48 * time.Hour, "2 days ago"},
	} {
		if got := relativeDeleted(time.Now().Add(-tc.ago)); got != tc.want {
			t.Errorf("%s ago = %q, want %q", tc.ago, got, tc.want)
		}
	}
}
